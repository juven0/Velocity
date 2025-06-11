package cors

import (
	"strings"

	"github.com/juven0/Velocity/types"
)

type Config struct {
	AllowedOrigin       []string
	AllowedMethode      []string
	AllowedHeader       []string
	AllowedCrendentials bool
	MaxAge              int
}

var DefaultConfig = Config{
	AllowedOrigin:       []string{"*"},
	AllowedMethode:      []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTION"},
	AllowedHeader:       []string{"Content-type", "Authorization", "Accepte"},
	AllowedCrendentials: false,
	MaxAge:              3600,
}

func New(config ...Config) types.HandlerFunc {
	cfg := DefaultConfig

	if len(config) > 0 {
		cfg = config[0]
	}

	AllowAllOrigin := false

	for _, origin := range cfg.AllowedOrigin {
		if origin == "*" {
			AllowAllOrigin = true
		}
		break
	}

	if AllowAllOrigin && cfg.AllowedCrendentials {
		panic("[CORS] Configuration error: When 'AllowCredentials' is set to true, 'AllowOrigins' cannot contain a wildcard origin '*'. Please specify allowed origins explicitly or adjust 'AllowCredentials' setting.")
	}

	return func(ctx *types.Context) error {
		ctx.Response.Header.Set("Access-Control-Allow-Origin", strings.Join(cfg.AllowedOrigin, ","))
		ctx.Response.Header.Set("Access-Control-Allow-Methods", strings.Join(cfg.AllowedMethode, ","))
		ctx.Response.Header.Set("Access-Control-Allow-Headers", strings.Join(cfg.AllowedHeader, ","))
		return ctx.Next()
	}
}
