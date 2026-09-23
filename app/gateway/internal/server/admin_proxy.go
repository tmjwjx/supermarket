package server

import (
	adminv1 "github.com/tmjwjx/supermarket/api/admin/v1"

	"github.com/go-kratos/kratos/v3/transport/http"
)

type adminProxy struct {
	gate
	admins adminv1.AdminUserServiceClient
}

func (p *adminProxy) routes(r *http.Router) {
	// 公开
	r.POST("/v1/admin/login", p.login)
	// 后台
	r.GET("/v1/admin/me", p.requireAdmin(p.getMe))
	// 后台
	r.GET("/v1/admin/users", p.requireAdmin(p.listAdminUsers))
	// 后台
	r.POST("/v1/admin/users", p.requireAdmin(p.createAdminUser))
	// 后台
	r.PATCH("/v1/admin/users/{id}", p.requireAdmin(p.updateAdminUser))
	// 后台
	r.POST("/v1/admin/users/{id}:resetPassword", p.requireAdmin(p.resetAdminPassword))
}

func (p *adminProxy) login(ctx http.Context) error {
	var in adminv1.LoginRequest
	if err := ctx.Bind(&in); err != nil {
		return err
	}
	return call(ctx, adminv1.OperationAdminUserServiceLogin, &in, rpc(p.admins.Login))
}

func (p *adminProxy) getMe(ctx http.Context) error {
	var in adminv1.GetMeRequest
	return call(ctx, adminv1.OperationAdminUserServiceGetMe, &in, rpc(p.admins.GetMe))
}

func (p *adminProxy) listAdminUsers(ctx http.Context) error {
	var in adminv1.ListAdminUsersRequest
	return call(ctx, adminv1.OperationAdminUserServiceListAdminUsers, &in, rpc(p.admins.ListAdminUsers))
}

func (p *adminProxy) updateAdminUser(ctx http.Context) error {
	var in adminv1.UpdateAdminUserRequest
	if err := ctx.Bind(&in); err != nil {
		return err
	}
	if err := ctx.BindVars(&in); err != nil {
		return err
	}
	return call(ctx, adminv1.OperationAdminUserServiceUpdateAdminUser, &in, rpc(p.admins.UpdateAdminUser))
}

func (p *adminProxy) resetAdminPassword(ctx http.Context) error {
	var in adminv1.ResetAdminPasswordRequest
	if err := ctx.Bind(&in); err != nil {
		return err
	}
	if err := ctx.BindVars(&in); err != nil {
		return err
	}
	return call(ctx, adminv1.OperationAdminUserServiceResetAdminPassword, &in, rpc(p.admins.ResetAdminPassword))
}

func (p *adminProxy) createAdminUser(ctx http.Context) error {
	var in adminv1.CreateAdminUserRequest
	if err := ctx.Bind(&in); err != nil {
		return err
	}
	return call(ctx, adminv1.OperationAdminUserServiceCreateAdminUser, &in, rpc(p.admins.CreateAdminUser))
}
