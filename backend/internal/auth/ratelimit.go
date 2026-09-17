package auth

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/alienmonk09/erp-escola-opensource/backend/internal/config"
	"github.com/alienmonk09/erp-escola-opensource/backend/internal/db"
	"github.com/alienmonk09/erp-escola-opensource/backend/internal/httpapi"
	"github.com/alienmonk09/erp-escola-opensource/backend/internal/httpx"
	"github.com/go-chi/httprate"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type ContadorPG struct {
	Q *db.Queries
}

func (c ContadorPG) Config(requestLimit int, windowLength time.Duration) {}

func (c ContadorPG) Increment(key string, currentWindow time.Time) error {
	return c.IncrementBy(key, currentWindow, 1)
}

func (c ContadorPG) IncrementBy(key string, currentWindow time.Time, amount int) error {
	ctx := context.Background()
	for range amount {
		if _, err := c.Q.IncrementarHttprate(ctx, db.IncrementarHttprateParams{
			Chave:  key,
			Janela: pgtype.Timestamptz{Time: currentWindow.UTC(), Valid: true},
		}); err != nil {
			return err
		}
	}
	return nil
}

func (c ContadorPG) Get(key string, currentWindow, previousWindow time.Time) (int, int, error) {
	ctx := context.Background()
	cur, err := c.ler(ctx, key, currentWindow)
	if err != nil {
		return 0, 0, err
	}
	prev, err := c.ler(ctx, key, previousWindow)
	if err != nil {
		return cur, 0, err
	}
	return cur, prev, nil
}

func (c ContadorPG) ler(ctx context.Context, key string, janela time.Time) (int, error) {
	n, err := c.Q.GetHttprate(ctx, db.GetHttprateParams{
		Chave:  key,
		Janela: pgtype.Timestamptz{Time: janela.UTC(), Valid: true},
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return int(n), nil
}

func LimitadorLogin(q *db.Queries) *httprate.RateLimiter {
	return httprate.NewRateLimiter(config.LoginPorMin, time.Minute,
		httprate.WithKeyByIP(),
		httprate.WithLimitCounter(ContadorPG{Q: q}),
		httprate.WithLimitHandler(func(w http.ResponseWriter, r *http.Request) {
			httpx.Escrever(w, http.StatusTooManyRequests, httpapi.ErroErroCodigoLimiteExcedido, httpx.MsgLimite)
		}),
	)
}
