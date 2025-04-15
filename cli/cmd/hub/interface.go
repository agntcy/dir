package hub

import "context"

type Hub interface {
	Run(ctx context.Context, args []string) error
}
