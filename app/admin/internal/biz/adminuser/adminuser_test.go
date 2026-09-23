package adminuser

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/tmjwjx/supermarket/app/admin/internal/conf"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type fakeRepo struct {
	byName map[string]*AdminUser
	byID   map[uuid.UUID]*AdminUser
	pass   map[uuid.UUID]string
}

func (f *fakeRepo) Create(_ context.Context, user *AdminUser, password string) (*AdminUser, error) {
	if f.byName == nil {
		f.byName = map[string]*AdminUser{}
		f.byID = map[uuid.UUID]*AdminUser{}
		f.pass = map[uuid.UUID]string{}
	}
	if _, ok := f.byName[user.Username]; ok {
		return nil, ErrAdminUsernameExists
	}
	id := uuid.Must(uuid.NewV7())
	cp := *user
	cp.ID = id
	cp.Status = StatusActive
	f.byName[cp.Username] = &cp
	f.byID[id] = &cp
	f.pass[id] = password
	out := cp
	return &out, nil
}

func (f *fakeRepo) FindByID(_ context.Context, id uuid.UUID) (*AdminUser, error) {
	u, ok := f.byID[id]
	if !ok {
		return nil, ErrAdminNotFound
	}
	cp := *u
	return &cp, nil
}

func (f *fakeRepo) FindByUsername(_ context.Context, username string) (*AdminUser, error) {
	u, ok := f.byName[username]
	if !ok {
		return nil, ErrAdminNotFound
	}
	cp := *u
	return &cp, nil
}

func (f *fakeRepo) VerifyPassword(_ context.Context, id uuid.UUID, password string) error {
	if f.pass[id] != password {
		return ErrAdminInvalidCredentials
	}
	return nil
}

func (f *fakeRepo) List(context.Context) ([]*AdminUser, error) {
	out := make([]*AdminUser, 0, len(f.byID))
	for _, user := range f.byID {
		cp := *user
		out = append(out, &cp)
	}
	return out, nil
}

func (f *fakeRepo) Update(_ context.Context, id uuid.UUID, role string, status int32) (*AdminUser, error) {
	user, ok := f.byID[id]
	if !ok {
		return nil, ErrAdminNotFound
	}
	if role != "" {
		user.Role = role
	}
	if status != 0 {
		user.Status = status
	}
	cp := *user
	return &cp, nil
}

func (f *fakeRepo) SetPassword(_ context.Context, id uuid.UUID, password string) error {
	if _, ok := f.byID[id]; !ok {
		return ErrAdminNotFound
	}
	f.pass[id] = password
	return nil
}

func TestLoginSameErrorDisabledAndClaims(t *testing.T) {
	repo := &fakeRepo{}
	uc := NewAdminUserUsecase(repo, &conf.Auth{JWTSecret: "change-me-admin-jwt-secret", TokenExpireMs: (12 * time.Hour).Milliseconds()})
	if _, err := uc.Login(context.Background(), "admin", "nope"); !errors.Is(err, ErrAdminInvalidCredentials) {
		t.Fatalf("missing: %v", err)
	}
	if _, err := repo.Create(context.Background(), &AdminUser{Username: "admin", DisplayName: "admin", Role: RoleSuper, Status: StatusActive}, "admin123"); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.Login(context.Background(), "admin", "wrong"); !errors.Is(err, ErrAdminInvalidCredentials) {
		t.Fatalf("wrong password: %v", err)
	}
	repo.byName["admin"].Status = StatusDisabled
	if _, err := uc.Login(context.Background(), "admin", "admin123"); !errors.Is(err, ErrAdminDisabled) {
		t.Fatalf("disabled: %v", err)
	}
	repo.byName["admin"].Status = StatusActive
	token, err := uc.Login(context.Background(), "admin", "admin123")
	if err != nil {
		t.Fatal(err)
	}
	if token.ExpiresInMs != (12 * time.Hour).Milliseconds() {
		t.Fatalf("ttl %d", token.ExpiresInMs)
	}
	parsed, err := jwt.ParseWithClaims(token.AccessToken, &adminClaims{}, func(tok *jwt.Token) (any, error) {
		if tok.Method != jwt.SigningMethodHS256 {
			t.Fatalf("method %v", tok.Method)
		}
		return []byte("change-me-admin-jwt-secret"), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	claims := parsed.Claims.(*adminClaims)
	if claims.Role != RoleSuper || claims.Subject != token.User.ID.String() {
		t.Fatalf("claims %+v", claims)
	}
	aud, err := claims.GetAudience()
	if err != nil || len(aud) != 1 || aud[0] != "admin" {
		t.Fatalf("aud %v err %v", aud, err)
	}
}

func TestSeedSkipsExisting(t *testing.T) {
	repo := &fakeRepo{}
	uc := NewAdminUserUsecase(repo, &conf.Auth{JWTSecret: "s"})
	created, err := uc.Seed(context.Background(), "admin", "admin123")
	if err != nil || !created {
		t.Fatalf("created %v err %v", created, err)
	}
	if repo.byName["admin"].Role != RoleSuper {
		t.Fatalf("role %s", repo.byName["admin"].Role)
	}
	again, err := uc.Seed(context.Background(), "admin", "other")
	if err != nil || again {
		t.Fatalf("again %v err %v", again, err)
	}
}
