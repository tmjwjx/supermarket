package conf

import "time"

// 网关运行配置 不含数据库
type Bootstrap struct {
	Server Server `json:"server"`
	Client Client `json:"client"`
	Auth   Auth   `json:"auth"`
}

type Server struct {
	HTTP HTTP `json:"http"`
}

type HTTP struct {
	Network   string `json:"network"`
	Addr      string `json:"addr"`
	TimeoutMs int64  `json:"timeout_ms"`
}

func (h HTTP) Timeout() time.Duration {
	return time.Duration(h.TimeoutMs) * time.Millisecond
}

// user 服务的 gRPC 地址
type Client struct {
	User User `json:"user"`
}

type User struct {
	Addr      string `json:"addr"`
	TimeoutMs int64  `json:"timeout_ms"`
}

func (u User) Timeout() time.Duration {
	return time.Duration(u.TimeoutMs) * time.Millisecond
}

// 与 user 的 jwt_secret 同一环境变量占位
type Auth struct {
	JWTSecret string `json:"jwt_secret"`
}
