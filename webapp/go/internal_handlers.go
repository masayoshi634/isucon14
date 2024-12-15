package main

import (
	"database/sql"
	"errors"
	"net/http"
)

// このAPIをインスタンス内から一定間隔で叩かせることで、椅子とライドをマッチングさせる

func internalGetMatching(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	_, span := tracer.Start(ctx, "internalGetMatching")
	defer span.End()

	// MEMO: 一旦最も待たせているリクエストに適当な空いている椅子マッチさせる実装とする。おそらくもっといい方法があるはず…
	ride := &Ride{}
	if err := db.GetContext(ctx, ride, `SELECT * FROM rides WHERE chair_id IS NULL ORDER BY created_at LIMIT 1`); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
	}
	var matchedChairID string
	if err := tx.GetContext(ctx, &matchedChairID, "SELECT chair_id FROM vacant_chair FOR UPDATE SKIP LOCKED LIMIT 1"); err != nil && !errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if matchedChairID == "" {
		w.WriteHeader(http.StatusNoContent)
	}

	if _, err := tx.ExecContext(ctx, "UPDATE rides SET chair_id = ?, updated_at = CURRENT_TIMESTAMP(6) WHERE id = ?", matchedChairID, ride.ID); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM vacant_chair WHERE chair_id = ?", matchedChairID); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

/*
func internalGetMatching(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	_, span := tracer.Start(ctx, "internalGetMatching")
	defer span.End()

	var rides []Ride
	if err := db.SelectContext(ctx, rides, "SELECT * FROM rides WHERE chair_id IS NULL"); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	var vacantChairs []VacantChair
	if err := db.SelectContext(ctx, vacantChairs, "SELECT * FROM vacant_chair"); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	matrix := make([][]float64, len(rides))
	for i, ride := range rides {
		row := make([]float64, len(vacantChairs))
		for j, vc := range vacantChairs {
			row[j] = float64(vc.Distance(ride)) / float64(vc.Speed)
		}
		matrix[i] = row
	}
	result := hungarian.SolveMin(matrix)
	for i, v := range result {
		ride := rides[i]
		var vi int
		for j := range v {
			vi = j
			break
		}
		vc := vacantChairs[vi]
		tx, err := db.BeginTxx(ctx, nil)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		if _, err := tx.ExecContext(ctx, "UPDATE rides SET chair_id = ?, updated_at = CURRENT_TIMESTAMP(6) WHERE id = ?", vc.ChairID, ride.ID); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		if _, err := tx.ExecContext(ctx, "DELETE FROM vacant_chair WHERE chair_id = ?", vc.ChairID); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		if err := tx.Commit(); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
	}

	w.WriteHeader(http.StatusNoContent)
}
*/
