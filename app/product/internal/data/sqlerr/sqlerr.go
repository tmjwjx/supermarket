package sqlerr

import (
	"errors"

	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

// IsUnique 判断是不是 MySQL 唯一索引冲突
func IsUnique(err error) bool {
	var mysqlErr *mysql.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == 1062
}

// IsNotFound 判断是不是查无记录
func IsNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}
