package conf

import "time"

type Bootstrap struct {
	Server  Server  `json:"server"`
	Data    Data    `json:"data"`
	Client  Client  `json:"client"`
	Payment Payment `json:"payment"`
	Auth    Auth    `json:"auth"`
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

type Client struct {
	Order Order `json:"order"`
}

type Order struct {
	Addr      string `json:"addr"`
	TimeoutMs int64  `json:"timeout_ms"`
}

func (o Order) Timeout() time.Duration {
	if o.TimeoutMs <= 0 {
		return time.Second
	}
	return time.Duration(o.TimeoutMs) * time.Millisecond
}

// SimulateEnabled 缺省为开启 配置显式 false 才关闭
type Payment struct {
	SimulateEnabled *bool `json:"simulate_enabled"`
}

func (p Payment) SimulateOn() bool {
	if p.SimulateEnabled == nil {
		return true
	}
	return *p.SimulateEnabled
}

// 买家令牌和后台令牌使用不同密钥 只用于本服务 HTTP 端口验签
type Auth struct {
	JWTSecret      string `json:"jwt_secret"`
	AdminJWTSecret string `json:"admin_jwt_secret"`
}
