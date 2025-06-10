package cors

type Option struct {
	AllowedOrigin       []string
	AllowedMethode      []string
	AllowedHeader       []string
	AllowedCrendentials bool
}

type Cors struct {
	AllowedOrigin      []string
	AllowedMethode     []string
	AllwoedHeader      []string
	AllowedCredentials bool
}

func New(option Option) *Cors {
	c := &Cors{
		AllowedOrigin:      option.AllowedOrigin,
		AllowedMethode:     option.AllowedMethode,
		AllwoedHeader:      option.AllowedHeader,
		AllowedCredentials: option.AllowedCrendentials,
	}
	return c
}
