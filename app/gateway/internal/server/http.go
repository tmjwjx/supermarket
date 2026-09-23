package server

import (
	"github.com/tmjwjx/supermarket/app/gateway/internal/conf"

	"github.com/go-kratos/kratos/contrib/otel/v3/tracing"
	"github.com/go-kratos/kratos/v3/middleware/recovery"
	"github.com/go-kratos/kratos/v3/transport/http"
)

// 只开 HTTP 公开 可选 登录和后台路由都转到上游 gRPC
func NewHTTPServer(c *conf.Server, s Services) *http.Server {
	var opts = []http.ServerOption{
		http.Middleware(
			recovery.Recovery(),
			tracing.Server(),
		),
	}
	if c.HTTP.Network != "" {
		opts = append(opts, http.Network(c.HTTP.Network))
	}
	if c.HTTP.Addr != "" {
		opts = append(opts, http.Address(c.HTTP.Addr))
	}
	if d := c.HTTP.Timeout(); d != 0 {
		opts = append(opts, http.Timeout(d))
	}
	srv := http.NewServer(opts...)
	g := gate{tokens: s.Tokens, admins: s.AdminTokens}
	r := srv.Route("/")
	(&userProxy{gate: g, users: s.Users, addresses: s.Addresses}).routes(r)
	(&productProxy{
		gate:            g,
		products:        s.Products,
		brands:          s.Brands,
		categories:      s.Categories,
		favorites:       s.Favorites,
		histories:       s.Histories,
		reviews:         s.Reviews,
		recommendations: s.Recommendations,
		attributes:      s.Attributes,
	}).routes(r)
	(&inventoryProxy{gate: g, stocks: s.Stocks}).routes(r)
	(&orderProxy{gate: g, carts: s.Carts, orders: s.Orders}).routes(r)
	(&paymentProxy{gate: g, payments: s.Payments}).routes(r)
	(&notificationProxy{gate: g, notifications: s.Notifications}).routes(r)
	(&adminProxy{gate: g, admins: s.Admins}).routes(r)
	return srv
}

// 自定义方法允许空 body 没有 Content-Type 时不把解码失败当成参数错误
func bindBody(ctx http.Context, in any) error {
	r := ctx.Request()
	if r.Header.Get("Content-Type") == "" && r.ContentLength == 0 {
		return nil
	}
	return ctx.Bind(in)
}
