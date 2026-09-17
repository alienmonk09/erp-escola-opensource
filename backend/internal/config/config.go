// Package config carrega configuração tipada via caarlos0/env (SRS §17).
package config

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"
)

// Config é fail-fast: falta de DATABASE_URL impede subir.
type Config struct {
	DatabaseURL string `env:"DATABASE_URL,required"`
	Porta       string `env:"PORT" envDefault:"8080"`
	// CookieSecure força Secure no cookie __Host- (RS-003/043).
	CookieSecure bool `env:"COOKIE_SECURE" envDefault:"true"`
}

func Carregar() (Config, error) {
	var c Config
	if err := env.Parse(&c); err != nil {
		return Config{}, fmt.Errorf("config: %w", err)
	}
	return c, nil
}

const (
	Inatividade = 30 * time.Minute
	Absoluto    = 12 * time.Hour
	CookieNome  = "__Host-sessao"
	LoginPorMin = 5
)
