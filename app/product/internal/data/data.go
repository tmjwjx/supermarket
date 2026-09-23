package data

import (
	"context"
	"time"

	"github.com/tmjwjx/supermarket/app/product/internal/conf"
	dataattr "github.com/tmjwjx/supermarket/app/product/internal/data/attribute"
	databrand "github.com/tmjwjx/supermarket/app/product/internal/data/brand"
	databrowse "github.com/tmjwjx/supermarket/app/product/internal/data/browse"
	datacategory "github.com/tmjwjx/supermarket/app/product/internal/data/category"
	datafavorite "github.com/tmjwjx/supermarket/app/product/internal/data/favorite"
	dataproduct "github.com/tmjwjx/supermarket/app/product/internal/data/product"
	datarecommendation "github.com/tmjwjx/supermarket/app/product/internal/data/recommendation"
	datareview "github.com/tmjwjx/supermarket/app/product/internal/data/review"
	"github.com/tmjwjx/supermarket/pkg/kafkaout"

	"github.com/go-kratos/kratos/v3/log"
	_ "github.com/go-sql-driver/mysql"
	"github.com/google/wire"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(
	NewData,
	NewDB,
	NewRedis,
	databrand.NewBrandRepo,
	datacategory.NewCategoryRepo,
	dataattr.NewTemplateRepo,
	dataproduct.NewProductRepo,
	kafkaout.NewStore,
	datarecommendation.NewRecommendationRepo,
	datafavorite.NewFavoriteRepo,
	databrowse.NewBrowseRepo,
	datareview.NewReviewRepo,
)

// Data 持有 MySQL 和可选的 Redis
type Data struct {
	db  *gorm.DB
	rdb *redis.Client
}

// NewDB 把共享连接交给资源仓库
func NewDB(d *Data) *gorm.DB { return d.db }

// NewRedis 可能为空 表示缓存不可用
func NewRedis(d *Data) *redis.Client { return d.rdb }

// NewData 打开 MySQL 并建表 Redis 连不上时只关掉缓存
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
		if err := db.AutoMigrate(
			&databrand.Brand{},
			&datacategory.Category{},
			&dataattr.Template{},
			&dataattr.Attribute{},
			&dataproduct.Product{},
			&dataproduct.Sku{},
			&dataproduct.ProductImage{},
			&dataproduct.ProductDetail{},
			&dataproduct.ProductParam{},
			&dataproduct.ProductAudit{},
			&dataproduct.ProductLog{},
			&dataproduct.ConsumedEvent{},
			&dataproduct.SalesApply{},
			&kafkaout.Row{},
			&datarecommendation.Recommendation{},
			&datafavorite.Favorite{},
			&databrowse.BrowseHistory{},
			&datareview.Review{},
		); err != nil {
			_ = sqlDB.Close()
			return nil, nil, err
		}
	}
	rdb := openRedis(c.Redis)
	cleanup := func() {
		log.Info("closing the data resources")
		if rdb != nil {
			if err := rdb.Close(); err != nil {
				log.Error("failed closing redis", "err", err)
			}
		}
		if err := sqlDB.Close(); err != nil {
			log.Error("failed closing the database", "err", err)
		}
	}
	return &Data{db: db, rdb: rdb}, cleanup, nil
}

func openRedis(c conf.Redis) *redis.Client {
	addr := c.Addr
	if addr == "" {
		addr = "127.0.0.1:6379"
	}
	rdb := redis.NewClient(&redis.Options{
		Network:      c.Network,
		Addr:         addr,
		ReadTimeout:  c.ReadTimeout(),
		WriteTimeout: c.WriteTimeout(),
	})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Warn("redis unavailable cache disabled", "err", err)
		_ = rdb.Close()
		return nil
	}
	return rdb
}
