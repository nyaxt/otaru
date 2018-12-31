package cli

import (
	"context"
)

type options struct {
	cfg            *CliConfig
	ctx            context.Context
	forceGrpc      bool
	allowOverwrite bool
	overrideToken  string
}

var defaultOptions = options{
	ctx:            context.Background(),
	forceGrpc:      false,
	allowOverwrite: false,
}

type Option func(*options)

func WithCliConfig(cfg *CliConfig) Option {
	return func(o *options) { o.cfg = cfg }
}

func WithContext(ctx context.Context) Option {
	return func(o *options) { o.ctx = ctx }
}

// AllowOverwrite allows NewWriter to open an existing file.
func AllowOverwrite(b bool) Option {
	return func(o *options) { o.allowOverwrite = b }
}

func ForceGrpc() Option {
	return func(o *options) { o.forceGrpc = true }
}

func WithOverrideToken(t string) Option {
	return func(o *options) { o.overrideToken = t }
}

func (o *options) QueryConnectionInfo(vhost string) (*ConnectionInfo, error) {
	ci, err := QueryConnectionInfo(o.cfg, vhost)
	if err != nil {
		return nil, err
	}

	if o.overrideToken != "" {
		ci.AuthToken = o.overrideToken
	}
	return ci, nil
}
