package conf

import "time"

const defaultTokenExpire = 12 * time.Hour

type Bootstrap struct {
	Server Server `json:"server"`
	Data   Data   `json:"data"`
	Auth   Auth   `json:"auth"`
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
}

type Database struct {
	Driver      string `json:"driver"`
	Source      string `json:"source"`
	Debug       bool   `json:"debug"`
	AutoMigrate bool   `json:"auto_migrate"`
}

type Auth struct {
	JWTSecret     string `json:"jwt_secret"`
	TokenExpireMs int64  `json:"token_expire_ms"`
}

func (a Auth) TokenExpire() time.Duration {
	if a.TokenExpireMs <= 0 {
		return defaultTokenExpire
	}
	return time.Duration(a.TokenExpireMs) * time.Millisecond
}
