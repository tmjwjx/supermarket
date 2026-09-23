package principal

import (
	"context"
	"strings"

	kerrors "github.com/go-kratos/kratos/v3/errors"
	kmetadata "github.com/go-kratos/kratos/v3/metadata"
)

// ErrUnauthorized 表示请求里没有用户或运营身份
var ErrUnauthorized = kerrors.Unauthorized("UNAUTHORIZED", "login required")

// ErrForbidden 表示运营角色不能做这件事
var ErrForbidden = kerrors.Forbidden("FORBIDDEN", "forbidden")

// UserID 从元数据 x-md-global-user-id 取当前用户 没有则 401
func UserID(ctx context.Context) (string, error) {
	id := optional(ctx)
	if id == "" {
		return "", ErrUnauthorized
	}
	return id, nil
}

// OptionalUserID 有用户 id 就返回 没有则空串
func OptionalUserID(ctx context.Context) string {
	return optional(ctx)
}

func optional(ctx context.Context) string {
	md, ok := kmetadata.FromServerContext(ctx)
	if !ok {
		return ""
	}
	return strings.TrimSpace(md.Get("x-md-global-user-id"))
}

// RequireCatalogAdmin 要求商品运营或超级管理员 没有身份 401 角色不对 403
func RequireCatalogAdmin(ctx context.Context) (string, error) {
	md, ok := kmetadata.FromServerContext(ctx)
	if !ok {
		return "", ErrUnauthorized
	}
	id := strings.TrimSpace(md.Get("x-md-global-admin-id"))
	role := strings.TrimSpace(md.Get("x-md-global-admin-role"))
	if id == "" || role == "" {
		return "", ErrUnauthorized
	}
	if role != "product" && role != "super" {
		return "", ErrForbidden
	}
	return id, nil
}
