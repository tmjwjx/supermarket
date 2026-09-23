package data

import (
	"time"

	"github.com/tmjwjx/supermarket/app/user/internal/conf"
	datauser "github.com/tmjwjx/supermarket/app/user/internal/data/user"

	"github.com/go-kratos/kratos/v3/log"
	_ "github.com/go-sql-driver/mysql"
	"github.com/google/wire"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(NewData, NewDB, datauser.NewUserRepo, datauser.NewAddressRepo)

// NewDB 把共享连接交给资源仓库；仓库包不能引回 data，否则和 ProviderSet 循环引用。
func NewDB(d *Data) *gorm.DB { return d.db }

// Data holds the long-lived storage clients shared by repos.
type Data struct {
	db *gorm.DB
}

// NewData opens the database client and returns it with a cleanup function.
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
		if err := db.AutoMigrate(&datauser.User{}, &datauser.Address{}); err != nil {
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
