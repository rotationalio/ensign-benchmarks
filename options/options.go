package options

// Reasonable defaults for benchmark options
const (
	DataSize   = 8192
	Operations = 10000
)

type Options struct {
	Addr       string `json:"addr" yaml:"addr"`
	Operations uint64 `json:"operations" yaml:"operations"`
	DataSize   int64  `json:"data_size" yaml:"data_size"`
}

func New() *Options {
	return &Options{
		Operations: Operations,
		DataSize:   DataSize,
	}
}
