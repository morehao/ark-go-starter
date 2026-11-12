package dbclient

import (
	"github.com/morehao/golib/dbaccess/dbredis"
	"github.com/morehao/golib/glog"
	"github.com/redis/go-redis/v9"
)

// RedisCli 是全局 Redis 客户端。
var RedisCli *redis.Client

// InitRedis 初始化 Redis 客户端。
func InitRedis(config dbredis.RedisConfig, logConfig *glog.LogConfig) error {
	var opts []dbredis.Option
	if logConfig != nil {
		opts = append(opts, dbredis.WithLogConfig(logConfig))
	}
	client, err := dbredis.New(&config, opts...)
	if err != nil {
		return err
	}
	RedisCli = client
	return nil
}
