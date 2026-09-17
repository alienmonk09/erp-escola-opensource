// Package auditoria grava ações sensíveis sem dados pessoais (RS-010/066).
package auditoria

import (
	"context"
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"strings"

	"github.com/alienmonk09/erp-escola-opensource/backend/internal/db"
	"github.com/jackc/pgx/v5/pgtype"
)

type Registro struct {
	Queries *db.Queries
}

func (r Registro) LoginOK(ctx context.Context, usuario pgtype.UUID, req *http.Request) {
	r.gravar(ctx, usuario, "login_ok", "sessao", req)
}

func (r Registro) LoginFalha(ctx context.Context, req *http.Request) {
	r.gravar(ctx, pgtype.UUID{}, "login_falha", "sessao", req)
}

func (r Registro) AcessoNegado(ctx context.Context, usuario pgtype.UUID, req *http.Request) {
	r.gravar(ctx, usuario, "acesso_negado", req.URL.Path, req)
}

func (r Registro) gravar(ctx context.Context, usuario pgtype.UUID, acao, recurso string, req *http.Request) {
	if r.Queries == nil {
		return
	}
	detalhe, _ := json.Marshal(map[string]string{"metodo": req.Method})
	_, err := r.Queries.InsertAuditoria(ctx, db.InsertAuditoriaParams{
		UsuarioID: usuario,
		Acao:      acao,
		Recurso:   recurso,
		Detalhe:   detalhe,
		Ip:        textoIP(IPCliente(req)),
	})
	if err != nil {
		slog.Error("auditoria falhou", "acao", acao)
	}
}

func IPCliente(req *http.Request) string {
	host, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		return strings.TrimSpace(req.RemoteAddr)
	}
	return host
}

func textoIP(ip string) pgtype.Text {
	if ip == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: ip, Valid: true}
}
