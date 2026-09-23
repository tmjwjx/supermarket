package httpauthtest

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tmjwjx/supermarket/pkg/httpauth"
)

// Route 是一条要验证的 HTTP 路由
type Route struct {
	Method string
	Path   string
}

// RejectsForged 只带伪造身份头不带令牌直连服务 HTTP 端口必须得到 401
func RejectsForged(t *testing.T, handler http.Handler, routes ...Route) {
	t.Helper()
	for _, rt := range routes {
		req := httptest.NewRequest(rt.Method, rt.Path, strings.NewReader(`{}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set(httpauth.MetaUserID, "0195f7a0-0000-7000-8000-000000000001")
		req.Header.Set(httpauth.MetaAdminID, "0195f7a0-0000-7000-8000-000000000002")
		req.Header.Set(httpauth.MetaAdminRole, "super")
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)
		if res.Code != http.StatusUnauthorized {
			t.Errorf("%s %s status %d want 401 body %s", rt.Method, rt.Path, res.Code, res.Body.String())
		}
	}
}
