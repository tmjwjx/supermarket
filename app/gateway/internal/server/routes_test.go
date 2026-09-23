package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/tmjwjx/supermarket/app/gateway/internal/auth"
	"github.com/tmjwjx/supermarket/app/gateway/internal/conf"

	"github.com/golang-jwt/jwt/v5"
)

func TestRouteAuth(t *testing.T) {
	handler := NewHTTPServer(&conf.Server{}, Services{
		Tokens:      auth.NewVerifier(&conf.Auth{JWTSecret: "user-secret"}),
		AdminTokens: auth.NewAdminVerifier(&conf.Auth{AdminJWTSecret: "admin-secret"}),
	})
	user := bearer(t, "user-secret", jwt.RegisteredClaims{
		Subject:   "user-1",
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	})
	productAdmin := adminBearer(t, "product")
	orderAdmin := adminBearer(t, "order")

	cases := []struct {
		method string
		path   string
		header string
		want   int
		panic  bool
	}{
		{http.MethodGet, "/v1/products", "", 0, true},
		{http.MethodGet, "/v1/products:search", "", 0, true},
		{http.MethodGet, "/v1/products/p1/detail", "", 0, true},
		{http.MethodGet, "/v1/products/p1", "", 0, true},
		{http.MethodGet, "/v1/brands", "", 0, true},
		{http.MethodGet, "/v1/categories", "", 0, true},
		{http.MethodGet, "/v1/recommendations", "", 0, true},
		{http.MethodGet, "/v1/products/p1/reviews", "", 0, true},
		{http.MethodGet, "/v1/stocks", "", 0, true},
		{http.MethodGet, "/v1/users/me", "", http.StatusUnauthorized, false},
		{http.MethodGet, "/v1/users/user-1", user, 0, true},
		{http.MethodGet, "/v1/orders", "", http.StatusUnauthorized, false},
		{http.MethodPost, "/v1/orders/o1:cancel", "", http.StatusUnauthorized, false},
		{http.MethodPost, "/v1/orders/o1:confirm", user, 0, true},
		{http.MethodPost, "/v1/payments/pay1:simulate", "", http.StatusUnauthorized, false},
		{http.MethodGet, "/v1/payments/pay1", user, 0, true},
		{http.MethodPost, "/v1/notifications:read", "", http.StatusUnauthorized, false},
		{http.MethodPost, "/v1/admin/login", "", 0, true},
		{http.MethodGet, "/v1/admin/me", "", http.StatusUnauthorized, false},
		{http.MethodGet, "/v1/admin/products", productAdmin, 0, true},
		{http.MethodGet, "/v1/admin/products", orderAdmin, http.StatusForbidden, false},
		{http.MethodPost, "/v1/admin/orders/o1:ship", orderAdmin, 0, true},
		{http.MethodPost, "/v1/admin/orders/o1:ship", productAdmin, http.StatusForbidden, false},
		{http.MethodPost, "/v1/admin/users", orderAdmin, http.StatusForbidden, false},
	}
	for _, tc := range cases {
		code, panicked := do(handler, tc.method, tc.path, tc.header)
		if tc.panic {
			if !panicked {
				t.Errorf("%s %s status %d 未进入转发", tc.method, tc.path, code)
			}
			continue
		}
		if panicked || code != tc.want {
			t.Errorf("%s %s got status %d panic %v", tc.method, tc.path, code, panicked)
		}
	}
}

func TestAdminCatalogRoutes(t *testing.T) {
	handler := NewHTTPServer(&conf.Server{}, Services{
		Tokens:      auth.NewVerifier(&conf.Auth{JWTSecret: "user-secret"}),
		AdminTokens: auth.NewAdminVerifier(&conf.Auth{AdminJWTSecret: "admin-secret"}),
	})
	productAdmin := adminBearer(t, "product")
	orderAdmin := adminBearer(t, "order")
	routes := []struct{ method, path string }{
		{http.MethodGet, "/v1/admin/products/p1"},
		{http.MethodGet, "/v1/admin/categories"},
		{http.MethodPatch, "/v1/admin/categories/c1"},
		{http.MethodPatch, "/v1/admin/attributes/a1"},
		{http.MethodGet, "/v1/admin/recommendations"},
		{http.MethodPatch, "/v1/admin/recommendations/r1"},
		{http.MethodDelete, "/v1/admin/recommendations/r1"},
		{http.MethodGet, "/v1/admin/stocks?sku_ids=a&sku_ids=b"},
		{http.MethodPatch, "/v1/admin/brands/b1"},
		{http.MethodDelete, "/v1/admin/brands/b1"},
		{http.MethodPost, "/v1/admin/products:batchPublish"},
		{http.MethodPost, "/v1/admin/products/p1:purge"},
		{http.MethodPut, "/v1/admin/products/p1"},
		{http.MethodGet, "/v1/admin/product-logs"},
	}
	for _, rt := range routes {
		if code, panicked := do(handler, rt.method, rt.path, ""); panicked || code != http.StatusUnauthorized {
			t.Errorf("%s %s 无令牌 got %d panic %v", rt.method, rt.path, code, panicked)
		}
		if code, panicked := do(handler, rt.method, rt.path, orderAdmin); panicked || code != http.StatusForbidden {
			t.Errorf("%s %s 订单运营 got %d panic %v", rt.method, rt.path, code, panicked)
		}
		if code, panicked := do(handler, rt.method, rt.path, productAdmin); !panicked {
			t.Errorf("%s %s 商品运营 status %d 未进入转发", rt.method, rt.path, code)
		}
	}
}

func do(handler http.Handler, method, path, header string) (code int, panicked bool) {
	defer func() {
		if recover() != nil {
			panicked = true
		}
	}()
	req := httptest.NewRequest(method, path, strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	if header != "" {
		req.Header.Set("Authorization", header)
	}
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	return res.Code, false
}

func bearer(t *testing.T, secret string, claims jwt.Claims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatal(err)
	}
	return "Bearer " + signed
}

func adminBearer(t *testing.T, role string) string {
	t.Helper()
	return bearer(t, "admin-secret", jwt.MapClaims{
		"sub":  "admin-1",
		"aud":  "admin",
		"role": role,
		"exp":  time.Now().Add(time.Hour).Unix(),
	})
}
