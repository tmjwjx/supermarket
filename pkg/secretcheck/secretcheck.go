// Package secretcheck 在服务启动时校验令牌签名密钥
package secretcheck

import (
	"errors"
	"fmt"
	"strings"

	"github.com/go-kratos/kratos/v3/log"
)

// placeholderPrefix 是配置文件里本地缺省密钥的前缀 线上必须换掉
const placeholderPrefix = "change-me-"

// Key 是一把签名密钥 Name 用配置字段名方便定位
type Key struct {
	Name  string
	Value string
}

// Check 任一密钥为空或任两把相同返回错误 仍是缺省占位值时只打警告
func Check(keys ...Key) error {
	var errs []error
	for i, k := range keys {
		if strings.TrimSpace(k.Value) == "" {
			errs = append(errs, fmt.Errorf("%s is empty", k.Name))
			continue
		}
		for _, other := range keys[:i] {
			if other.Value == k.Value {
				errs = append(errs, fmt.Errorf("%s must differ from %s", k.Name, other.Name))
			}
		}
		if strings.HasPrefix(k.Value, placeholderPrefix) {
			log.Warn("jwt secret still uses local placeholder, set it before deploying", "key", k.Name)
		}
	}
	return errors.Join(errs...)
}
