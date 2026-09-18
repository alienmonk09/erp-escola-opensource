// Package auth — TOTP da F-05 (RF-001..003, RF-007, RS-008, RS-009).
//
// Segredo cifrado com NaCl secretbox; chave de 32 bytes lida de
// TOTP_ENCRYPTION_KEY (só Production tem valor real — RS-060/066).
// A chave aceita bytes crus (32) ou base64 de 32 bytes. Sem chave válida,
// as operações de segredo falham sem expor nada em log/erro (RS-066).
// Códigos TOTP nunca são persistidos (só validados em memória).
package auth

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/nacl/secretbox"
)

// Emissor aparece no otpauth:// do autenticador (cadastro guiado, I-10).
const EmissorTOTP = "ERP Escola"

// ErroChave indica TOTP_ENCRYPTION_KEY ausente ou inválida. A mensagem é
// genérica de propósito: nunca vaza detalhe da chave (RS-066).
var ErroChave = errors.New("totp indisponivel")

// ObrigatorioPara diz se o perfil exige 2FA (SRS §9, RS-009).
func ObrigatorioPara(perfis []string) bool {
	for _, p := range perfis {
		switch p {
		case "FIN", "RH", "DIR", "ADM":
			return true
		}
	}
	return false
}

// ExigeTOTP diz se o login pede segundo fator: 2FA obrigatório pendente ou
// TOTP ativo (opcional habilitado ou obrigatório já cadastrado).
func ExigeTOTP(usuarioTotpAtivo bool, perfis []string) bool {
	return usuarioTotpAtivo || ObrigatorioPara(perfis)
}

func chave() ([32]byte, error) {
	var k [32]byte
	raw := os.Getenv("TOTP_ENCRYPTION_KEY")
	if raw == "" {
		return k, ErroChave
	}
	if len(raw) == 32 {
		copy(k[:], raw)
		return k, nil
	}
	dec, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return k, ErroChave
	}
	if len(dec) != 32 {
		return k, ErroChave
	}
	copy(k[:], dec)
	return k, nil
}

// GerarSegredo cria um segredo TOTP novo e o devolve cifrado para persistir,
// junto do segredo em claro (só para exibir UMA vez no cadastro guiado) e da
// URL otpauth:// do QR code. O claro nunca deve ser logado (RS-066).
func GerarSegredo(email string) (cifrado []byte, segredoClaro, url string, err error) {
	chave, err := chave()
	if err != nil {
		return nil, "", "", err
	}
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      EmissorTOTP,
		AccountName: email,
	})
	if err != nil {
		return nil, "", "", fmt.Errorf("gerar totp: %w", err)
	}
	segredoClaro = key.Secret()
	var nonce [24]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return nil, "", "", fmt.Errorf("nonce: %w", err)
	}
	cifrado = secretbox.Seal(nonce[:], []byte(segredoClaro), &nonce, &chave)
	return cifrado, segredoClaro, key.URL(), nil
}

// DecifrarSegredo abre o segredo cifrado do banco. Erro genérico (RS-066).
func DecifrarSegredo(cifrado []byte) (string, error) {
	chave, err := chave()
	if err != nil {
		return "", err
	}
	if len(cifrado) < 24 {
		return "", ErroChave
	}
	var nonce [24]byte
	copy(nonce[:], cifrado[:24])
	aberto, ok := secretbox.Open(nil, cifrado[24:], &nonce, &chave)
	if !ok {
		return "", ErroChave
	}
	return string(aberto), nil
}

// ValidarCodigo confere o código de 6 dígitos contra o segredo em claro.
// Tolera uma janela de 30 s para desvio de relógio (pquerna/otp Validate).
// O código nunca é persistido — só validado em memória (RS-010).
func ValidarCodigo(segredoClaro, codigo string) bool {
	return totp.Validate(codigo, segredoClaro)
}

// CodigoAtual gera o código vigente para um segredo (usado em testes para
// concluir o cadastro; em produção o código vem do autenticador do usuário).
func CodigoAtual(segredoClaro string) (string, error) {
	return totp.GenerateCode(segredoClaro, time.Now())
}

// ValidarFormato garante 6 dígitos antes de qualquer validação (RS-030).
func ValidarFormato(codigo string) bool {
	if len(codigo) != 6 {
		return false
	}
	for i := 0; i < 6; i++ {
		if codigo[i] < '0' || codigo[i] > '9' {
			return false
		}
	}
	return true
}

// GerarSenhaTemporaria cria senha de uso único (RF-005, RS-007): 20
// caracteres alfanuméricos via crypto/rand. Devolvida uma única vez na
// resposta da redefinição; só o hash argon2id é persistido.
func GerarSenhaTemporaria() (string, error) {
	const alfabeto = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz23456789"
	const tamanho = 20
	buf := make([]byte, tamanho)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("aleatorio: %w", err)
	}
	for i := range buf {
		buf[i] = alfabeto[int(buf[i])%len(alfabeto)]
	}
	return string(buf), nil
}
