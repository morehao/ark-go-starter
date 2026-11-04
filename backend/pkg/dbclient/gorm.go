// Package dbclient 封装 GORM/Redis 客户端的初始化与取用。
package dbclient

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/morehao/golib/dbaccess/dbgorm"
	// 注册 postgres 驱动：gormdao 按驱动名 "postgres" 解析连接，必须 blank import 触发 init。
	_ "github.com/morehao/golib/dbaccess/dbgorm/driver/postgres"
	"github.com/morehao/golib/glog"
	"gorm.io/gorm"
)

var (
	dbMap   = make(map[string]*gorm.DB)
	dbMutex sync.RWMutex
)

const (
	dbNameDemo = "demo"
)

// InitMultiDB 按配置初始化多个数据库连接并注册到全局 map。
func InitMultiDB(configs []dbgorm.Config, logConfig *glog.LogConfig) error {
	if len(configs) == 0 {
		return errors.New("db config is empty")
	}

	var opts []dbgorm.Option
	if logConfig != nil {
		opts = append(opts, dbgorm.WithLogConfig(logConfig))
	}
	// 业务层经 gormdao 封装调用，比直接持有 *gorm.DB 多一层栈帧，额外跳过 1 帧使 SQL 日志 caller 定位到 svc 服务层
	opts = append(opts, dbgorm.WithCallerSkip(3))
	for _, cfg := range configs {
		client, err := dbgorm.New(&cfg, opts...)
		if err != nil {
			return fmt.Errorf("init db failed: %w", err)
		}
		dbMutex.Lock()
		dbMap[cfg.Service] = client
		dbMutex.Unlock()
	}
	return nil
}

// GetDB 返回指定 name 的数据库连接，并绑定到 ctx（携带链路/超时）。
func GetDB(ctx context.Context, dbName string) *gorm.DB {
	dbMutex.RLock()
	defer dbMutex.RUnlock()
	return dbMap[dbName].WithContext(ctx)
}

// DemoDB 返回 demo 应用默认库的连接，并绑定到 ctx。
func DemoDB(ctx context.Context) *gorm.DB {
	return GetDB(ctx, dbNameDemo)
}
