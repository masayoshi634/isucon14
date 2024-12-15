package main

import (
	"context"
	crand "crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os/exec"
	"runtime"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jmoiron/sqlx"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

var db *sqlx.DB

func main() {
	tp, _ := initTracer(context.Background())
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			panic(err)
		}
	}()

	mux := setup()
	slog.Info("Listening on :8080")
	http.ListenAndServe(":8080", mux)
}

/*
	func setup() http.Handler {
		host := os.Getenv("ISUCON_DB_HOST")
		if host == "" {
			host = "127.0.0.1"
		}
		port := os.Getenv("ISUCON_DB_PORT")
		if port == "" {
			port = "3306"
		}
		_, err := strconv.Atoi(port)
		if err != nil {
			panic(fmt.Sprintf("failed to convert DB port number from ISUCON_DB_PORT environment variable into int: %v", err))
		}
		user := os.Getenv("ISUCON_DB_USER")
		if user == "" {
			user = "isucon"
		}
		password := os.Getenv("ISUCON_DB_PASSWORD")
		if password == "" {
			password = "isucon"
		}
		dbname := os.Getenv("ISUCON_DB_NAME")
		if dbname == "" {
			dbname = "isuride"
		}

		dbConfig := mysql.NewConfig()
		dbConfig.User = user
		dbConfig.Passwd = password
		dbConfig.Addr = net.JoinHostPort(host, port)
		dbConfig.Net = "tcp"
		dbConfig.DBName = dbname
		dbConfig.ParseTime = true
*/
func setup() http.Handler {
	_db, err := GetDB()
	if err != nil {
		panic(err)
	}
	db = _db

	mux := chi.NewRouter()
	mux.Use(middleware.Logger)
	mux.Use(middleware.Recoverer)
	serverName := "isuride"
	mi := otelhttp.NewMiddleware(serverName)
	mux.Use(mi)

	mux.HandleFunc("POST /api/initialize", postInitialize)

	// app handlers
	{
		mux.HandleFunc("POST /api/app/users", appPostUsers)

		authedMux := mux.With(appAuthMiddleware)
		authedMux.HandleFunc("POST /api/app/payment-methods", appPostPaymentMethods)
		authedMux.HandleFunc("GET /api/app/rides", appGetRides)
		authedMux.HandleFunc("POST /api/app/rides", appPostRides)
		authedMux.HandleFunc("POST /api/app/rides/estimated-fare", appPostRidesEstimatedFare)
		authedMux.HandleFunc("POST /api/app/rides/{ride_id}/evaluation", appPostRideEvaluatation)
		authedMux.HandleFunc("GET /api/app/notification", appGetNotification)
		authedMux.HandleFunc("GET /api/app/nearby-chairs", appGetNearbyChairs)
	}

	// owner handlers
	{
		mux.HandleFunc("POST /api/owner/owners", ownerPostOwners)

		authedMux := mux.With(ownerAuthMiddleware)
		authedMux.HandleFunc("GET /api/owner/sales", ownerGetSales)
		authedMux.HandleFunc("GET /api/owner/chairs", ownerGetChairs)
	}

	// chair handlers
	{
		mux.HandleFunc("POST /api/chair/chairs", chairPostChairs)

		authedMux := mux.With(chairAuthMiddleware)
		authedMux.HandleFunc("POST /api/chair/activity", chairPostActivity)
		authedMux.HandleFunc("POST /api/chair/coordinate", chairPostCoordinate)
		authedMux.HandleFunc("GET /api/chair/notification", chairGetNotification)
		authedMux.HandleFunc("POST /api/chair/rides/{ride_id}/status", chairPostRideStatus)
	}

	// internal handlers
	{
		mux.HandleFunc("GET /api/internal/matching", internalGetMatching)
	}

	return mux
}

type postInitializeRequest struct {
	PaymentServer string `json:"payment_server"`
}

type postInitializeResponse struct {
	Language string `json:"language"`
}

func postInitialize(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	req := &postInitializeRequest{}
	if err := bindJSON(r, req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	if out, err := exec.Command("../sql/pg/pg_init.sh").CombinedOutput(); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("failed to initialize: %s: %w", string(out), err))
		return
	}

	if _, err := db.ExecContext(ctx, "UPDATE settings SET value = ? WHERE name = 'payment_gateway_url'", req.PaymentServer); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	if err := rdb.FlushAll(ctx).Err(); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	if err := initializeChairsTotalDistance(ctx); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	if err := initializeChairsTotalRideCount(ctx); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, postInitializeResponse{Language: "go"})
}

func initializeChairsTotalDistance(ctx context.Context) error {
	type chairsWithTotalDistance struct {
		ID                     string        `db:"id"`
		OwnerID                string        `db:"owner_id"`
		TotalDistance          sql.NullInt64 `db:"total_distance"`
		TotalDistanceUpdatedAt sql.NullTime  `db:"total_distance_updated_at"`
	}
	var chairs []chairsWithTotalDistance
	if err := db.SelectContext(ctx, &chairs, `
SELECT id,
  owner_id,
  total_distance,
  total_distance_updated_at
FROM isu1.chairs
  LEFT JOIN (SELECT chair_id,
    SUM(COALESCE(distance, 0)) AS total_distance,
    MAX(created_at)          AS total_distance_updated_at
  FROM (SELECT chair_id,
    created_at,
    ABS(latitude - LAG(latitude) OVER (PARTITION BY chair_id ORDER BY created_at)) +
    ABS(longitude - LAG(longitude) OVER (PARTITION BY chair_id ORDER BY created_at)) AS distance
    FROM isu1.chair_locations) tmp
  GROUP BY chair_id) distance_table ON distance_table.chair_id = chairs.id
`); err != nil {
		return fmt.Errorf("failed to select chairs: %w", err)
	}
	for _, chair := range chairs {
		var updatedAt int64
		if chair.TotalDistanceUpdatedAt.Valid {
			updatedAt = chair.TotalDistanceUpdatedAt.Time.UnixMilli()
		}
		var totalDistance int
		if chair.TotalDistance.Valid {
			totalDistance = int(chair.TotalDistance.Int64)
		}
		if err := addChairTotalDistance(ctx, chair.ID, totalDistance, updatedAt); err != nil {
			return fmt.Errorf("failed to add chair total distance: %w", err)
		}
	}
	return nil
}

func initializeChairsTotalRideCount(ctx context.Context) error {
	type chairsWithTotalRideCount struct {
		ChairID         string `db:"chair_id"`
		TotalRideCount  int    `db:"total_ride_count"`
		TotalEvaluation int    `db:"total_evaluation"`
	}
	var chairs []chairsWithTotalRideCount
	if err := db.SelectContext(ctx, &chairs, `
SELECT chair_id,
  COUNT(id) AS total_ride_count,
  SUM(COALESCE(evaluation, 0)) AS total_evaluation
FROM rides
WHERE evaluation IS NOT NULL
GROUP BY chair_id
	`); err != nil {
		return fmt.Errorf("failed to select chairs: %w", err)
	}
	for _, chair := range chairs {
		if err := setChairTotalRideCount(ctx, chair.ChairID, chair.TotalRideCount, chair.TotalEvaluation); err != nil {
			return fmt.Errorf("failed to add chair total ride count: %w", err)
		}
	}
	return nil
}

type Coordinate struct {
	Latitude  int `json:"latitude"`
	Longitude int `json:"longitude"`
}

func bindJSON(r *http.Request, v interface{}) error {
	return json.NewDecoder(r.Body).Decode(v)
}

func writeJSON(w http.ResponseWriter, statusCode int, v interface{}) {
	w.Header().Set("Content-Type", "application/json;charset=utf-8")
	buf, err := json.Marshal(v)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(statusCode)
	w.Write(buf)
}

func writeError(w http.ResponseWriter, statusCode int, err error) {
	_, filename, linenum, ok := runtime.Caller(1)
	w.Header().Set("Content-Type", "application/json;charset=utf-8")
	w.WriteHeader(statusCode)
	buf, marshalError := json.Marshal(map[string]string{"message": err.Error()})
	if marshalError != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"marshaling error failed"}`))
		return
	}
	w.Write(buf)

	if ok {
		slog.Error("error response wrote", slog.Any("error", err), slog.String("file", filename), slog.Int("line", linenum))
	} else {
		slog.Error("error response wrote", slog.Any("error", err))
	}
}

func secureRandomStr(b int) string {
	k := make([]byte, b)
	if _, err := crand.Read(k); err != nil {
		panic(err)
	}
	return fmt.Sprintf("%x", k)
}
