package review

import (
	"context"
	"time"

	bizreview "github.com/tmjwjx/supermarket/app/product/internal/biz/review"
	"github.com/tmjwjx/supermarket/app/product/internal/data/sqlerr"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Review 是评价表 一个订单项只能有一条
type Review struct {
	ID          string    `gorm:"type:varchar(36);primaryKey"`
	ProductID   string    `gorm:"type:varchar(36);not null;index"`
	OrderItemID string    `gorm:"type:varchar(36);not null;uniqueIndex"`
	UserID      string    `gorm:"type:varchar(36);not null;index"`
	Stars       int32     `gorm:"not null"`
	Content     string    `gorm:"size:2000"`
	DisplayName string    `gorm:"size:64"`
	Anonymous   bool      `gorm:"not null"`
	SpecsJSON   string    `gorm:"size:1024"`
	CreatedAt   time.Time `gorm:"index"`
	UpdatedAt   time.Time
}

type reviewRepo struct {
	db *gorm.DB
}

func NewReviewRepo(db *gorm.DB) bizreview.ReviewRepo {
	return &reviewRepo{db: db}
}

func (r *reviewRepo) Add(ctx context.Context, in *bizreview.Review) (*bizreview.Review, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	po := &Review{
		ID:          id.String(),
		ProductID:   in.ProductID,
		OrderItemID: in.OrderItemID,
		UserID:      in.UserID,
		Stars:       in.Stars,
		Content:     in.Content,
		DisplayName: in.DisplayName,
		Anonymous:   in.Anonymous,
		SpecsJSON:   in.SpecsJSON,
	}
	if err := r.db.WithContext(ctx).Create(po).Error; err != nil {
		if sqlerr.IsUnique(err) {
			return nil, bizreview.ErrExists
		}
		return nil, err
	}
	in.ID = po.ID
	return in, nil
}

func (r *reviewRepo) List(ctx context.Context, productID string, size, offset int) ([]*bizreview.Review, bool, error) {
	if size <= 0 {
		size = 20
	}
	var rows []Review
	err := r.db.WithContext(ctx).Where("product_id = ?", productID).
		Order("created_at desc").Offset(offset).Limit(size + 1).Find(&rows).Error
	if err != nil {
		return nil, false, err
	}
	hasMore := len(rows) > size
	if hasMore {
		rows = rows[:size]
	}
	out := make([]*bizreview.Review, 0, len(rows))
	for _, row := range rows {
		out = append(out, &bizreview.Review{
			ID:          row.ID,
			ProductID:   row.ProductID,
			OrderItemID: row.OrderItemID,
			UserID:      row.UserID,
			DisplayName: row.DisplayName,
			Content:     row.Content,
			Stars:       row.Stars,
			Anonymous:   row.Anonymous,
			SpecsJSON:   row.SpecsJSON,
		})
	}
	return out, hasMore, nil
}
