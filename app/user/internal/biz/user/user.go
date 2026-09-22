package user

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"

	v1 "github.com/tmjwjx/supermarket/api/user/v1"
	"github.com/tmjwjx/supermarket/app/user/internal/conf"

	kerrors "github.com/go-kratos/kratos/v3/errors"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// 类型错误：API 错误码带上 HTTP/gRPC 语义 上层原样返回
var (
	ErrUserNotFound           = kerrors.NotFound(v1.ErrorReason_USER_NOT_FOUND.String(), "user not found")
	ErrUserInvalidArgument    = kerrors.BadRequest(v1.ErrorReason_USER_INVALID_ARGUMENT.String(), "invalid user argument")
	ErrUserPhoneAlreadyExists = kerrors.Conflict(v1.ErrorReason_USER_PHONE_ALREADY_EXISTS.String(), "phone already exists")
	ErrUserInvalidCredentials = kerrors.Unauthorized(v1.ErrorReason_USER_INVALID_CREDENTIALS.String(), "invalid credentials")
	ErrUserDisabled           = kerrors.Forbidden(v1.ErrorReason_USER_DISABLED.String(), "user disabled")
)

// 大陆 11 位手机号：1 开头后跟 10 位数字
var mainlandMobile = regexp.MustCompile(`^1\d{10}$`)

// 与存储列和 proto 枚举对齐 0 未指定 1 正常 2 停用
type UserStatus int32

const (
	UserStatusUnspecified UserStatus = 0
	UserStatusActive      UserStatus = 1
	UserStatusDisabled    UserStatus = 2
)

// 领域对象 密码哈希不出现在这一层
type User struct {
	ID        uuid.UUID
	Phone     string
	Nickname  string
	AvatarURL string
	Status    UserStatus
	CreatedAt time.Time
	UpdatedAt time.Time
}

type AuthToken struct {
	AccessToken string
	ExpiresInMs int64
	User        *User
}

// 用户存取与验密
type UserRepo interface {
	Create(ctx context.Context, user *User, password string) (*User, error)
	FindByID(ctx context.Context, id uuid.UUID) (*User, error)
	FindByPhone(ctx context.Context, phone string) (*User, error)
	VerifyPassword(ctx context.Context, id uuid.UUID, password string) error
}

type UserUsecase struct {
	repo          UserRepo
	jwtSecret     []byte
	tokenExpireMs int64
}

// 装配用例与 JWT 配置
func NewUserUsecase(repo UserRepo, auth *conf.Auth) *UserUsecase {
	var a conf.Auth
	if auth != nil {
		a = *auth
	}
	return &UserUsecase{
		repo:          repo,
		jwtSecret:     []byte(a.JWTSecret),
		tokenExpireMs: a.TokenExpireMilliseconds(),
	}
}

// 校验手机号与非空密码 建号并签发 JWT
func (uc *UserUsecase) Register(ctx context.Context, phone, password, nickname string) (*AuthToken, error) {
	phone = strings.TrimSpace(phone)
	if err := ValidatePhone(phone); err != nil {
		return nil, err
	}
	if password == "" {
		return nil, ErrUserInvalidArgument
	}
	created, err := uc.repo.Create(ctx, &User{
		Phone:    phone,
		Nickname: strings.TrimSpace(nickname),
		Status:   UserStatusActive,
	}, password)
	if err != nil {
		return nil, err
	}
	return uc.issueToken(created)
}

// 按手机号登录并签发 JWT
func (uc *UserUsecase) Login(ctx context.Context, phone, password string) (*AuthToken, error) {
	phone = strings.TrimSpace(phone)
	if err := ValidatePhone(phone); err != nil {
		return nil, err
	}
	if password == "" {
		return nil, ErrUserInvalidArgument
	}
	user, err := uc.repo.FindByPhone(ctx, phone)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, ErrUserInvalidCredentials
		}
		return nil, err
	}
	if user.Status == UserStatusDisabled {
		return nil, ErrUserDisabled
	}
	if err := uc.repo.VerifyPassword(ctx, user.ID, password); err != nil {
		return nil, err
	}
	return uc.issueToken(user)
}

// 按 id 取用户
func (uc *UserUsecase) GetUser(ctx context.Context, id uuid.UUID) (*User, error) {
	if id == uuid.Nil {
		return nil, ErrUserInvalidArgument
	}
	return uc.repo.FindByID(ctx, id)
}

// 只认大陆 11 位手机号
func ValidatePhone(phone string) error {
	if !mainlandMobile.MatchString(phone) {
		return ErrUserInvalidArgument
	}
	return nil
}

// 签发只含 user id 与过期时间的 JWT
func (uc *UserUsecase) issueToken(user *User) (*AuthToken, error) {
	now := time.Now()
	expires := now.Add(time.Duration(uc.tokenExpireMs) * time.Millisecond)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Subject:   user.ID.String(),
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(expires),
	})
	signed, err := token.SignedString(uc.jwtSecret)
	if err != nil {
		return nil, err
	}
	return &AuthToken{
		AccessToken: signed,
		ExpiresInMs: uc.tokenExpireMs,
		User:        user,
	}, nil
}
