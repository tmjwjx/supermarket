package auth

import (
	"github.com/tmjwjx/supermarket/app/gateway/internal/conf"
	"github.com/tmjwjx/supermarket/pkg/httpauth"

	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(NewVerifier, NewAdminVerifier)

type Verifier struct {
	secret []byte
}

func NewVerifier(a *conf.Auth) *Verifier {
	var secret string
	if a != nil {
		secret = a.JWTSecret
	}
	return &Verifier{secret: []byte(secret)}
}

// 只接受带非空 sub 的 HS256 Bearer 令牌 成功返回 sub
func (v *Verifier) Verify(header string) (string, error) {
	return httpauth.VerifyUser(header, v.secret)
}

type AdminVerifier struct {
	secret []byte
}

func NewAdminVerifier(a *conf.Auth) *AdminVerifier {
	var secret string
	if a != nil {
		secret = a.AdminJWTSecret
	}
	return &AdminVerifier{secret: []byte(secret)}
}

// 后台令牌必须是 aud 为 admin 的 HS256 成功返回运营账号 id 和角色
func (v *AdminVerifier) Verify(header string) (string, string, error) {
	return httpauth.VerifyAdmin(header, v.secret)
}
