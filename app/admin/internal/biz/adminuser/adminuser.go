package adminuser

import (
	"context"
	"errors"
	"strings"
	"time"

	v1 "github.com/tmjwjx/supermarket/api/admin/v1"
	"github.com/tmjwjx/supermarket/app/admin/internal/conf"

	kerrors "github.com/go-kratos/kratos/v3/errors"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	RoleSuper   = "super"
	RoleProduct = "product"
	RoleOrder   = "order"

	StatusActive   int32 = 1
	StatusDisabled int32 = 2
)

var (
	ErrAdminNotFound           = kerrors.NotFound("ADMIN_NOT_FOUND", "admin not found")
	ErrAdminInvalidArgument    = kerrors.BadRequest(v1.ErrorReason_ADMIN_INVALID_ARGUMENT.String(), "invalid admin argument")
	ErrAdminInvalidCredentials = kerrors.Unauthorized(v1.ErrorReason_ADMIN_INVALID_CREDENTIALS.String(), "invalid credentials")
	ErrAdminUnauthenticated    = kerrors.Unauthorized(v1.ErrorReason_ADMIN_INVALID_CREDENTIALS.String(), "unauthenticated")
	ErrAdminDisabled           = kerrors.Forbidden(v1.ErrorReason_ADMIN_DISABLED.String(), "admin disabled")
	ErrAdminUsernameExists     = kerrors.Conflict("ADMIN_USERNAME_EXISTS", "username already exists")
	ErrAdminForbidden          = kerrors.Forbidden(v1.ErrorReason_ADMIN_FORBIDDEN.String(), "forbidden")
)

type AdminUser struct {
	ID          uuid.UUID
	Username    string
	DisplayName string
	Role        string
	Status      int32
}

type AuthToken struct {
	AccessToken string
	ExpiresInMs int64
	User        *AdminUser
}

type adminClaims struct {
	Role string `json:"role"`
	jwt.RegisteredClaims
}

type AdminUserRepo interface {
	Create(ctx context.Context, user *AdminUser, password string) (*AdminUser, error)
	FindByID(ctx context.Context, id uuid.UUID) (*AdminUser, error)
	FindByUsername(ctx context.Context, username string) (*AdminUser, error)
	VerifyPassword(ctx context.Context, id uuid.UUID, password string) error
	List(ctx context.Context) ([]*AdminUser, error)
	Update(ctx context.Context, id uuid.UUID, role string, status int32) (*AdminUser, error)
	SetPassword(ctx context.Context, id uuid.UUID, password string) error
}

type AdminUserUsecase struct {
	repo   AdminUserRepo
	secret []byte
	ttl    time.Duration
}

func NewAdminUserUsecase(repo AdminUserRepo, auth *conf.Auth) *AdminUserUsecase {
	var a conf.Auth
	if auth != nil {
		a = *auth
	}
	return &AdminUserUsecase{repo: repo, secret: []byte(a.JWTSecret), ttl: a.TokenExpire()}
}

func (uc *AdminUserUsecase) Create(ctx context.Context, user *AdminUser, password string) (*AdminUser, error) {
	if user == nil {
		return nil, ErrAdminInvalidArgument
	}
	user.Username = strings.TrimSpace(user.Username)
	user.DisplayName = strings.TrimSpace(user.DisplayName)
	if user.Username == "" || len(user.Username) > 64 || password == "" || !validRole(user.Role) {
		return nil, ErrAdminInvalidArgument
	}
	if user.DisplayName == "" {
		user.DisplayName = user.Username
	}
	if len(user.DisplayName) > 64 {
		return nil, ErrAdminInvalidArgument
	}
	user.Status = StatusActive
	return uc.repo.Create(ctx, user, password)
}

// 用户名不存在和密码错误返回同一错误 停用账号拒绝登录
func (uc *AdminUserUsecase) Login(ctx context.Context, username, password string) (*AuthToken, error) {
	username = strings.TrimSpace(username)
	if username == "" || password == "" {
		return nil, ErrAdminInvalidArgument
	}
	user, err := uc.repo.FindByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, ErrAdminNotFound) {
			return nil, ErrAdminInvalidCredentials
		}
		return nil, err
	}
	if user.Status == StatusDisabled {
		return nil, ErrAdminDisabled
	}
	if err := uc.repo.VerifyPassword(ctx, user.ID, password); err != nil {
		return nil, err
	}
	return uc.issueToken(user)
}

func (uc *AdminUserUsecase) Get(ctx context.Context, id uuid.UUID) (*AdminUser, error) {
	if id == uuid.Nil {
		return nil, ErrAdminInvalidArgument
	}
	return uc.repo.FindByID(ctx, id)
}

func (uc *AdminUserUsecase) List(ctx context.Context) ([]*AdminUser, error) {
	return uc.repo.List(ctx)
}

// Update 改角色或停用 空角色和 0 状态表示保持原值
func (uc *AdminUserUsecase) Update(ctx context.Context, id uuid.UUID, role string, status int32) (*AdminUser, error) {
	if id == uuid.Nil {
		return nil, ErrAdminInvalidArgument
	}
	role = strings.TrimSpace(role)
	if role != "" && !validRole(role) {
		return nil, ErrAdminInvalidArgument
	}
	if status != 0 && status != StatusActive && status != StatusDisabled {
		return nil, ErrAdminInvalidArgument
	}
	if role == "" && status == 0 {
		return nil, ErrAdminInvalidArgument
	}
	return uc.repo.Update(ctx, id, role, status)
}

func (uc *AdminUserUsecase) ResetPassword(ctx context.Context, id uuid.UUID, password string) error {
	if id == uuid.Nil || password == "" {
		return ErrAdminInvalidArgument
	}
	return uc.repo.SetPassword(ctx, id, password)
}

// 用户名已存在则跳过 角色固定为超级管理员
func (uc *AdminUserUsecase) Seed(ctx context.Context, username, password string) (bool, error) {
	_, err := uc.repo.FindByUsername(ctx, username)
	if err == nil {
		return false, nil
	}
	if !errors.Is(err, ErrAdminNotFound) {
		return false, err
	}
	_, err = uc.Create(ctx, &AdminUser{Username: username, DisplayName: username, Role: RoleSuper}, password)
	if errors.Is(err, ErrAdminUsernameExists) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// 签发 aud 为 admin 且带角色的 HS256 令牌 有效期 12 小时
func (uc *AdminUserUsecase) issueToken(user *AdminUser) (*AuthToken, error) {
	now := time.Now()
	ttl := uc.ttl
	if ttl <= 0 {
		ttl = 12 * time.Hour
	}
	expires := now.Add(ttl)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, adminClaims{
		Role: user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID.String(),
			Audience:  jwt.ClaimStrings{"admin"},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expires),
		},
	})
	signed, err := token.SignedString(uc.secret)
	if err != nil {
		return nil, err
	}
	return &AuthToken{AccessToken: signed, ExpiresInMs: ttl.Milliseconds(), User: user}, nil
}

func validRole(role string) bool {
	switch role {
	case RoleSuper, RoleProduct, RoleOrder:
		return true
	default:
		return false
	}
}
