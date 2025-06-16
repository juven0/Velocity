package cors

import (
	"strconv"
	"strings"

	"github.com/juven0/Velocity/types"
)

type Config struct {
	AllowedOrigins     []string
	AllowedMethods     []string
	AllowedHeaders     []string
	AllowedCredentials bool
	MaxAge             int
}

var DefaultConfig = Config{
	AllowedOrigins:     []string{"localhost:5555"},
	AllowedMethods:     []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
	AllowedHeaders:     []string{"Content-Type", "Authorization", "Accept"},
	AllowedCredentials: false,
	MaxAge:             86400,
}

func New(config ...Config) types.HandlerFunc {
	cfg := DefaultConfig

	if len(config) > 0 {
		cfg = config[0]
	}

	allowAllOrigins := false
	for _, origin := range cfg.AllowedOrigins {
		if origin == "*" {
			allowAllOrigins = true
			break
		}
	}

	if allowAllOrigins && cfg.AllowedCredentials {
		panic("[CORS] Configuration error: When 'AllowCredentials' is set to true, 'AllowOrigins' cannot contain a wildcard origin '*'. Please specify allowed origins explicitly or adjust 'AllowCredentials' setting.")
	}

	return func(ctx *types.Context) error {
		origin := string(ctx.Request.Header.Peek("Origin"))

		var allowedOrigin string
		if allowAllOrigins {
			allowedOrigin = "*"
		} else {
			for _, allowedOrig := range cfg.AllowedOrigins {
				if allowedOrig == origin {
					allowedOrigin = origin
					break
				}
			}
		}

		if allowedOrigin == "" && origin != "" {
			ctx.SetStatus(403)
			return nil
		}

		if allowedOrigin != "" {
			ctx.Response.Header.Set("Access-Control-Allow-Origin", allowedOrigin)
		}

		if cfg.AllowedCredentials {
			ctx.Response.Header.Set("Access-Control-Allow-Credentials", "true")
		}

		if string(ctx.Request.Header.Method()) == "OPTIONS" {
			ctx.Response.Header.Set("Access-Control-Allow-Methods", strings.Join(cfg.AllowedMethods, ", "))
			ctx.Response.Header.Set("Access-Control-Allow-Headers", strings.Join(cfg.AllowedHeaders, ", "))

			if cfg.MaxAge > 0 {
				ctx.Response.Header.Set("Access-Control-Max-Age", strconv.Itoa(cfg.MaxAge))
			}

			ctx.SetStatus(204)
			return nil
		}

		ctx.Response.Header.Set("Access-Control-Allow-Methods", strings.Join(cfg.AllowedMethods, ", "))

		return ctx.Next()
	}
}
