package adminuser

import (
	"context"
	"strings"

	v1 "github.com/tmjwjx/supermarket/api/admin/v1"
	bizadmin "github.com/tmjwjx/supermarket/app/admin/internal/biz/adminuser"

	kmd "github.com/go-kratos/kratos/v3/metadata"
	"github.com/google/uuid"
)

type AdminUserService struct {
	v1.UnimplementedAdminUserServiceServer
	uc *bizadmin.AdminUserUsecase
}

func NewAdminUserService(uc *bizadmin.AdminUserUsecase) *AdminUserService {
	return &AdminUserService{uc: uc}
}

func (s *AdminUserService) Login(ctx context.Context, req *v1.LoginRequest) (*v1.LoginResponse, error) {
	if req == nil {
		return nil, bizadmin.ErrAdminInvalidArgument
	}
	token, err := s.uc.Login(ctx, req.GetUsername(), req.GetPassword())
	if err != nil {
		return nil, err
	}
	return &v1.LoginResponse{
		AccessToken: token.AccessToken,
		ExpiresInMs: token.ExpiresInMs,
		User:        toProto(token.User),
	}, nil
}

// 从元数据 x-md-global-admin-id 取当前运营账号
func (s *AdminUserService) GetMe(ctx context.Context, _ *v1.GetMeRequest) (*v1.GetMeResponse, error) {
	id, err := callerID(ctx)
	if err != nil {
		return nil, err
	}
	user, err := s.uc.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return &v1.GetMeResponse{User: toProto(user)}, nil
}

// 只有超级管理员能建账号 种子命令不走这个入口
func (s *AdminUserService) CreateAdminUser(ctx context.Context, req *v1.CreateAdminUserRequest) (*v1.CreateAdminUserResponse, error) {
	if err := requireSuper(ctx); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, bizadmin.ErrAdminInvalidArgument
	}
	created, err := s.uc.Create(ctx, &bizadmin.AdminUser{
		Username:    strings.TrimSpace(req.GetUsername()),
		DisplayName: strings.TrimSpace(req.GetDisplayName()),
		Role:        strings.TrimSpace(req.GetRole()),
	}, req.GetPassword())
	if err != nil {
		return nil, err
	}
	return &v1.CreateAdminUserResponse{User: toProto(created)}, nil
}

func (s *AdminUserService) ListAdminUsers(ctx context.Context, _ *v1.ListAdminUsersRequest) (*v1.ListAdminUsersResponse, error) {
	if err := requireSuper(ctx); err != nil {
		return nil, err
	}
	rows, err := s.uc.List(ctx)
	if err != nil {
		return nil, err
	}
	out := &v1.ListAdminUsersResponse{}
	for _, row := range rows {
		out.Users = append(out.Users, toProto(row))
	}
	return out, nil
}

func (s *AdminUserService) UpdateAdminUser(ctx context.Context, req *v1.UpdateAdminUserRequest) (*v1.UpdateAdminUserResponse, error) {
	if err := requireSuper(ctx); err != nil {
		return nil, err
	}
	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, bizadmin.ErrAdminInvalidArgument
	}
	updated, err := s.uc.Update(ctx, id, req.GetRole(), req.GetStatus())
	if err != nil {
		return nil, err
	}
	return &v1.UpdateAdminUserResponse{User: toProto(updated)}, nil
}

func (s *AdminUserService) ResetAdminPassword(ctx context.Context, req *v1.ResetAdminPasswordRequest) (*v1.ResetAdminPasswordResponse, error) {
	if err := requireSuper(ctx); err != nil {
		return nil, err
	}
	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, bizadmin.ErrAdminInvalidArgument
	}
	if err := s.uc.ResetPassword(ctx, id, req.GetPassword()); err != nil {
		return nil, err
	}
	return &v1.ResetAdminPasswordResponse{}, nil
}

func requireSuper(ctx context.Context) error {
	md, ok := kmd.FromServerContext(ctx)
	if !ok {
		return bizadmin.ErrAdminUnauthenticated
	}
	id := strings.TrimSpace(md.Get("x-md-global-admin-id"))
	role := strings.TrimSpace(md.Get("x-md-global-admin-role"))
	if id == "" || role == "" {
		return bizadmin.ErrAdminUnauthenticated
	}
	if _, err := uuid.Parse(id); err != nil {
		return bizadmin.ErrAdminUnauthenticated
	}
	if role != bizadmin.RoleSuper {
		return bizadmin.ErrAdminForbidden
	}
	return nil
}

func callerID(ctx context.Context) (uuid.UUID, error) {
	md, ok := kmd.FromServerContext(ctx)
	if !ok {
		return uuid.Nil, bizadmin.ErrAdminUnauthenticated
	}
	id, err := uuid.Parse(md.Get("x-md-global-admin-id"))
	if err != nil {
		return uuid.Nil, bizadmin.ErrAdminUnauthenticated
	}
	return id, nil
}

func toProto(u *bizadmin.AdminUser) *v1.AdminUser {
	if u == nil {
		return nil
	}
	return &v1.AdminUser{
		Id:          u.ID.String(),
		Username:    u.Username,
		DisplayName: u.DisplayName,
		Role:        u.Role,
		Status:      u.Status,
	}
}
