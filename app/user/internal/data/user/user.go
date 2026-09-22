package user

import (
	"context"
	"errors"
	"time"

	bizuser "github.com/tmjwjx/supermarket/app/user/internal/biz/user"

	"github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// 存储形态：默认表名 users Phone 唯一 PasswordHash 不出本层 长度对齐 bcrypt 72 字节上限
type User struct {
	ID           string `gorm:"type:char(36);primaryKey"`
	Phone        string `gorm:"size:11;uniqueIndex"`
	PasswordHash string `gorm:"size:72"`
	Nickname     string `gorm:"size:64"`
	AvatarURL    string `gorm:"size:512"`
	Status       int32  `gorm:"index"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// 领域对象转存储记录
func newUser(u *bizuser.User) *User {
	return &User{
		ID:        u.ID.String(),
		Phone:     u.Phone,
		Nickname:  u.Nickname,
		AvatarURL: u.AvatarURL,
		Status:    int32(u.Status),
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}

// 存储记录转领域对象
func (p *User) toBiz() *bizuser.User {
	if p == nil {
		return nil
	}
	id, _ := uuid.Parse(p.ID)
	return &bizuser.User{
		ID:        id,
		Phone:     p.Phone,
		Nickname:  p.Nickname,
		AvatarURL: p.AvatarURL,
		Status:    bizuser.UserStatus(p.Status),
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
	}
}

type userRepo struct {
	db *gorm.DB
}

// 用共享 *gorm.DB 建仓库
func NewUserRepo(db *gorm.DB) bizuser.UserRepo {
	return &userRepo{db: db}
}

// UUID v7 建号 明文 bcrypt 后入库 状态固定 Active
func (r *userRepo) Create(ctx context.Context, u *bizuser.User, password string) (*bizuser.User, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, bizuser.ErrUserInvalidArgument
	}
	u.ID = id
	u.Status = bizuser.UserStatusActive
	po := newUser(u)
	po.PasswordHash = string(hash)
	if err := r.db.WithContext(ctx).Create(po).Error; err != nil {
		if isUniqueViolation(err) {
			return nil, bizuser.ErrUserPhoneAlreadyExists
		}
		return nil, err
	}
	return po.toBiz(), nil
}

// 按 id 取用户
func (r *userRepo) FindByID(ctx context.Context, id uuid.UUID) (*bizuser.User, error) {
	var po User
	if err := r.db.WithContext(ctx).First(&po, "id = ?", id.String()).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, bizuser.ErrUserNotFound
		}
		return nil, err
	}
	return po.toBiz(), nil
}

// 按手机号取用户
func (r *userRepo) FindByPhone(ctx context.Context, phone string) (*bizuser.User, error) {
	var po User
	if err := r.db.WithContext(ctx).First(&po, "phone = ?", phone).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, bizuser.ErrUserNotFound
		}
		return nil, err
	}
	return po.toBiz(), nil
}

// 按 id 核对明文密码
func (r *userRepo) VerifyPassword(ctx context.Context, id uuid.UUID, password string) error {
	var po User
	if err := r.db.WithContext(ctx).Select("password_hash").First(&po, "id = ?", id.String()).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return bizuser.ErrUserInvalidCredentials
		}
		return err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(po.PasswordHash), []byte(password)); err != nil {
		return bizuser.ErrUserInvalidCredentials
	}
	return nil
}

func isUniqueViolation(err error) bool {
	var mysqlErr *mysql.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == 1062
}
