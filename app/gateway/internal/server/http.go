package server

import (
	"context"

	userv1 "github.com/tmjwjx/supermarket/api/user/v1"
	"github.com/tmjwjx/supermarket/app/gateway/internal/auth"
	"github.com/tmjwjx/supermarket/app/gateway/internal/conf"

	"github.com/go-kratos/kratos/v3/middleware/recovery"
	"github.com/go-kratos/kratos/v3/transport/http"
)

type userProxy struct {
	users  userv1.UserServiceClient
	tokens *auth.Verifier
}

// 只开 HTTP 注册和登录公开 查用户先验本地 JWT
func NewHTTPServer(c *conf.Server, users userv1.UserServiceClient, tokens *auth.Verifier) *http.Server {
	var opts = []http.ServerOption{
		http.Middleware(
			recovery.Recovery(),
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
	p := &userProxy{users: users, tokens: tokens}
	r := srv.Route("/")
	// 公开
	r.POST("/v1/users/register", p.register)
	// 公开
	r.POST("/v1/users/login", p.login)
	// 要 token 验签失败不打到 user
	r.GET("/v1/users/{id}", p.requireToken(p.getUser))
	return srv
}

// 缺令牌或验签失败时拒绝 不进入后面的转发
func (p *userProxy) requireToken(next func(http.Context) error) func(http.Context) error {
	return func(ctx http.Context) error {
		if err := p.tokens.Verify(ctx.Request().Header.Get("Authorization")); err != nil {
			return err
		}
		return next(ctx)
	}
}

// 公开 请求体原样交给上游 Register
func (p *userProxy) register(ctx http.Context) error {
	var in userv1.RegisterRequest
	if err := ctx.Bind(&in); err != nil {
		return err
	}
	http.SetOperation(ctx, userv1.OperationUserServiceRegister)
	h := ctx.Middleware(func(c context.Context, req any) (any, error) {
		return p.users.Register(c, req.(*userv1.RegisterRequest))
	})
	out, err := h(ctx, &in)
	if err != nil {
		return err
	}
	return ctx.Result(200, out.(*userv1.RegisterResponse))
}

// 公开 请求体原样交给上游 Login
func (p *userProxy) login(ctx http.Context) error {
	var in userv1.LoginRequest
	if err := ctx.Bind(&in); err != nil {
		return err
	}
	http.SetOperation(ctx, userv1.OperationUserServiceLogin)
	h := ctx.Middleware(func(c context.Context, req any) (any, error) {
		return p.users.Login(c, req.(*userv1.LoginRequest))
	})
	out, err := h(ctx, &in)
	if err != nil {
		return err
	}
	return ctx.Result(200, out.(*userv1.LoginResponse))
}

// 验签通过后按路径 id 转发 GetUser
func (p *userProxy) getUser(ctx http.Context) error {
	var in userv1.GetUserRequest
	if err := ctx.BindQuery(&in); err != nil {
		return err
	}
	if err := ctx.BindVars(&in); err != nil {
		return err
	}
	http.SetOperation(ctx, userv1.OperationUserServiceGetUser)
	h := ctx.Middleware(func(c context.Context, req any) (any, error) {
		return p.users.GetUser(c, req.(*userv1.GetUserRequest))
	})
	out, err := h(ctx, &in)
	if err != nil {
		return err
	}
	return ctx.Result(200, out.(*userv1.GetUserResponse))
}
