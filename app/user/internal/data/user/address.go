package user

import (
	"context"
	"errors"
	"time"

	bizuser "github.com/tmjwjx/supermarket/app/user/internal/biz/user"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Address struct {
	ID        string `gorm:"type:char(36);primaryKey"`
	UserID    string `gorm:"type:char(36);index"`
	Receiver  string `gorm:"size:64"`
	Phone     string `gorm:"size:11"`
	Province  string `gorm:"size:32"`
	City      string `gorm:"size:32"`
	District  string `gorm:"size:32"`
	Detail    string `gorm:"size:255"`
	IsDefault bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (Address) TableName() string { return "addresses" }

type addressRepo struct{ db *gorm.DB }

func NewAddressRepo(db *gorm.DB) bizuser.AddressRepo { return &addressRepo{db: db} }

func (r *addressRepo) Create(ctx context.Context, in *bizuser.Address) (*bizuser.Address, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	po := &Address{
		ID: id.String(), UserID: in.UserID.String(), Receiver: in.Receiver, Phone: in.Phone,
		Province: in.Province, City: in.City, District: in.District, Detail: in.Detail, IsDefault: in.IsDefault,
	}
	if err := r.db.WithContext(ctx).Create(po).Error; err != nil {
		return nil, err
	}
	return po.toBiz(), nil
}

func (r *addressRepo) List(ctx context.Context, userID uuid.UUID) ([]*bizuser.Address, error) {
	var rows []Address
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID.String()).Order("created_at desc").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*bizuser.Address, 0, len(rows))
	for i := range rows {
		out = append(out, rows[i].toBiz())
	}
	return out, nil
}

func (r *addressRepo) Delete(ctx context.Context, userID, id uuid.UUID) error {
	res := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", id.String(), userID.String()).Delete(&Address{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return bizuser.ErrUserNotFound
	}
	return nil
}

func (r *addressRepo) Get(ctx context.Context, userID, id uuid.UUID) (*bizuser.Address, error) {
	var po Address
	err := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", id.String(), userID.String()).First(&po).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, bizuser.ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return po.toBiz(), nil
}

func (r *addressRepo) Count(ctx context.Context, userID uuid.UUID) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&Address{}).Where("user_id = ?", userID.String()).Count(&n).Error
	return n, err
}

func (r *addressRepo) ClearDefault(ctx context.Context, userID uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&Address{}).Where("user_id = ?", userID.String()).Update("is_default", false).Error
}

func (r *addressRepo) Update(ctx context.Context, in *bizuser.Address) (*bizuser.Address, error) {
	res := r.db.WithContext(ctx).Model(&Address{}).Where("id = ? AND user_id = ?", in.ID.String(), in.UserID.String()).Updates(map[string]any{
		"receiver": in.Receiver, "phone": in.Phone, "province": in.Province,
		"city": in.City, "district": in.District, "detail": in.Detail,
	})
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, bizuser.ErrUserNotFound
	}
	return r.Get(ctx, in.UserID, in.ID)
}

func (r *addressRepo) SetDefault(ctx context.Context, userID, id uuid.UUID) (*bizuser.Address, error) {
	var out *bizuser.Address
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var po Address
		err := tx.Where("id = ? AND user_id = ?", id.String(), userID.String()).First(&po).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return bizuser.ErrUserNotFound
		}
		if err != nil {
			return err
		}
		if err := tx.Model(&Address{}).Where("user_id = ?", userID.String()).Update("is_default", false).Error; err != nil {
			return err
		}
		if err := tx.Model(&Address{}).Where("id = ? AND user_id = ?", id.String(), userID.String()).Update("is_default", true).Error; err != nil {
			return err
		}
		po.IsDefault = true
		out = po.toBiz()
		return nil
	})
	return out, err
}

func (p *Address) toBiz() *bizuser.Address {
	id, _ := uuid.Parse(p.ID)
	uid, _ := uuid.Parse(p.UserID)
	return &bizuser.Address{
		ID: id, UserID: uid, Receiver: p.Receiver, Phone: p.Phone, Province: p.Province,
		City: p.City, District: p.District, Detail: p.Detail, IsDefault: p.IsDefault,
		CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt,
	}
}
