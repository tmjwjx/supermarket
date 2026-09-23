package httpauth

import (
	"context"
	"strings"

	kerrors "github.com/go-kratos/kratos/v3/errors"
	"github.com/go-kratos/kratos/v3/metadata"
	"github.com/go-kratos/kratos/v3/middleware"
	"github.com/go-kratos/kratos/v3/transport"
	khttp "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/golang-jwt/jwt/v5"
)

const (
	MetaUserID    = "x-md-global-user-id"
	MetaAdminID   = "x-md-global-admin-id"
	MetaAdminRole = "x-md-global-admin-role"
)

var (
	ErrUnauthorized = kerrors.Unauthorized("UNAUTHORIZED", "unauthorized")
	ErrForbidden    = kerrors.Forbidden("FORBIDDEN", "forbidden")
)

// Options 描述一个服务 HTTP 端口的验签规则 未列出的非后台操作都要求买家登录
type Options struct {
	UserSecret  string
	AdminSecret string
	Public      []string
	Optional    []string
}

type adminClaims struct {
	jwt.RegisteredClaims
	Role string `json:"role"`
}

// Server 丢弃客户端自带的身份头 只把验签通过的身份写进服务端元数据
func Server(o Options) middleware.Middleware {
	public := toSet(o.Public)
	optional := toSet(o.Optional)
	userSecret := []byte(o.UserSecret)
	adminSecret := []byte(o.AdminSecret)
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			tr, ok := transport.FromServerContext(ctx)
			if !ok || tr.Kind() != transport.KindHTTP {
				return handler(ctx, req)
			}
			md := trusted(ctx)
			op := tr.Operation()
			header := tr.RequestHeader().Get("Authorization")
			path := ""
			if r, ok := khttp.RequestFromServerContext(ctx); ok && r != nil {
				path = r.URL.Path
			}
			switch {
			case public[op]:
			case strings.HasPrefix(path, "/v1/admin/"):
				id, role, err := VerifyAdmin(header, adminSecret)
				if err != nil {
					return nil, err
				}
				if !AllowAdmin(path, role) {
					return nil, ErrForbidden
				}
				md.Set(MetaAdminID, id)
				md.Set(MetaAdminRole, role)
			default:
				sub, err := VerifyUser(header, userSecret)
				if err == nil {
					md.Set(MetaUserID, sub)
				} else if !optional[op] {
					return nil, err
				}
			}
			return handler(metadata.NewServerContext(ctx, md), req)
		}
	}
}

// 保留其他透传元数据 去掉三个身份键
func trusted(ctx context.Context) metadata.Metadata {
	out := metadata.New()
	md, ok := metadata.FromServerContext(ctx)
	if !ok {
		return out
	}
	md.Range(func(k string, v []string) bool {
		switch strings.ToLower(k) {
		case MetaUserID, MetaAdminID, MetaAdminRole:
		default:
			out[k] = append([]string(nil), v...)
		}
		return true
	})
	return out
}

// AllowAdmin 查自己谁都能看 订单和支付给 order 与 super 运营账号管理只给 super 其余后台给 product 与 super
func AllowAdmin(path, role string) bool {
	switch {
	case path == "/v1/admin/me":
		return role == "product" || role == "order" || role == "super"
	case strings.HasPrefix(path, "/v1/admin/orders"), strings.HasPrefix(path, "/v1/admin/payments"):
		return role == "order" || role == "super"
	case strings.HasPrefix(path, "/v1/admin/users"):
		return role == "super"
	default:
		return role == "product" || role == "super"
	}
}

// VerifyUser 只接受带非空 sub 的 HS256 Bearer 令牌 成功返回 sub
func VerifyUser(header string, secret []byte) (string, error) {
	claims := &jwt.RegisteredClaims{}
	if err := parseHS256(header, secret, claims); err != nil {
		return "", err
	}
	if claims.Subject == "" {
		return "", ErrUnauthorized
	}
	return claims.Subject, nil
}

// VerifyAdmin 后台令牌必须是 aud 为 admin 的 HS256 成功返回运营账号 id 和角色
func VerifyAdmin(header string, secret []byte) (string, string, error) {
	claims := &adminClaims{}
	if err := parseHS256(header, secret, claims); err != nil {
		return "", "", err
	}
	if claims.Subject == "" || claims.Role == "" || !hasAudience(claims.Audience, "admin") {
		return "", "", ErrUnauthorized
	}
	return claims.Subject, claims.Role, nil
}

func parseHS256(header string, secret []byte, claims jwt.Claims) error {
	token, ok := bearerToken(header)
	if !ok || len(secret) == 0 {
		return ErrUnauthorized
	}
	parsed, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (any, error) {
		return secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil || !parsed.Valid {
		return ErrUnauthorized
	}
	return nil
}

func hasAudience(aud jwt.ClaimStrings, want string) bool {
	for _, item := range aud {
		if item == want {
			return true
		}
	}
	return false
}

func bearerToken(header string) (string, bool) {
	fields := strings.Fields(header)
	if len(fields) != 2 || !strings.EqualFold(fields[0], "Bearer") {
		return "", false
	}
	return fields[1], true
}

func toSet(items []string) map[string]bool {
	out := make(map[string]bool, len(items))
	for _, item := range items {
		out[item] = true
	}
	return out
}
