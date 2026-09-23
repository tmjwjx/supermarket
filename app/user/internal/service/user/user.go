package user

import (
	"context"
	"strings"

	v1 "github.com/tmjwjx/supermarket/api/user/v1"
	bizuser "github.com/tmjwjx/supermarket/app/user/internal/biz/user"

	kmetadata "github.com/go-kratos/kratos/v3/metadata"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// 测试缝：service 测假用例 不碰真实 biz
type userUsecase interface {
	Register(ctx context.Context, phone, password, nickname string) (*bizuser.AuthToken, error)
	Login(ctx context.Context, phone, password string) (*bizuser.AuthToken, error)
	GetUser(ctx context.Context, id uuid.UUID) (*bizuser.User, error)
}

type UserService struct {
	v1.UnimplementedUserServiceServer

	uc userUsecase
}

func NewUserService(uc *bizuser.UserUsecase) *UserService {
	return &UserService{uc: uc}
}

// 注册并返回令牌
func (s *UserService) Register(ctx context.Context, req *v1.RegisterRequest) (*v1.RegisterResponse, error) {
	user, password := convertRegister(req)
	if user == nil {
		return nil, bizuser.ErrUserInvalidArgument
	}
	if err := bizuser.ValidatePhone(user.Phone); err != nil {
		return nil, err
	}
	if password == "" {
		return nil, bizuser.ErrUserInvalidArgument
	}
	token, err := s.uc.Register(ctx, user.Phone, password, user.Nickname)
	if err != nil {
		return nil, err
	}
	return &v1.RegisterResponse{
		AccessToken: token.AccessToken,
		ExpiresInMs: token.ExpiresInMs,
		User:        convertUserReply(token.User),
	}, nil
}

// 登录并返回令牌
func (s *UserService) Login(ctx context.Context, req *v1.LoginRequest) (*v1.LoginResponse, error) {
	phone, password := convertLogin(req)
	if err := bizuser.ValidatePhone(phone); err != nil {
		return nil, err
	}
	if password == "" {
		return nil, bizuser.ErrUserInvalidArgument
	}
	token, err := s.uc.Login(ctx, phone, password)
	if err != nil {
		return nil, err
	}
	return &v1.LoginResponse{
		AccessToken: token.AccessToken,
		ExpiresInMs: token.ExpiresInMs,
		User:        convertUserReply(token.User),
	}, nil
}

// 按 id 取用户
func (s *UserService) GetUser(ctx context.Context, req *v1.GetUserRequest) (*v1.GetUserResponse, error) {
	id, err := parseUserID(req.GetId())
	if err != nil {
		return nil, err
	}
	caller, err := callerID(ctx)
	if err != nil {
		return nil, err
	}
	if caller != id {
		return nil, bizuser.ErrUserForbidden
	}
	user, err := s.uc.GetUser(ctx, id)
	if err != nil {
		return nil, err
	}
	return &v1.GetUserResponse{User: convertUserReply(user)}, nil
}

func (s *UserService) GetMe(ctx context.Context, _ *v1.GetMeRequest) (*v1.GetMeResponse, error) {
	caller, err := callerID(ctx)
	if err != nil {
		return nil, err
	}
	user, err := s.uc.GetUser(ctx, caller)
	if err != nil {
		return nil, err
	}
	return &v1.GetMeResponse{User: convertUserReply(user)}, nil
}

func callerID(ctx context.Context) (uuid.UUID, error) {
	md, ok := kmetadata.FromServerContext(ctx)
	if !ok {
		return uuid.Nil, bizuser.ErrUserInvalidCredentials
	}
	return parseUserID(md.Get("x-md-global-user-id"))
}

// 解析用户 id
func parseUserID(id string) (uuid.UUID, error) {
	parsed, err := uuid.Parse(id)
	if err != nil {
		return uuid.Nil, bizuser.ErrUserInvalidArgument
	}
	return parsed, nil
}

// 解析注册请求
func convertRegister(in *v1.RegisterRequest) (*bizuser.User, string) {
	if in == nil {
		return nil, ""
	}
	return &bizuser.User{
		Phone:    strings.TrimSpace(in.GetPhone()),
		Nickname: strings.TrimSpace(in.GetNickname()),
	}, in.GetPassword()
}

// 解析登录请求
func convertLogin(in *v1.LoginRequest) (string, string) {
	if in == nil {
		return "", ""
	}
	return strings.TrimSpace(in.GetPhone()), in.GetPassword()
}

// 领域用户转 proto
func convertUserReply(in *bizuser.User) *v1.User {
	if in == nil {
		return nil
	}
	return &v1.User{
		Id:        in.ID.String(),
		Phone:     in.Phone,
		Nickname:  in.Nickname,
		AvatarUrl: in.AvatarURL,
		Status:    convertUserStatus(in.Status),
		CreatedAt: timestamppb.New(in.CreatedAt),
		UpdatedAt: timestamppb.New(in.UpdatedAt),
	}
}

// 领域状态转 proto
func convertUserStatus(in bizuser.UserStatus) v1.UserStatus {
	switch in {
	case bizuser.UserStatusActive:
		return v1.UserStatus_USER_STATUS_ACTIVE
	case bizuser.UserStatusDisabled:
		return v1.UserStatus_USER_STATUS_DISABLED
	default:
		return v1.UserStatus_USER_STATUS_UNSPECIFIED
	}
}
