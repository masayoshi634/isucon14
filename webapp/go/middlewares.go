package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
)

func appAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		_, span := tracer.Start(ctx, "appAuthMiddleware")
		defer span.End()

		c, err := r.Cookie("app_session")
		if errors.Is(err, http.ErrNoCookie) || c.Value == "" {
			writeError(w, http.StatusUnauthorized, errors.New("app_session cookie is required"))
			return
		}
		accessToken := c.Value
		fmt.Printf("accessToken: %v\n", accessToken)
		// user := &User{}
		// err = db.GetContext(ctx, user, "SELECT * FROM users WHERE access_token = ?", accessToken)
		// if err != nil {
		// 	if errors.Is(err, sql.ErrNoRows) {
		// 		writeError(w, http.StatusUnauthorized, errors.New("invalid access token"))
		// 		return
		// 	}
		// 	writeError(w, http.StatusInternalServerError, err)
		// 	return
		// }
		user, err := getCacheUser(ctx, accessToken)

		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				writeError(w, http.StatusUnauthorized, errors.New("invalid access token"))
				return
			}
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		ctx = context.WithValue(ctx, "user", user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

var userCache = NewCache[string, *User]()

// userをシングルフライトでキャッシュする
func getCacheUser(ctx context.Context, accessToken string) (*User, error) {
	_, span := tracer.Start(ctx, "getCacheUser")
	defer span.End()
	user := &User{}
	cacheUser, _ := userCache.Get(accessToken)

	if cacheUser != nil {
		// fmt.Println("cache hit %v", cacheUser)
		return cacheUser, nil
	}

	err := db.GetContext(ctx, user, "SELECT * FROM users WHERE access_token = ?", accessToken)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("invalid access token")
		}
		return nil, err
	}
	// fmt.Println("cache nohit %v+", user)
	userCache.Set(accessToken, user)

	return user, nil
}

func ownerAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		_, span := tracer.Start(ctx, "ownerAuthMiddleware")
		defer span.End()

		c, err := r.Cookie("owner_session")
		if errors.Is(err, http.ErrNoCookie) || c.Value == "" {
			writeError(w, http.StatusUnauthorized, errors.New("owner_session cookie is required"))
			return
		}
		accessToken := c.Value
		// owner := &Owner{}
		// if err := db.GetContext(ctx, owner, "SELECT * FROM isu1.wners WHERE access_token = ?", accessToken); err != nil {
		// 	if errors.Is(err, sql.ErrNoRows) {
		// 		writeError(w, http.StatusUnauthorized, errors.New("invalid access token"))
		// 		return
		// 	}
		// 	writeError(w, http.StatusInternalServerError, err)
		// 	return
		// }

		owner, err := getCacheOwner(ctx, accessToken)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				writeError(w, http.StatusUnauthorized, errors.New("invalid access token"))
				return
			}
			writeError(w, http.StatusInternalServerError, err)
			return
		}

		ctx = context.WithValue(ctx, "owner", owner)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

var ownerCache = NewCache[string, *Owner]()

// ownerをキャッシュする
func getCacheOwner(ctx context.Context, accessToken string) (*Owner, error) {
	_, span := tracer.Start(ctx, "getCacheUser")
	defer span.End()
	cacheOwner, _ := ownerCache.Get(accessToken)

	if cacheOwner != nil {
		// fmt.Println("cache hit %v", cacheOwner)
		return cacheOwner, nil
	}

	owner := &Owner{}
	err := db.GetContext(ctx, owner, "SELECT * FROM owners WHERE access_token = ?", accessToken)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("invalid access token")
		}
		return nil, err
	}
	// fmt.Println("cache nohit %v+", owner)
	ownerCache.Set(accessToken, owner)

	return owner, nil
}

func chairAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		_, span := tracer.Start(ctx, "chairAuthMiddleware")
		defer span.End()

		c, err := r.Cookie("chair_session")
		if errors.Is(err, http.ErrNoCookie) || c.Value == "" {
			writeError(w, http.StatusUnauthorized, errors.New("chair_session cookie is required"))
			return
		}
		accessToken := c.Value
		// chair := &Chair{}
		// err = db.GetContext(ctx, chair, "SELECT * FROM isu1.chairs WHERE access_token = ?", accessToken)
		// if err != nil {
		// 	if errors.Is(err, sql.ErrNoRows) {
		// 		writeError(w, http.StatusUnauthorized, errors.New("invalid access token"))
		// 		return
		// 	}
		// 	writeError(w, http.StatusInternalServerError, err)
		// 	return
		// }
		chair, err := getCacheChair(ctx, accessToken)

		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				writeError(w, http.StatusUnauthorized, errors.New("invalid access token"))
				return
			}
			writeError(w, http.StatusInternalServerError, err)
			return
		}

		ctx = context.WithValue(ctx, "chair", chair)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

var chairCache = NewCache[string, *Chair]()

// chairをキャッシュする
func getCacheChair(ctx context.Context, accessToken string) (*Chair, error) {
	_, span := tracer.Start(ctx, "getCacheUser")
	defer span.End()
	cacheChair, _ := chairCache.Get(accessToken)

	if cacheChair != nil {
		// fmt.Println("cache hit %v", cacheChair)
		return cacheChair, nil
	}

	chair := &Chair{}
	err := db.GetContext(ctx, chair, "SELECT * FROM isu1.chairs WHERE access_token = ?", accessToken)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("invalid access token")
		}
		return nil, err
	}
	// fmt.Println("cache nohit %v+", chair)
	chairCache.Set(accessToken, chair)

	return chair, nil
}
