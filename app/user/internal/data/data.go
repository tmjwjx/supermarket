package data

import (
	"time"

	"github.com/tmjwjx/supermarket/app/user/internal/conf"

	"github.com/go-kratos/kratos/v3/log"
	_ "github.com/go-sql-driver/mysql"
	"github.com/google/wire"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(NewData, NewTodoRepo)

// Data holds the long-lived storage clients shared by repos.
type Data struct {
	db *gorm.DB
}

// NewData opens the database client and returns it with a cleanup function.
func NewData(c *conf.Data) (*Data, func(), error) {
	dc := c.GetDatabase()
	level := logger.Warn
	if dc.GetDebug() {
		level = logger.Info
	}
	db, err := gorm.Open(mysql.Open(dc.GetSource()), &gorm.Config{
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
	// Auto migration is a convenience for local development. In production,
	// apply schema changes as a separate reviewed step instead.
	// TODO: register one model list per resource as the domain grows.
	if dc.GetAutoMigrate() {
		if err := db.AutoMigrate(&TodoPO{}); err != nil {
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
