package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/redis/go-redis/v9"
	"github.com/samber/lo"
)

const (
	initialFare     = 500
	farePerDistance = 100
)

type ownerPostOwnersRequest struct {
	Name string `json:"name"`
}

type ownerPostOwnersResponse struct {
	ID                 string `json:"id"`
	ChairRegisterToken string `json:"chair_register_token"`
}

func ownerPostOwners(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	_, span := tracer.Start(ctx, "ownerPostOwners")
	defer span.End()

	req := &ownerPostOwnersRequest{}
	if err := bindJSON(r, req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, errors.New("some of required fields(name) are empty"))
		return
	}

	ownerID := ulid.Make().String()
	accessToken := secureRandomStr(32)
	chairRegisterToken := secureRandomStr(32)

	_, err := db.ExecContext(
		ctx,
		"INSERT INTO owners (id, name, access_token, chair_register_token) VALUES (?, ?, ?, ?)",
		ownerID, req.Name, accessToken, chairRegisterToken,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Path:  "/",
		Name:  "owner_session",
		Value: accessToken,
	})

	writeJSON(w, http.StatusCreated, &ownerPostOwnersResponse{
		ID:                 ownerID,
		ChairRegisterToken: chairRegisterToken,
	})
}

type chairSales struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Sales int    `json:"sales"`
}

type modelSales struct {
	Model string `json:"model"`
	Sales int    `json:"sales"`
}

type ownerGetSalesResponse struct {
	TotalSales int          `json:"total_sales"`
	Chairs     []chairSales `json:"chairs"`
	Models     []modelSales `json:"models"`
}

func ownerGetSales(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	_, span := tracer.Start(ctx, "ownerGetSales")
	defer span.End()

	since := time.Unix(0, 0)
	until := time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC)
	if r.URL.Query().Get("since") != "" {
		parsed, err := strconv.ParseInt(r.URL.Query().Get("since"), 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		since = time.UnixMilli(parsed)
	}
	if r.URL.Query().Get("until") != "" {
		parsed, err := strconv.ParseInt(r.URL.Query().Get("until"), 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		until = time.UnixMilli(parsed)
	}

	owner := r.Context().Value("owner").(*Owner)

	tx, err := db.Beginx()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	defer tx.Rollback()

	chairs := []Chair{}
	if err := tx.SelectContext(ctx, &chairs, "SELECT * FROM chairs WHERE owner_id = ?", owner.ID); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	res := ownerGetSalesResponse{
		TotalSales: 0,
	}

	modelSalesByModel := map[string]int{}
	for _, chair := range chairs {
		rides := []Ride{}

		if err := tx.SelectContext(ctx, &rides, "SELECT rides.* FROM rides JOIN ride_statuses ON rides.id = ride_statuses.ride_id WHERE chair_id = ? AND status = 'COMPLETED' AND updated_at BETWEEN ? AND ?", chair.ID, since, until.Add(999*time.Microsecond)); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}

		sales := sumSales(rides)
		res.TotalSales += sales

		res.Chairs = append(res.Chairs, chairSales{
			ID:    chair.ID,
			Name:  chair.Name,
			Sales: sales,
		})

		modelSalesByModel[chair.Model] += sales
	}

	models := []modelSales{}
	for model, sales := range modelSalesByModel {
		models = append(models, modelSales{
			Model: model,
			Sales: sales,
		})
	}
	res.Models = models

	writeJSON(w, http.StatusOK, res)
}

func sumSales(rides []Ride) int {
	sale := 0
	for _, ride := range rides {
		sale += calculateSale(ride)
	}
	return sale
}

func calculateSale(ride Ride) int {
	return calculateFare(ride.PickupLatitude, ride.PickupLongitude, ride.DestinationLatitude, ride.DestinationLongitude)
}

type chairWithDetail struct {
	ID                     string       `db:"id"`
	OwnerID                string       `db:"owner_id"`
	Name                   string       `db:"name"`
	AccessToken            string       `db:"access_token"`
	Model                  string       `db:"model"`
	IsActive               int          `db:"is_active"`
	CreatedAt              time.Time    `db:"created_at"`
	UpdatedAt              time.Time    `db:"updated_at"`
	TotalDistance          int          `db:"total_distance"`
	TotalDistanceUpdatedAt sql.NullTime `db:"total_distance_updated_at"`
}

type ownerGetChairResponse struct {
	Chairs []ownerGetChairResponseChair `json:"chairs"`
}

type ownerGetChairResponseChair struct {
	ID                     string `json:"id"`
	Name                   string `json:"name"`
	Model                  string `json:"model"`
	Active                 bool   `json:"active"`
	RegisteredAt           int64  `json:"registered_at"`
	TotalDistance          int    `json:"total_distance"`
	TotalDistanceUpdatedAt *int64 `json:"total_distance_updated_at,omitempty"`
}

func chairTotalDistanceKey(chairID string) string {
	return fmt.Sprintf("chair:%s:total_distance", chairID)
}

func chairTotalDistanceUpdatedAtKey(chairID string) string {
	return fmt.Sprintf("chair:%s:total_distance_updated_at", chairID)
}

func addChairTotalDistance(ctx context.Context, chairID string, distance int, updatedAtMilli int64) error {
	if _, err := rdb.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		if err := rdb.IncrBy(ctx, chairTotalDistanceKey(chairID), int64(distance)).Err(); err != nil {
			return fmt.Errorf("failed to add total distance: %w", err)
		}
		if err := rdb.Set(ctx, chairTotalDistanceUpdatedAtKey(chairID), updatedAtMilli, 0).Err(); err != nil {
			return fmt.Errorf("failed to set total distance updated at: %w", err)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("failed to add total distance: %w", err)
	}
	return nil
}

type chairTotalDistance struct {
	ChairID       string
	TotalDistance int
	UpdatedAt     int64
}

func getChairsTotalDistances(ctx context.Context, chairIDs []string) (map[string]*chairTotalDistance, error) {
	keys := lo.FlatMap(chairIDs, func(id string, _ int) []string {
		return []string{chairTotalDistanceKey(id), chairTotalDistanceUpdatedAtKey(id)}
	})
	result := rdb.MGet(ctx, keys...)
	if err := result.Err(); err != nil {
		return nil, fmt.Errorf("failed to get total distances: %w", err)
	}
	chairTotalDistances := make(map[string]*chairTotalDistance, len(chairIDs))
	vals := result.Val()
	for i := 0; i < len(keys); i += 2 {
		if vals[i] == nil {
			continue
		}
		distance, err := strconv.Atoi(vals[i].(string))
		if err != nil {
			return nil, fmt.Errorf("failed to parse total distance: %w", err)
		}
		updatedAt, err := strconv.ParseInt(vals[i+1].(string), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse total distance updated at: %w", err)
		}
		chairTotalDistances[chairIDs[i/2]] = &chairTotalDistance{
			ChairID:       chairIDs[i/2],
			TotalDistance: distance,
			UpdatedAt:     updatedAt,
		}
	}
	return chairTotalDistances, nil
}

func ownerGetChairs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	_, span := tracer.Start(ctx, "ownerGetChairs")
	defer span.End()

	owner := ctx.Value("owner").(*Owner)

	type chairDetail struct {
		ID        string    `db:"id"`
		Name      string    `db:"name"`
		Model     string    `db:"model"`
		IsActive  int       `db:"is_active"`
		CreatedAt time.Time `db:"created_at"`
	}
	chairs := []chairWithDetail{}
	if err := db.SelectContext(
		ctx,
		&chairs,
		`SELECT id, owner_id, name, access_token, model, is_active, created_at, updated_at FROM chairs WHERE owner_id = ?`,
		owner.ID,
	); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	chaidIDs := lo.Map(chairs, func(chair chairWithDetail, _ int) string { return chair.ID })
	totalDistances, err := getChairsTotalDistances(ctx, chaidIDs)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	res := ownerGetChairResponse{}
	for _, chair := range chairs {
		isActive := false
		if chair.IsActive != 0 {
			isActive = true
		}
		totalDistanceDetail := totalDistances[chair.ID]
		var totalDistance int
		var totalDistanceUpdatedAt *int64
		if totalDistanceDetail != nil {
			totalDistance = totalDistanceDetail.TotalDistance
			totalDistanceUpdatedAt = &totalDistanceDetail.UpdatedAt
		}
		c := ownerGetChairResponseChair{
			ID:                     chair.ID,
			Name:                   chair.Name,
			Model:                  chair.Model,
			Active:                 isActive,
			RegisteredAt:           chair.CreatedAt.UnixMilli(),
			TotalDistance:          totalDistance,
			TotalDistanceUpdatedAt: totalDistanceUpdatedAt,
		}
		res.Chairs = append(res.Chairs, c)
	}
	writeJSON(w, http.StatusOK, res)
}
