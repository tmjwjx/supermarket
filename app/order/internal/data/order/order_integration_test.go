//go:build integration

package order_test

import (
	"context"
	"testing"
	"time"

	bizorder "github.com/tmjwjx/supermarket/app/order/internal/biz/order"
	dataorder "github.com/tmjwjx/supermarket/app/order/internal/data/order"
	"github.com/tmjwjx/supermarket/pkg/kafkaout"
	"github.com/tmjwjx/supermarket/pkg/testdb"

	"github.com/google/uuid"
)

// legacyOrder 是加标记列之前的订单表
type legacyOrder struct {
	ID             string `gorm:"type:char(36);primaryKey"`
	UserID         string `gorm:"size:36"`
	RequestID      string `gorm:"size:64"`
	Status         int32
	StockConfirmed bool
	PaidAt         *time.Time
	ExpiresAt      time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (legacyOrder) TableName() string { return "orders" }

// 存量订单加列后是 false 仍在未确认扫描里 落了终态标记之后被排除
func TestListUnconfirmedSkipsConfirmFailed(t *testing.T) {
	ctx := context.Background()
	db := testdb.Open(t, "order_it_confirm_failed", &legacyOrder{})
	paidAt := time.Now()
	oldID := uuid.Must(uuid.NewV7())
	if err := db.Create(&legacyOrder{
		ID: oldID.String(), UserID: uuid.NewString(), RequestID: "r-old", Status: bizorder.StatusPaid, PaidAt: &paidAt, ExpiresAt: paidAt,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&dataorder.Order{}, &dataorder.OrderItem{}, &kafkaout.Row{}); err != nil {
		t.Fatal(err)
	}
	repo := dataorder.NewOrderRepo(db)
	rows, err := repo.ListUnconfirmed(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].ID != oldID || rows[0].StockConfirmFailed {
		t.Fatalf("rows %+v", rows)
	}
	if err := repo.MarkStockConfirmFailed(ctx, oldID); err != nil {
		t.Fatal(err)
	}
	rows, err = repo.ListUnconfirmed(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 0 {
		t.Fatalf("flagged order still listed %+v", rows)
	}
	got, err := repo.Find(ctx, oldID)
	if err != nil {
		t.Fatal(err)
	}
	if !got.StockConfirmFailed {
		t.Fatalf("order %+v", got)
	}
}
