package types

import (
	"github.com/juven0/Velocity/velocity"
)

type HandlerFunc = func(*velocity.Context) error
