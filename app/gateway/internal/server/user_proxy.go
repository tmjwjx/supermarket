package server

import (
	userv1 "github.com/tmjwjx/supermarket/api/user/v1"

	"github.com/go-kratos/kratos/v3/transport/http"
)

type userProxy struct {
	gate
	users     userv1.UserServiceClient
	addresses userv1.AddressServiceClient
}

func (p *userProxy) routes(r *http.Router) {
	// 公开
	r.POST("/v1/users/register", p.register)
	// 公开
	r.POST("/v1/users/login", p.login)
	// 登录
	r.GET("/v1/users/me", p.requireLogin(p.getMe))
	// 登录 只转发 是否本人由 user 服务核对
	r.GET("/v1/users/{id}", p.requireLogin(p.getUser))
	// 登录
	r.POST("/v1/addresses", p.requireLogin(p.createAddress))
	// 登录
	r.GET("/v1/addresses", p.requireLogin(p.listAddresses))
	// 登录
	r.PATCH("/v1/addresses/{id}", p.requireLogin(p.updateAddress))
	// 登录
	r.POST("/v1/addresses/{id}:default", p.requireLogin(p.setDefaultAddress))
	// 登录
	r.DELETE("/v1/addresses/{id}", p.requireLogin(p.deleteAddress))
}

func (p *userProxy) register(ctx http.Context) error {
	var in userv1.RegisterRequest
	if err := ctx.Bind(&in); err != nil {
		return err
	}
	return call(ctx, userv1.OperationUserServiceRegister, &in, rpc(p.users.Register))
}

func (p *userProxy) login(ctx http.Context) error {
	var in userv1.LoginRequest
	if err := ctx.Bind(&in); err != nil {
		return err
	}
	return call(ctx, userv1.OperationUserServiceLogin, &in, rpc(p.users.Login))
}

func (p *userProxy) getMe(ctx http.Context) error {
	var in userv1.GetMeRequest
	return call(ctx, userv1.OperationUserServiceGetMe, &in, rpc(p.users.GetMe))
}

func (p *userProxy) getUser(ctx http.Context) error {
	var in userv1.GetUserRequest
	if err := ctx.BindVars(&in); err != nil {
		return err
	}
	return call(ctx, userv1.OperationUserServiceGetUser, &in, rpc(p.users.GetUser))
}

func (p *userProxy) createAddress(ctx http.Context) error {
	var in userv1.CreateAddressRequest
	if err := ctx.Bind(&in); err != nil {
		return err
	}
	return call(ctx, userv1.OperationAddressServiceCreateAddress, &in, rpc(p.addresses.CreateAddress))
}

func (p *userProxy) listAddresses(ctx http.Context) error {
	var in userv1.ListAddressesRequest
	return call(ctx, userv1.OperationAddressServiceListAddresses, &in, rpc(p.addresses.ListAddresses))
}

func (p *userProxy) updateAddress(ctx http.Context) error {
	var in userv1.UpdateAddressRequest
	if err := ctx.Bind(&in); err != nil {
		return err
	}
	if err := ctx.BindVars(&in); err != nil {
		return err
	}
	return call(ctx, userv1.OperationAddressServiceUpdateAddress, &in, rpc(p.addresses.UpdateAddress))
}

func (p *userProxy) setDefaultAddress(ctx http.Context) error {
	var in userv1.SetDefaultAddressRequest
	if err := bindBody(ctx, &in); err != nil {
		return err
	}
	if err := ctx.BindVars(&in); err != nil {
		return err
	}
	return call(ctx, userv1.OperationAddressServiceSetDefaultAddress, &in, rpc(p.addresses.SetDefaultAddress))
}

func (p *userProxy) deleteAddress(ctx http.Context) error {
	var in userv1.DeleteAddressRequest
	if err := ctx.BindVars(&in); err != nil {
		return err
	}
	return call(ctx, userv1.OperationAddressServiceDeleteAddress, &in, rpc(p.addresses.DeleteAddress))
}
