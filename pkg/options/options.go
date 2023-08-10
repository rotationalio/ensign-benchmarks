package options

import "github.com/rotationalio/go-ensign"

// Reasonable defaults for benchmark options
const (
	Topic      = "benchmarks"
	DataSize   = 8192
	Operations = 10000
)

type Options struct {
	Topic       string `json:"topic" yaml:"topic"`
	Endpoint    string `json:"endpoint" yaml:"endpoint"`
	AuthURL     string `json:"auth_url" yaml:"auth_url"`
	Credentials string `json:"-" yaml:"-"`
	Operations  uint64 `json:"operations" yaml:"operations"`
	DataSize    int64  `json:"data_size" yaml:"data_size"`
}

func New() *Options {
	return &Options{
		Topic:      Topic,
		Operations: Operations,
		DataSize:   DataSize,
	}
}

func (o Options) Ensign() []ensign.Option {
	opts := make([]ensign.Option, 0, 3)
	if o.Credentials != "" {
		opts = append(opts, ensign.WithLoadCredentials(o.Credentials))
	}

	if o.Endpoint != "" {
		opts = append(opts, ensign.WithEnsignEndpoint(o.Endpoint, false))
	}

	if o.AuthURL != "" {
		opts = append(opts, ensign.WithAuthenticator(o.AuthURL, false))
	}

	return opts
}
