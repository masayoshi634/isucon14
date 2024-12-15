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
	// if err := tx.GetContext(ctx, &matchedChairID, "SELECT chair_id FROM vacant_chair FOR UPDATE SKIP LOCKED LIMIT 1"); err != nil && !errors.Is(err, sql.ErrNoRows) {
	// 	writeError(w, http.StatusInternalServerError, err)
	// 	return
	// }

	if err := tx.GetContext(ctx, &matchedChairID, "DELETE FROM vacant_chair WHERE CTID = (SELECT CTID FROM vacant_chair FOR UPDATE SKIP LOCKED LIMIT 1) RETURNING chair_id"); err != nil && !errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	if matchedChairID == "" {
		w.WriteHeader(http.StatusNoContent)
	}
	/*
		empty := false
		for i := 0; i < 10; i++ {
			if err := db.GetContext(ctx, matched, "SELECT * FROM isu1.chairs INNER JOIN (SELECT id FROM isu1.chairs WHERE is_active = 1 ORDER BY RANDOM() LIMIT 1) AS tmp ON chairs.id = tmp.id LIMIT 1"); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					w.WriteHeader(http.StatusNoContent)
					return
				}
				writeError(w, http.StatusInternalServerError, err)
			}

			if err := db.GetContext(ctx, &empty, "SELECT COUNT(*) = 0 FROM (SELECT COUNT(chair_sent_at) = 6 AS completed FROM ride_statuses WHERE ride_id IN (SELECT id FROM rides WHERE chair_id = ?) GROUP BY ride_id) is_completed WHERE completed = FALSE", matched.ID); err != nil {
				writeError(w, http.StatusInternalServerError, err)
				return
			}
			if empty {
				break
			}
		}
		if !empty {
			w.WriteHeader(http.StatusNoContent)
			return
		}
	*/

	if _, err := tx.ExecContext(ctx, "UPDATE rides SET chair_id = ?, updated_at = CURRENT_TIMESTAMP(6) WHERE id = ?", matchedChairID, ride.ID); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	// if _, err := tx.ExecContext(ctx, "DELETE FROM vacant_chair WHERE chair_id = ?", matchedChairID); err != nil {
	// 	writeError(w, http.StatusInternalServerError, err)
	// 	return
	// }
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
