package server

import (
	"context"

	adminv1 "github.com/tmjwjx/supermarket/api/admin/v1"
	inventoryv1 "github.com/tmjwjx/supermarket/api/inventory/v1"
	notificationv1 "github.com/tmjwjx/supermarket/api/notification/v1"
	orderv1 "github.com/tmjwjx/supermarket/api/order/v1"
	paymentv1 "github.com/tmjwjx/supermarket/api/payment/v1"
	productv1 "github.com/tmjwjx/supermarket/api/product/v1"
	userv1 "github.com/tmjwjx/supermarket/api/user/v1"
	"github.com/tmjwjx/supermarket/app/gateway/internal/auth"
	"github.com/tmjwjx/supermarket/pkg/httpauth"

	kerrors "github.com/go-kratos/kratos/v3/errors"
	"github.com/go-kratos/kratos/v3/metadata"
	"github.com/go-kratos/kratos/v3/transport/http"
	"google.golang.org/grpc"
)

const (
	metaUserID    = "x-md-global-user-id"
	metaAdminID   = "x-md-global-admin-id"
	metaAdminRole = "x-md-global-admin-role"
)

var errForbidden = kerrors.Forbidden("GATEWAY_FORBIDDEN", "forbidden")

// 上游客户端和两套验签器 由 Wire 按类型填入
type Services struct {
	Users           userv1.UserServiceClient
	Addresses       userv1.AddressServiceClient
	Products        productv1.ProductServiceClient
	Brands          productv1.BrandServiceClient
	Categories      productv1.CategoryServiceClient
	Favorites       productv1.FavoriteServiceClient
	Histories       productv1.BrowseHistoryServiceClient
	Reviews         productv1.ReviewServiceClient
	Recommendations productv1.RecommendationServiceClient
	Attributes      productv1.AttributeServiceClient
	Stocks          inventoryv1.StockServiceClient
	Carts           orderv1.CartServiceClient
	Orders          orderv1.OrderServiceClient
	Payments        paymentv1.PaymentServiceClient
	Notifications   notificationv1.NotificationServiceClient
	Admins          adminv1.AdminUserServiceClient
	Tokens          *auth.Verifier
	AdminTokens     *auth.AdminVerifier
}

type gate struct {
	tokens *auth.Verifier
	admins *auth.AdminVerifier
}

// 登录模式 验签失败返回 401 成功则写入用户 id
func (g gate) requireLogin(next func(http.Context) error) func(http.Context) error {
	return func(ctx http.Context) error {
		sub, err := g.tokens.Verify(ctx.Request().Header.Get("Authorization"))
		if err != nil {
			return err
		}
		stamp(ctx, metaUserID, sub)
		return next(ctx)
	}
}

// 可选模式 令牌有效才写入用户 id 没有或无效也继续转发
func (g gate) optionalLogin(next func(http.Context) error) func(http.Context) error {
	return func(ctx http.Context) error {
		sub, err := g.tokens.Verify(ctx.Request().Header.Get("Authorization"))
		if err == nil && sub != "" {
			stamp(ctx, metaUserID, sub)
		}
		return next(ctx)
	}
}

// 后台模式 先验签再按路径检查角色 角色不符返回 403
func (g gate) requireAdmin(next func(http.Context) error) func(http.Context) error {
	return func(ctx http.Context) error {
		id, role, err := g.admins.Verify(ctx.Request().Header.Get("Authorization"))
		if err != nil {
			return err
		}
		if !allowAdmin(ctx.Request().URL.Path, role) {
			return errForbidden
		}
		stamp(ctx, metaAdminID, id, metaAdminRole, role)
		return next(ctx)
	}
}

// 和各服务 HTTP 端口共用同一套路径角色规则
func allowAdmin(path, role string) bool {
	return httpauth.AllowAdmin(path, role)
}

func stamp(ctx http.Context, kv ...string) {
	req := ctx.Request()
	*req = *req.WithContext(metadata.AppendToClientContext(req.Context(), kv...))
}

func call(ctx http.Context, op string, in any, fn func(context.Context, any) (any, error)) error {
	http.SetOperation(ctx, op)
	h := ctx.Middleware(func(c context.Context, req any) (any, error) {
		c = forwardIdentity(ctx, c)
		return fn(c, req)
	})
	out, err := h(ctx, in)
	if err != nil {
		return err
	}
	return ctx.Result(200, out)
}

func forwardIdentity(httpCtx http.Context, rpcCtx context.Context) context.Context {
	md, ok := metadata.FromClientContext(httpCtx.Request().Context())
	if !ok || len(md) == 0 {
		return rpcCtx
	}
	kv := make([]string, 0, len(md)*2)
	md.Range(func(k string, v []string) bool {
		if len(v) > 0 {
			kv = append(kv, k, v[0])
		}
		return true
	})
	if len(kv) == 0 {
		return rpcCtx
	}
	return metadata.AppendToClientContext(rpcCtx, kv...)
}

func rpc[Req any, Resp any](fn func(context.Context, *Req, ...grpc.CallOption) (*Resp, error)) func(context.Context, any) (any, error) {
	return func(ctx context.Context, req any) (any, error) {
		return fn(ctx, req.(*Req))
	}
}
