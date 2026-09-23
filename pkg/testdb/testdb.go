package testdb

import (
	"os"
	"strings"
	"testing"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DSN 从 TEST_MYSQL_DSN 读取不带库名的连接串 缺省本机 root 账号
func DSN() string {
	if v := strings.TrimSpace(os.Getenv("TEST_MYSQL_DSN")); v != "" {
		return v
	}
	return "root:root@tcp(127.0.0.1:3306)/"
}

// Open 重建一个独立测试库并建好表 不碰服务正在用的库
func Open(t *testing.T, name string, models ...any) *gorm.DB {
	t.Helper()
	root, err := gorm.Open(mysql.Open(DSN()+"?parseTime=True&loc=Local&charset=utf8mb4"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("connect mysql: %v", err)
	}
	if err := root.Exec("DROP DATABASE IF EXISTS `" + name + "`").Error; err != nil {
		t.Fatal(err)
	}
	if err := root.Exec("CREATE DATABASE `" + name + "` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci").Error; err != nil {
		t.Fatal(err)
	}
	if sqlDB, err := root.DB(); err == nil {
		_ = sqlDB.Close()
	}
	db, err := gorm.Open(mysql.Open(DSN()+name+"?parseTime=True&loc=Local&charset=utf8mb4"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(64)
	if err := db.AutoMigrate(models...); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	return db
}
