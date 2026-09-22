package user

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/tmjwjx/supermarket/app/user/internal/conf"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const testJWTSecret = "test-jwt-secret"

func newTestUserUsecase() (*UserUsecase, *fakeUserRepo) {
	repo := newFakeUserRepo()
	uc := NewUserUsecase(repo, &conf.Auth{
		JWTSecret:     testJWTSecret,
		TokenExpireMs: 604800000,
	})
	return uc, repo
}

type fakeUserRepo struct {
	users     map[uuid.UUID]*User
	byPhone   map[string]uuid.UUID
	passwords map[uuid.UUID]string
	verifyErr error
}

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{
		users:     make(map[uuid.UUID]*User),
		byPhone:   make(map[string]uuid.UUID),
		passwords: make(map[uuid.UUID]string),
	}
}

func (r *fakeUserRepo) Create(_ context.Context, user *User, password string) (*User, error) {
	if _, ok := r.byPhone[user.Phone]; ok {
		return nil, ErrUserPhoneAlreadyExists
	}
	now := time.Now()
	created := cloneUser(user)
	created.ID = uuid.Must(uuid.NewV7())
	created.Status = UserStatusActive
	created.CreatedAt = now
	created.UpdatedAt = now
	r.users[created.ID] = cloneUser(created)
	r.byPhone[created.Phone] = created.ID
	r.passwords[created.ID] = password
	return cloneUser(created), nil
}

func (r *fakeUserRepo) FindByID(_ context.Context, id uuid.UUID) (*User, error) {
	user, ok := r.users[id]
	if !ok {
		return nil, ErrUserNotFound
	}
	return cloneUser(user), nil
}

func (r *fakeUserRepo) FindByPhone(_ context.Context, phone string) (*User, error) {
	id, ok := r.byPhone[phone]
	if !ok {
		return nil, ErrUserNotFound
	}
	return cloneUser(r.users[id]), nil
}

func (r *fakeUserRepo) VerifyPassword(_ context.Context, id uuid.UUID, password string) error {
	if r.verifyErr != nil {
		return r.verifyErr
	}
	stored, ok := r.passwords[id]
	if !ok || stored != password {
		return ErrUserInvalidCredentials
	}
	return nil
}

func cloneUser(user *User) *User {
	if user == nil {
		return nil
	}
	clone := *user
	return &clone
}

func TestUserUsecaseRegisterLoginGetUser(t *testing.T) {
	ctx := context.Background()
	uc, _ := newTestUserUsecase()

	registered, err := uc.Register(ctx, "13800138000", "secret", "alice")
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if registered.User == nil || registered.User.Phone != "13800138000" || registered.User.Nickname != "alice" {
		t.Fatalf("Register() user = %+v", registered.User)
	}
	if registered.User.Status != UserStatusActive {
		t.Fatalf("Register() status = %v, want active", registered.User.Status)
	}
	if registered.AccessToken == "" || registered.ExpiresInMs != 604800000 {
		t.Fatalf("Register() token = %+v", registered)
	}
	assertJWTSubject(t, registered.AccessToken, registered.User.ID.String())

	loggedIn, err := uc.Login(ctx, "13800138000", "secret")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if loggedIn.User.ID != registered.User.ID {
		t.Fatalf("Login() id = %s, want %s", loggedIn.User.ID, registered.User.ID)
	}
	assertJWTSubject(t, loggedIn.AccessToken, registered.User.ID.String())

	got, err := uc.GetUser(ctx, registered.User.ID)
	if err != nil {
		t.Fatalf("GetUser() error = %v", err)
	}
	if got.Phone != "13800138000" || got.Nickname != "alice" {
		t.Fatalf("GetUser() = %+v", got)
	}
}

func TestUserUsecaseRegisterPhoneTaken(t *testing.T) {
	ctx := context.Background()
	uc, _ := newTestUserUsecase()

	if _, err := uc.Register(ctx, "13912345678", "secret", ""); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if _, err := uc.Register(ctx, "13912345678", "other", "bob"); !errors.Is(err, ErrUserPhoneAlreadyExists) {
		t.Fatalf("Register(duplicate) error = %v, want phone already exists", err)
	}
}

func TestUserUsecaseLoginBadPassword(t *testing.T) {
	ctx := context.Background()
	uc, _ := newTestUserUsecase()

	if _, err := uc.Register(ctx, "13700001111", "correct", ""); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if _, err := uc.Login(ctx, "13700001111", "wrong"); !errors.Is(err, ErrUserInvalidCredentials) {
		t.Fatalf("Login(bad password) error = %v, want invalid credentials", err)
	}
	if _, err := uc.Login(ctx, "13600002222", "correct"); !errors.Is(err, ErrUserInvalidCredentials) {
		t.Fatalf("Login(missing user) error = %v, want invalid credentials", err)
	}
}

func TestUserUsecaseGetUserNotFound(t *testing.T) {
	ctx := context.Background()
	uc, _ := newTestUserUsecase()

	if _, err := uc.GetUser(ctx, uuid.Must(uuid.NewV7())); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("GetUser(missing) error = %v, want not found", err)
	}
}

func TestUserUsecaseLoginPassesThroughRepoError(t *testing.T) {
	ctx := context.Background()
	uc, repo := newTestUserUsecase()

	if _, err := uc.Register(ctx, "13400004444", "secret", ""); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	repo.verifyErr = errors.New("db down")
	if _, err := uc.Login(ctx, "13400004444", "secret"); !errors.Is(err, repo.verifyErr) {
		t.Fatalf("Login(repo error) error = %v, want db down", err)
	}
}

func TestUserUsecaseLoginDisabled(t *testing.T) {
	ctx := context.Background()
	uc, repo := newTestUserUsecase()

	registered, err := uc.Register(ctx, "13500003333", "secret", "")
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	repo.users[registered.User.ID].Status = UserStatusDisabled

	if _, err := uc.Login(ctx, "13500003333", "secret"); !errors.Is(err, ErrUserDisabled) {
		t.Fatalf("Login(disabled) error = %v, want disabled", err)
	}
}

func TestUserUsecaseValidatePhone(t *testing.T) {
	ctx := context.Background()
	uc, _ := newTestUserUsecase()

	for _, phone := range []string{"23800138000", "1380013800", "138001380001", "+8613800138000", ""} {
		if _, err := uc.Register(ctx, phone, "secret", ""); !errors.Is(err, ErrUserInvalidArgument) {
			t.Fatalf("Register(%q) error = %v, want invalid argument", phone, err)
		}
	}
	if _, err := uc.Register(ctx, "13800138000", "", ""); !errors.Is(err, ErrUserInvalidArgument) {
		t.Fatalf("Register(empty password) error = %v, want invalid argument", err)
	}
}

func assertJWTSubject(t *testing.T, token, wantSubject string) {
	t.Helper()
	parsed, err := jwt.Parse(token, func(t *jwt.Token) (any, error) {
		return []byte(testJWTSecret), nil
	})
	if err != nil {
		t.Fatalf("jwt.Parse() error = %v", err)
	}
	sub, err := parsed.Claims.GetSubject()
	if err != nil {
		t.Fatalf("GetSubject() error = %v", err)
	}
	if sub != wantSubject {
		t.Fatalf("jwt subject = %q, want %q", sub, wantSubject)
	}
}
