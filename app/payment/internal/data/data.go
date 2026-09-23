package data

import (
	"time"

	"github.com/tmjwjx/supermarket/app/payment/internal/conf"
	datapayment "github.com/tmjwjx/supermarket/app/payment/internal/data/payment"
	"github.com/tmjwjx/supermarket/pkg/kafkaout"

	"github.com/go-kratos/kratos/v3/log"
	_ "github.com/go-sql-driver/mysql"
	"github.com/google/wire"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var ProviderSet = wire.NewSet(NewData, NewDB, datapayment.NewPaymentRepo, kafkaout.NewStore)

func NewDB(d *Data) *gorm.DB { return d.db }

type Data struct {
	db *gorm.DB
}

func NewData(c *conf.Data) (*Data, func(), error) {
	level := logger.Warn
	if c.Database.Debug {
		level = logger.Info
	}
	db, err := gorm.Open(mysql.Open(c.Database.Source), &gorm.Config{
		Logger: logger.Default.LogMode(level),
	})
	if err != nil {
		return nil, nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, nil, err
	}
	sqlDB.SetMaxOpenConns(20)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(time.Hour)
	if c.Database.AutoMigrate {
		if err := db.AutoMigrate(&datapayment.Payment{}, &datapayment.ReconcileDiff{}, &kafkaout.Row{}); err != nil {
			sqlDB.Close()
			return nil, nil, err
		}
	}
	cleanup := func() {
		log.Info("closing the data resources")
		if err := sqlDB.Close(); err != nil {
			log.Error("failed closing the database", "err", err)
		}
	}
	return &Data{db: db}, cleanup, nil
}
