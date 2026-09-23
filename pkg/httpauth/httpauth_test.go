package httpauth

import (
	"context"
	"encoding/json"
	nethttp "net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v3/metadata"
	mmd "github.com/go-kratos/kratos/v3/middleware/metadata"
	khttp "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/golang-jwt/jwt/v5"
)

const (
	userSecret  = "user-secret"
	adminSecret = "admin-secret"
)

// 用和生成代码相同的方式挂路由 回显服务端元数据里的身份
func newServer() *khttp.Server {
	srv := khttp.NewServer(khttp.Middleware(mmd.Server(), Server(Options{
		UserSecret:  userSecret,
		AdminSecret: adminSecret,
		Public:      []string{"/test/Public", "/test/AdminLogin"},
		Optional:    []string{"/test/Optional"},
	})))
	r := srv.Route("/")
	add := func(method, path, op string) {
		r.Handle(method, path, func(ctx khttp.Context) error {
			khttp.SetOperation(ctx, op)
			h := ctx.Middleware(func(c context.Context, _ any) (any, error) {
				md, _ := metadata.FromServerContext(c)
				return map[string]string{
					"user":  md.Get(MetaUserID),
					"admin": md.Get(MetaAdminID),
					"role":  md.Get(MetaAdminRole),
					"trace": md.Get("x-md-global-trace"),
				}, nil
			})
			out, err := h(ctx, nil)
			if err != nil {
				return err
			}
			return ctx.Result(200, out)
		})
	}
	add(nethttp.MethodGet, "/v1/public", "/test/Public")
	add(nethttp.MethodGet, "/v1/optional", "/test/Optional")
	add(nethttp.MethodGet, "/v1/buyer", "/test/Buyer")
	add(nethttp.MethodPost, "/v1/admin/login", "/test/AdminLogin")
	add(nethttp.MethodGet, "/v1/admin/products", "/test/AdminProducts")
	add(nethttp.MethodGet, "/v1/admin/orders", "/test/AdminOrders")
	return srv
}

func sign(t *testing.T, secret string, claims jwt.Claims) string {
	t.Helper()
	s, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	if err != nil {
		t.Fatal(err)
	}
	return "Bearer " + s
}

func userToken(t *testing.T, sub string) string {
	return sign(t, userSecret, jwt.RegisteredClaims{Subject: sub, ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))})
}

func adminToken(t *testing.T, secret, aud, role string) string {
	return sign(t, secret, jwt.MapClaims{"sub": "admin-1", "aud": aud, "role": role, "exp": time.Now().Add(time.Hour).Unix()})
}

func forged(req *nethttp.Request) {
	req.Header.Set(MetaUserID, "forged-user")
	req.Header.Set(MetaAdminID, "forged-admin")
	req.Header.Set(MetaAdminRole, "super")
	req.Header.Set("x-md-global-trace", "t1")
}

func TestServer(t *testing.T) {
	srv := newServer()
	cases := []struct {
		name   string
		method string
		path   string
		auth   string
		want   int
		user   string
		admin  string
	}{
		{"公开路由不带令牌也放行且不带伪造身份", nethttp.MethodGet, "/v1/public", "", 200, "", ""},
		{"买家路由只有伪造头被拒", nethttp.MethodGet, "/v1/buyer", "", 401, "", ""},
		{"买家路由用错密钥被拒", nethttp.MethodGet, "/v1/buyer", sign(t, "other", jwt.RegisteredClaims{Subject: "u1"}), 401, "", ""},
		{"买家路由令牌身份覆盖伪造头", nethttp.MethodGet, "/v1/buyer", userToken(t, "u1"), 200, "u1", ""},
		{"可选路由无令牌放行但丢掉伪造头", nethttp.MethodGet, "/v1/optional", "", 200, "", ""},
		{"可选路由带令牌写入用户", nethttp.MethodGet, "/v1/optional", userToken(t, "u2"), 200, "u2", ""},
		{"后台登录公开", nethttp.MethodPost, "/v1/admin/login", "", 200, "", ""},
		{"后台路由只有伪造头被拒", nethttp.MethodGet, "/v1/admin/products", "", 401, "", ""},
		{"后台路由不接受买家令牌", nethttp.MethodGet, "/v1/admin/products", userToken(t, "u1"), 401, "", ""},
		{"后台路由要求 aud 为 admin", nethttp.MethodGet, "/v1/admin/products", adminToken(t, adminSecret, "web", "super"), 401, "", ""},
		{"后台路由按路径查角色", nethttp.MethodGet, "/v1/admin/products", adminToken(t, adminSecret, "admin", "order"), 403, "", ""},
		{"订单后台给 order 角色", nethttp.MethodGet, "/v1/admin/orders", adminToken(t, adminSecret, "admin", "order"), 200, "", "admin-1"},
		{"商品后台给 product 角色", nethttp.MethodGet, "/v1/admin/products", adminToken(t, adminSecret, "admin", "product"), 200, "", "admin-1"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			forged(req)
			if tc.auth != "" {
				req.Header.Set("Authorization", tc.auth)
			}
			res := httptest.NewRecorder()
			srv.ServeHTTP(res, req)
			if res.Code != tc.want {
				t.Fatalf("status %d want %d body %s", res.Code, tc.want, res.Body.String())
			}
			if tc.want != 200 {
				return
			}
			var got map[string]string
			if err := json.Unmarshal(res.Body.Bytes(), &got); err != nil {
				t.Fatal(err)
			}
			if got["user"] != tc.user || got["admin"] != tc.admin {
				t.Fatalf("identity %v", got)
			}
			if got["trace"] != "t1" {
				t.Fatalf("非身份元数据应保留 %v", got)
			}
		})
	}
}
