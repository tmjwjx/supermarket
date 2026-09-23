package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"unicode/utf8"

	bizadmin "github.com/tmjwjx/supermarket/app/admin/internal/biz/adminuser"
	"github.com/tmjwjx/supermarket/app/admin/internal/conf"
	"github.com/tmjwjx/supermarket/app/admin/internal/data"
	dataadmin "github.com/tmjwjx/supermarket/app/admin/internal/data/adminuser"

	"github.com/go-kratos/kratos/v3/config"
	"github.com/go-kratos/kratos/v3/config/file"
)

var flagconf string

func init() {
	flag.StringVar(&flagconf, "conf", "../../configs", "config path, eg: -conf config.yaml")
}

func main() {
	flag.Parse()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	c := config.New(config.WithSource(file.NewSource(flagconf)))
	defer c.Close()
	if err := c.Load(); err != nil {
		panic(err)
	}
	var bc conf.Bootstrap
	if err := c.Scan(&bc); err != nil {
		panic(err)
	}
	password, err := initPassword(bc.Seed.InitPassword)
	if err != nil {
		logger.Error("admin seed aborted", "err", err)
		os.Exit(1)
	}
	d, cleanup, err := data.NewData(&bc.Data)
	if err != nil {
		panic(err)
	}
	defer cleanup()
	uc := bizadmin.NewAdminUserUsecase(dataadmin.NewAdminUserRepo(data.NewDB(d)), &bc.Auth)
	created, err := uc.Seed(context.Background(), "admin", password)
	if err != nil {
		panic(err)
	}
	if created {
		logger.Info("admin seed created", "username", "admin")
		return
	}
	logger.Info("admin seed skipped", "username", "admin")
}

// minPasswordLen 是超级管理员初始口令的最短长度
const minPasswordLen = 8

// initPassword 要求配置给出初始口令 空或短于 8 位则失败
func initPassword(raw string) (string, error) {
	password := strings.TrimSpace(raw)
	if password == "" {
		return "", errors.New("seed.init_password is required")
	}
	if utf8.RuneCountInString(password) < minPasswordLen {
		return "", fmt.Errorf("seed.init_password must be at least %d characters", minPasswordLen)
	}
	return password, nil
}
