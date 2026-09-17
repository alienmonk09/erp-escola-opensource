// main.go — hello vazio da fundação (F-01).
//
// Sobe o backend com `task dev` para cumprir o aceite local ("backend sobe,
// mesmo que com hello vazio"). Sem negócio, sem banco, sem sessão: só
// GET /api/saude → {"status":"ok"}. Handlers reais chegam em F-04 (SRS §12).
// Escuta SOMENTE em loopback.
package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
)

func saude(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"erro":"metodo_nao_permitido"}`, http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func main() {
	porta := os.Getenv("PORT")
	if porta == "" {
		porta = "8080"
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/saude", saude)
	endereco := "127.0.0.1:" + porta
	slog.Info("backend hello no ar", "endereco", "http://"+endereco)
	if err := http.ListenAndServe(endereco, mux); err != nil {
		slog.Error("backend encerrou", "erro", err)
		os.Exit(1)
	}
}
