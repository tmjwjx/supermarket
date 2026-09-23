package paging

import (
	"errors"
	"strconv"
)

const (
	defaultSize = 20
	maxSize     = 50
)

// ErrToken 表示 page_token 不是合法偏移
var ErrToken = errors.New("bad page token")

// Size 把页大小收成默认 20 最大 50
func Size(n int32) int {
	if n <= 0 {
		return defaultSize
	}
	if int(n) > maxSize {
		return maxSize
	}
	return int(n)
}

// Offset 把 page_token 解成偏移 空串是第一页
func Offset(token string) (int, error) {
	if token == "" {
		return 0, nil
	}
	n, err := strconv.Atoi(token)
	if err != nil || n < 0 {
		return 0, ErrToken
	}
	return n, nil
}

// Token 把下一页偏移编成 page_token
func Token(offset int) string {
	if offset <= 0 {
		return ""
	}
	return strconv.Itoa(offset)
}
