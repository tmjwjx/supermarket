package conf

import "time"

// Bootstrap 是 configs/config.yaml 的顶层结构
type Bootstrap struct {
	Server Server `json:"server"`
	Data   Data   `json:"data"`
	Client Client `json:"client"`
	Auth   Auth   `json:"auth"`
}

// 买家令牌和后台令牌使用不同密钥
type Auth struct {
	JWTSecret      string `json:"jwt_secret"`
	AdminJWTSecret string `json:"admin_jwt_secret"`
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

// Client 是 product 调用的下游地址
type Client struct {
	Inventory Endpoint `json:"inventory"`
	Order     Endpoint `json:"order"`
	User      Endpoint `json:"user"`
}

type Endpoint struct {
	Addr      string `json:"addr"`
	TimeoutMs int64  `json:"timeout_ms"`
}

func (e Endpoint) Timeout() time.Duration {
	return time.Duration(e.TimeoutMs) * time.Millisecond
}
