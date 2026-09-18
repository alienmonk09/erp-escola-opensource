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

// TotpOK registra a efetivação da sessão via segundo fator (RS-010).
// Nunca inclui o código (RS-010/066).
func (r Registro) TotpOK(ctx context.Context, usuario pgtype.UUID, req *http.Request) {
	r.gravar(ctx, usuario, "totp_ok", "sessao", req)
}

// TotpFalha registra código inválido sem incluir o código (RS-010/066).
func (r Registro) TotpFalha(ctx context.Context, usuario pgtype.UUID, req *http.Request) {
	r.gravar(ctx, usuario, "totp_falha", "sessao", req)
}

// TotpCadastrado registra a conclusão do cadastro do autenticador (RF-003).
func (r Registro) TotpCadastrado(ctx context.Context, usuario pgtype.UUID, req *http.Request) {
	r.gravar(ctx, usuario, "totp_cadastrado", "conta", req)
}

// TotpDesativado registra o desligamento do 2FA opcional (RF-003).
func (r Registro) TotpDesativado(ctx context.Context, usuario pgtype.UUID, req *http.Request) {
	r.gravar(ctx, usuario, "totp_desativado", "conta", req)
}

// SenhaTrocada registra a troca da própria senha (RS-006/010).
func (r Registro) SenhaTrocada(ctx context.Context, usuario pgtype.UUID, req *http.Request) {
	r.gravar(ctx, usuario, "senha_trocada", "conta", req)
}

// SenhaRedefinida registra a redefinição por admin com referência ao alvo
// (RS-007/010). Sem senha ou hash no detalhe.
func (r Registro) SenhaRedefinida(ctx context.Context, admin pgtype.UUID, alvo string, req *http.Request) {
	r.gravarRecurso(ctx, admin, "senha_redefinida", "usuario:"+alvo, req)
}

// TotpReiniciado registra o reinício por admin com referência ao alvo
// (RF-007, RS-008/010). Sem segredo ou código no detalhe.
func (r Registro) TotpReiniciado(ctx context.Context, admin pgtype.UUID, alvo string, req *http.Request) {
	r.gravarRecurso(ctx, admin, "totp_reiniciado", "usuario:"+alvo, req)
}

func (r Registro) gravar(ctx context.Context, usuario pgtype.UUID, acao, recurso string, req *http.Request) {
	r.gravarRecurso(ctx, usuario, acao, recurso, req)
}

func (r Registro) gravarRecurso(ctx context.Context, usuario pgtype.UUID, acao, recurso string, req *http.Request) {
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
