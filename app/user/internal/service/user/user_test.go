package user

import (
	"context"
	"testing"
	"time"

	v1 "github.com/tmjwjx/supermarket/api/user/v1"
	bizuser "github.com/tmjwjx/supermarket/app/user/internal/biz/user"

	kratoserrors "github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
)

type fakeUserUsecase struct {
	register func(ctx context.Context, phone, password, nickname string) (*bizuser.AuthToken, error)
	login    func(ctx context.Context, phone, password string) (*bizuser.AuthToken, error)
	getUser  func(ctx context.Context, id uuid.UUID) (*bizuser.User, error)
}

func (f *fakeUserUsecase) Register(ctx context.Context, phone, password, nickname string) (*bizuser.AuthToken, error) {
	return f.register(ctx, phone, password, nickname)
}

func (f *fakeUserUsecase) Login(ctx context.Context, phone, password string) (*bizuser.AuthToken, error) {
	return f.login(ctx, phone, password)
}

func (f *fakeUserUsecase) GetUser(ctx context.Context, id uuid.UUID) (*bizuser.User, error) {
	return f.getUser(ctx, id)
}

func newUserDO(phone, nickname string) *bizuser.User {
	now := time.Now()
	return &bizuser.User{
		ID:        uuid.Must(uuid.NewV7()),
		Phone:     phone,
		Nickname:  nickname,
		Status:    bizuser.UserStatusActive,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func newAuthToken(user *bizuser.User) *bizuser.AuthToken {
	return &bizuser.AuthToken{
		AccessToken: "test-access-token",
		ExpiresInMs: 604800000,
		User:        user,
	}
}

func TestUserServiceRegisterLoginGetUser(t *testing.T) {
	ctx := context.Background()
	user := newUserDO("13800138000", "alice")
	token := newAuthToken(user)
	svc := &UserService{uc: &fakeUserUsecase{
		register: func(_ context.Context, phone, password, nickname string) (*bizuser.AuthToken, error) {
			if phone != "13800138000" || password != "secret" || nickname != "alice" {
				t.Fatalf("Register usecase args = (%q, %q, %q)", phone, password, nickname)
			}
			return token, nil
		},
		login: func(_ context.Context, phone, password string) (*bizuser.AuthToken, error) {
			if phone != "13800138000" || password != "secret" {
				t.Fatalf("Login usecase args = (%q, %q)", phone, password)
			}
			return token, nil
		},
		getUser: func(_ context.Context, id uuid.UUID) (*bizuser.User, error) {
			if id != user.ID {
				t.Fatalf("GetUser usecase id = %s, want %s", id, user.ID)
			}
			return user, nil
		},
	}}

	registered, err := svc.Register(ctx, &v1.RegisterRequest{
		Phone:    "13800138000",
		Password: "secret",
		Nickname: "alice",
	})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if registered.GetAccessToken() != "test-access-token" || registered.GetExpiresInMs() != 604800000 {
		t.Fatalf("Register() token = %+v", registered)
	}
	if registered.GetUser().GetId() != user.ID.String() || registered.GetUser().GetPhone() != "13800138000" {
		t.Fatalf("Register() user = %+v", registered.GetUser())
	}
	if registered.GetUser().GetStatus() != v1.UserStatus_USER_STATUS_ACTIVE {
		t.Fatalf("Register() status = %v, want active", registered.GetUser().GetStatus())
	}
	if registered.GetUser().GetCreatedAt() == nil || registered.GetUser().GetUpdatedAt() == nil {
		t.Fatal("Register() did not map timestamps")
	}

	loggedIn, err := svc.Login(ctx, &v1.LoginRequest{Phone: "13800138000", Password: "secret"})
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if loggedIn.GetUser().GetId() != user.ID.String() {
		t.Fatalf("Login() user = %+v", loggedIn.GetUser())
	}

	got, err := svc.GetUser(ctx, &v1.GetUserRequest{Id: user.ID.String()})
	if err != nil {
		t.Fatalf("GetUser() error = %v", err)
	}
	if got.GetUser().GetNickname() != "alice" {
		t.Fatalf("GetUser() = %+v", got.GetUser())
	}
}

func TestUserServiceRegisterPhoneTaken(t *testing.T) {
	ctx := context.Background()
	svc := &UserService{uc: &fakeUserUsecase{
		register: func(context.Context, string, string, string) (*bizuser.AuthToken, error) {
			return nil, bizuser.ErrUserPhoneAlreadyExists
		},
	}}
	if _, err := svc.Register(ctx, &v1.RegisterRequest{Phone: "13912345678", Password: "secret"}); !kratoserrors.IsConflict(err) {
		t.Fatalf("Register(taken) error = %v, want conflict", err)
	}
}

func TestUserServiceLoginBadPassword(t *testing.T) {
	ctx := context.Background()
	svc := &UserService{uc: &fakeUserUsecase{
		login: func(context.Context, string, string) (*bizuser.AuthToken, error) {
			return nil, bizuser.ErrUserInvalidCredentials
		},
	}}
	if _, err := svc.Login(ctx, &v1.LoginRequest{Phone: "13700001111", Password: "wrong"}); !kratoserrors.IsUnauthorized(err) {
		t.Fatalf("Login(bad password) error = %v, want unauthorized", err)
	}
}

func TestUserServiceGetUserNotFound(t *testing.T) {
	ctx := context.Background()
	svc := &UserService{uc: &fakeUserUsecase{
		getUser: func(context.Context, uuid.UUID) (*bizuser.User, error) {
			return nil, bizuser.ErrUserNotFound
		},
	}}
	if _, err := svc.GetUser(ctx, &v1.GetUserRequest{Id: uuid.Must(uuid.NewV7()).String()}); !kratoserrors.IsNotFound(err) {
		t.Fatalf("GetUser(missing) error = %v, want not found", err)
	}
}

func TestUserServiceValidation(t *testing.T) {
	ctx := context.Background()
	svc := &UserService{uc: &fakeUserUsecase{}}

	if _, err := svc.Register(ctx, &v1.RegisterRequest{Phone: "23800138000", Password: "secret"}); !kratoserrors.IsBadRequest(err) {
		t.Fatalf("Register(bad phone) error = %v, want bad request", err)
	}
	if _, err := svc.Register(ctx, &v1.RegisterRequest{Phone: "13800138000", Password: ""}); !kratoserrors.IsBadRequest(err) {
		t.Fatalf("Register(empty password) error = %v, want bad request", err)
	}
	if _, err := svc.Login(ctx, &v1.LoginRequest{Phone: "1380013800", Password: "secret"}); !kratoserrors.IsBadRequest(err) {
		t.Fatalf("Login(bad phone) error = %v, want bad request", err)
	}
	if _, err := svc.GetUser(ctx, &v1.GetUserRequest{Id: "not-a-uuid"}); !kratoserrors.IsBadRequest(err) {
		t.Fatalf("GetUser(malformed id) error = %v, want bad request", err)
	}
}
