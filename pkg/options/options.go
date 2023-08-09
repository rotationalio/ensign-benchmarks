package options

import "github.com/rotationalio/go-ensign"

// Reasonable defaults for benchmark options
const (
	DataSize   = 8192
	Operations = 10000
)

type Options struct {
	Endpoint    string `json:"endpoint" yaml:"endpoint"`
	AuthURL     string `json:"auth_url" yaml:"auth_url"`
	Credentials string `json:"-" yaml:"-"`
	Operations  uint64 `json:"operations" yaml:"operations"`
	DataSize    int64  `json:"data_size" yaml:"data_size"`
}

func New() *Options {
	return &Options{
		Operations: Operations,
		DataSize:   DataSize,
	}
}

func Ensign() []*ensign.Option {
	return nil
}
