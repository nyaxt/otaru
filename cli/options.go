package cli

import (
	"context"
)

type options struct {
	cfg       *CliConfig
	ctx       context.Context
	forceGrpc bool
}

var defaultOptions = options{
	ctx:       context.Background(),
	forceGrpc: false,
}

type Option func(*options)

func WithCliConfig(cfg *CliConfig) Option {
	return func(o *options) { o.cfg = cfg }
}

func WithContext(ctx context.Context) Option {
	return func(o *options) { o.ctx = ctx }
}

func ForceGrpc() Option {
	return func(o *options) { o.forceGrpc = true }
}
