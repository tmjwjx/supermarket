package conf

import "time"

type Bootstrap struct {
	Server Server `json:"server"`
	Data   Data   `json:"data"`
	Auth   Auth   `json:"auth"`
	Kafka  Kafka  `json:"kafka"`
}

// Kafka 是消费商品事件用的地址 容器内写服务名
type Kafka struct {
	Brokers string `json:"brokers"`
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

// 买家令牌和后台令牌使用不同密钥 只用于本服务 HTTP 端口验签
type Auth struct {
	JWTSecret      string `json:"jwt_secret"`
	AdminJWTSecret string `json:"admin_jwt_secret"`
}
