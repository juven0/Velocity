package cors

import "github.com/juven0/Velocity/types"

type Option struct {
	AllowedOrigin       []string
	AllowedMethode      []string
	AllowedHeader       []string
	AllowedCrendentials bool
}

func New(option Option) types.HandlerFunc {
	return func(ctx types.Context) error {
		return ctx.Next()
	}
}
