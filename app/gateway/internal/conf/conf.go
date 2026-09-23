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

// 各上游 gRPC 地址和超时
type Client struct {
	User         Endpoint `json:"user"`
	Product      Endpoint `json:"product"`
	Inventory    Endpoint `json:"inventory"`
	Order        Endpoint `json:"order"`
	Payment      Endpoint `json:"payment"`
	Notification Endpoint `json:"notification"`
	Admin        Endpoint `json:"admin"`
}

type Endpoint struct {
	Addr      string `json:"addr"`
	TimeoutMs int64  `json:"timeout_ms"`
}

func (e Endpoint) Timeout() time.Duration {
	return time.Duration(e.TimeoutMs) * time.Millisecond
}

// 买家令牌和后台令牌使用不同密钥
type Auth struct {
	JWTSecret      string `json:"jwt_secret"`
	AdminJWTSecret string `json:"admin_jwt_secret"`
}
