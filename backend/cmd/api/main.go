// main.go — API F-04: sessão, RBAC, rate limit, headers, health.
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/alienmonk09/erp-escola-opensource/backend/internal/auth"
	"github.com/alienmonk09/erp-escola-opensource/backend/internal/config"
	"github.com/alienmonk09/erp-escola-opensource/backend/internal/servidor"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	cfg, err := config.Carregar()
	if err != nil {
		slog.Error("config invalida")
		os.Exit(1)
	}

	ctx := context.Background()
	pcfg, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		slog.Error("database_url invalida")
		os.Exit(1)
	}
	// Pooler Neon: QueryExecModeExec evita prepared statements no pooler
	// ([VERIFICAR] SRS §31 item 9).
	pcfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeExec
	pool, err := pgxpool.NewWithConfig(ctx, pcfg)
	if err != nil {
		slog.Error("pool pgx falhou")
		os.Exit(1)
	}
	defer pool.Close()

	sess := auth.NovaSessao(pool, cfg.CookieSecure)
	h := servidor.Novo(pool, sess)

	endereco := "127.0.0.1:" + cfg.Porta
	srv := &http.Server{
		Addr:              endereco,
		Handler:           h,
		ReadHeaderTimeout: 5 * time.Second,
	}
	slog.Info("backend no ar", "endereco", "http://"+endereco)
	if err := srv.ListenAndServe(); err != nil {
		slog.Error("backend encerrou")
		os.Exit(1)
	}
}
