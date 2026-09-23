package adminuser

import (
	"context"
	"errors"
	"time"

	bizadmin "github.com/tmjwjx/supermarket/app/admin/internal/biz/adminuser"

	"github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AdminUser struct {
	ID           string `gorm:"type:char(36);primaryKey"`
	Username     string `gorm:"size:64;uniqueIndex"`
	PasswordHash string `gorm:"size:72"`
	DisplayName  string `gorm:"size:64"`
	Role         string `gorm:"size:16"`
	Status       int32
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (AdminUser) TableName() string { return "admin_users" }

func (p *AdminUser) toBiz() *bizadmin.AdminUser {
	if p == nil {
		return nil
	}
	id, _ := uuid.Parse(p.ID)
	return &bizadmin.AdminUser{
		ID:          id,
		Username:    p.Username,
		DisplayName: p.DisplayName,
		Role:        p.Role,
		Status:      p.Status,
	}
}

type adminUserRepo struct {
	db *gorm.DB
}

func NewAdminUserRepo(db *gorm.DB) bizadmin.AdminUserRepo {
	return &adminUserRepo{db: db}
}

func (r *adminUserRepo) Create(ctx context.Context, u *bizadmin.AdminUser, password string) (*bizadmin.AdminUser, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, bizadmin.ErrAdminInvalidArgument
	}
	po := &AdminUser{
		ID:           id.String(),
		Username:     u.Username,
		PasswordHash: string(hash),
		DisplayName:  u.DisplayName,
		Role:         u.Role,
		Status:       bizadmin.StatusActive,
	}
	if err := r.db.WithContext(ctx).Create(po).Error; err != nil {
		if isUniqueViolation(err) {
			return nil, bizadmin.ErrAdminUsernameExists
		}
		return nil, err
	}
	return po.toBiz(), nil
}

func (r *adminUserRepo) FindByID(ctx context.Context, id uuid.UUID) (*bizadmin.AdminUser, error) {
	var po AdminUser
	if err := r.db.WithContext(ctx).First(&po, "id = ?", id.String()).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, bizadmin.ErrAdminNotFound
		}
		return nil, err
	}
	return po.toBiz(), nil
}

func (r *adminUserRepo) FindByUsername(ctx context.Context, username string) (*bizadmin.AdminUser, error) {
	var po AdminUser
	if err := r.db.WithContext(ctx).First(&po, "username = ?", username).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, bizadmin.ErrAdminNotFound
		}
		return nil, err
	}
	return po.toBiz(), nil
}

func (r *adminUserRepo) VerifyPassword(ctx context.Context, id uuid.UUID, password string) error {
	var po AdminUser
	if err := r.db.WithContext(ctx).Select("password_hash").First(&po, "id = ?", id.String()).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return bizadmin.ErrAdminInvalidCredentials
		}
		return err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(po.PasswordHash), []byte(password)); err != nil {
		return bizadmin.ErrAdminInvalidCredentials
	}
	return nil
}

func (r *adminUserRepo) List(ctx context.Context) ([]*bizadmin.AdminUser, error) {
	var rows []AdminUser
	if err := r.db.WithContext(ctx).Order("created_at asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*bizadmin.AdminUser, 0, len(rows))
	for i := range rows {
		out = append(out, rows[i].toBiz())
	}
	return out, nil
}

func (r *adminUserRepo) Update(ctx context.Context, id uuid.UUID, role string, status int32) (*bizadmin.AdminUser, error) {
	updates := map[string]any{}
	if role != "" {
		updates["role"] = role
	}
	if status != 0 {
		updates["status"] = status
	}
	res := r.db.WithContext(ctx).Model(&AdminUser{}).Where("id = ?", id.String()).Updates(updates)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, bizadmin.ErrAdminNotFound
	}
	return r.FindByID(ctx, id)
}

func (r *adminUserRepo) SetPassword(ctx context.Context, id uuid.UUID, password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return bizadmin.ErrAdminInvalidArgument
	}
	res := r.db.WithContext(ctx).Model(&AdminUser{}).Where("id = ?", id.String()).Update("password_hash", string(hash))
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return bizadmin.ErrAdminNotFound
	}
	return nil
}

func isUniqueViolation(err error) bool {
	var mysqlErr *mysql.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == 1062
}
