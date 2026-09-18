// Package httpx monta o envelope Erro do contrato (SRS §15.3).
package httpx

import (
	"encoding/json"
	"net/http"

	"github.com/alienmonk09/erp-escola-opensource/backend/internal/httpapi"
)

func Envelope(codigo httpapi.ErroErroCodigo, mensagem string) httpapi.Erro {
	var e httpapi.Erro
	e.Erro.Codigo = codigo
	e.Erro.Mensagem = mensagem
	return e
}

func Escrever(w http.ResponseWriter, status int, codigo httpapi.ErroErroCodigo, mensagem string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(Envelope(codigo, mensagem))
}

const (
	MsgCredenciais = "Credenciais inválidas."
	MsgNaoAuth     = "Não autenticado."
	MsgSemPerm     = "Sem permissão."
	MsgLimite      = "Limite de tentativas excedido."
	MsgInterno     = "Erro interno."
	MsgPayload     = "Payload inválido."
	MsgTroca       = "Troca de senha obrigatória."
	MsgTotp        = "Cadastro do autenticador pendente."
)
