package auth

import (
	"context"
	"errors"
	"time"

	"github.com/alienmonk09/erp-escola-opensource/backend/internal/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	falhasParaBloqueio = 5
	bloqueioInicial    = time.Minute
	bloqueioMaximo     = 30 * time.Minute
)

func ContaBloqueada(ctx context.Context, q *db.Queries, email string) (bool, error) {
	row, err := q.GetLoginBloqueio(ctx, email)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if !row.BloqueadoAte.Valid {
		return false, nil
	}
	return time.Now().Before(row.BloqueadoAte.Time), nil
}

func RegistrarFalha(ctx context.Context, q *db.Queries, email string) error {
	atual, err := q.GetLoginBloqueio(ctx, email)
	falhas := int32(0)
	if err == nil {
		falhas = atual.Falhas
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	proxima := falhas + 1
	var ate pgtype.Timestamptz
	if proxima >= falhasParaBloqueio {
		exp := 1
		extra := int(proxima - falhasParaBloqueio)
		for range extra {
			exp *= 2
		}
		d := time.Duration(exp) * bloqueioInicial
		if d > bloqueioMaximo {
			d = bloqueioMaximo
		}
		ate = pgtype.Timestamptz{Time: time.Now().Add(d), Valid: true}
	}
	_, err = q.UpsertLoginFalha(ctx, db.UpsertLoginFalhaParams{
		Email:        email,
		BloqueadoAte: ate,
	})
	return err
}

func LimparFalhas(ctx context.Context, q *db.Queries, email string) error {
	return q.ResetLoginBloqueio(ctx, email)
}
