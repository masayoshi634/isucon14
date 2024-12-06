package main

import (
	"context"
	"fmt"

	"github.com/labstack/echo/v4"
	"github.com/mackee/isutools/lazyresolve"
)

type resolvers struct {
	// usersResolver              lazyresolve.Resolver[User, int64]
}

type sqlxIf interface {
	SelectContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
}

func NewResolvers(conn sqlxIf) *resolvers {
	rs := &resolvers{}
	// rs.usersResolver = usersResolver(conn)
	return rs
}

func (r *resolvers) ResolveAll(ctx context.Context) error {
	_, span := tracer.Start(ctx, "resolvers.ResolveAll")
	defer span.End()
	resolvers := []lazyresolve.ResolverSubset{
		// r.usersResolver,
	}
	if err := lazyresolve.ResolveAll(ctx, resolvers...); err != nil {
		return fmt.Errorf("failed to resolve all: %w", err)
	}
	return nil
}

func WithResolvers(ctx context.Context, conn sqlxIf) func(context.Context) (context.Context, error) {
	_, span := tracer.Start(ctx, "WithResolvers")
	defer span.End()
	return func(ctx context.Context) (context.Context, error) {
		rs := NewResolvers(conn)
		return lazyresolve.WithResolvers(ctx, rs), nil
	}
}

/*
func usersResolver(conn sqlxIf) lazyresolve.Resolver[User, int64] {
	return lazyresolve.NewResolver("users", func(ctx context.Context, _userIDs []int64) ([]User, error) {
		resolver, err := lazyresolve.GetResolvers[*resolvers](ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get resolvers: %w", err)
		}
		userIDs := lo.Uniq(_userIDs)
		var userModels []UserModel
		query, args, err := sqlx.In("SELECT * FROM users WHERE id IN (?)", userIDs)
		if err != nil {
			return nil, fmt.Errorf("failed to build in query by userIDs: %w", err)
		}
		if err := conn.SelectContext(ctx, &userModels, query, args...); err != nil {
			return nil, fmt.Errorf("failed to query by userIDs: %w", err)
		}

		users := lo.Map(userModels, func(userModel UserModel, _ int) User {
			return User{
				ID:          userModel.ID,
				Name:        userModel.Name,
				DisplayName: userModel.DisplayName,
				Description: userModel.Description,
				Theme:       resolver.themesResolver.Future(userModel.ID),
				IconHash:    resolver.imagesResolver.Future(userModel.ID),
			}
		})

		return lazyresolve.SortByIndex(users, _userIDs, func(u User) int64 { return u.ID }), nil
	})
}
*/

func main_with_lazyresolve() {
	e := echo.New()
	ctx := context.Background()
	var conn sqlxIf // ここはDB接続のインターフェースを入れる
	e.Use(lazyresolve.ResolversMiddleware(WithResolvers(ctx, conn)))
	e.JSONSerializer = lazyresolve.NewJSONSerializer()
}
