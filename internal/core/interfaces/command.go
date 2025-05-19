package interfaces

import (
	"context"
)

type Command[T any] interface {
	Execute(ctx context.Context, args T)
}
