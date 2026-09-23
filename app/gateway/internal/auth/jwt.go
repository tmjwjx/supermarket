package auth

import (
	"strings"

	"github.com/tmjwjx/supermarket/app/gateway/internal/conf"

	kerrors "github.com/go-kratos/kratos/v3/errors"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(NewVerifier)

var errUnauthorized = kerrors.Unauthorized("GATEWAY_UNAUTHORIZED", "unauthorized")

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

// 只接受带非空 sub 的 HS256 Bearer 令牌 失败则拒绝
func (v *Verifier) Verify(header string) error {
	token, ok := bearerToken(header)
	if !ok || len(v.secret) == 0 {
		return errUnauthorized
	}
	parsed, err := jwt.ParseWithClaims(token, &jwt.RegisteredClaims{}, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, errUnauthorized
		}
		return v.secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil || !parsed.Valid {
		return errUnauthorized
	}
	claims, ok := parsed.Claims.(*jwt.RegisteredClaims)
	if !ok || claims.Subject == "" {
		return errUnauthorized
	}
	return nil
}

func bearerToken(header string) (string, bool) {
	fields := strings.Fields(header)
	if len(fields) != 2 || !strings.EqualFold(fields[0], "Bearer") {
		return "", false
	}
	return fields[1], true
}
