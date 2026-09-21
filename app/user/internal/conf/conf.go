package conf

import "time"

// Bootstrap is the top-level config shape of configs/config.yaml.
type Bootstrap struct {
	Server Server `json:"server"`
	Data   Data   `json:"data"`
}

type Server struct {
	HTTP HTTP `json:"http"`
	GRPC GRPC `json:"grpc"`
}

type HTTP struct {
	Network   string `json:"network"`
	Addr      string `json:"addr"`
	TimeoutMs int64  `json:"timeout_ms"`
}

func (h HTTP) Timeout() time.Duration {
	return time.Duration(h.TimeoutMs) * time.Millisecond
}

type GRPC struct {
	Network   string `json:"network"`
	Addr      string `json:"addr"`
	TimeoutMs int64  `json:"timeout_ms"`
}

func (g GRPC) Timeout() time.Duration {
	return time.Duration(g.TimeoutMs) * time.Millisecond
}

type Data struct {
	Database Database `json:"database"`
	Redis    Redis    `json:"redis"`
}

type Database struct {
	Driver      string `json:"driver"`
	Source      string `json:"source"`
	Debug       bool   `json:"debug"`
	AutoMigrate bool   `json:"auto_migrate"`
}

type Redis struct {
	Network        string `json:"network"`
	Addr           string `json:"addr"`
	ReadTimeoutMs  int64  `json:"read_timeout_ms"`
	WriteTimeoutMs int64  `json:"write_timeout_ms"`
}

func (r Redis) ReadTimeout() time.Duration {
	return time.Duration(r.ReadTimeoutMs) * time.Millisecond
}

func (r Redis) WriteTimeout() time.Duration {
	return time.Duration(r.WriteTimeoutMs) * time.Millisecond
}
