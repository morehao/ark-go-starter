package main

import (
	"context"
	"fmt"
	"time"

	"github.com/morehao/ark-go-starter/demo"
	"github.com/morehao/ark-go-starter/demo/config"
	"github.com/morehao/ark-go-starter/pkg/dbclient"
	"github.com/morehao/golib/dict"
	"github.com/morehao/golib/distlock"
	"github.com/morehao/golib/glog"
	_ "github.com/morehao/golib/glog/driver/zap"
	"github.com/morehao/golib/gsetting"
	"github.com/morehao/golib/gtrace/otel"
	"github.com/morehao/golib/gtrace/otel/otlptracegrpc"
	"github.com/morehao/golib/task/gasync"
	"github.com/morehao/golib/task/gcron"
)

var traceProvider *otel.Provider

func serverInit() error {
	if err := preInit(); err != nil {
		return err
	}
	if err := initTrace(); err != nil {
		return err
	}
	if err := resourceInit(); err != nil {
		return err
	}
	if err := initComponents(); err != nil {
		return err
	}
	return nil
}

func preInit() error {
	config.InitConf()
	defaultLogCfg := config.Conf.Log["default"]
	if err := glog.InitLogger(&defaultLogCfg); err != nil {
		return fmt.Errorf("init logger failed: %w", err)
	}
	return nil
}

func resourceInit() error {
	var gormLogConfig *glog.LogConfig
	if cfg, ok := config.Conf.Log["gorm"]; ok {
		gormLogConfig = &cfg
	}
	if err := dbclient.InitMultiDB(config.Conf.DBConfigs, gormLogConfig); err != nil {
		return fmt.Errorf("init db failed: %w", err)
	}
	var redisLogConfig *glog.LogConfig
	if cfg, ok := config.Conf.Log["redis"]; ok {
		redisLogConfig = &cfg
	}
	if err := dbclient.InitRedis(config.Conf.RedisConfig, redisLogConfig); err != nil {
		return fmt.Errorf("init redis failed: %w", err)
	}
	return nil
}

// initComponents 初始化 golib 的字典、配置(设置)、定时任务、异步任务组件实例，
// 供路由注册使用；任一组件初始化失败即返回错误（fail-fast）。
func initComponents() error {
	db := dbclient.DemoDB(context.Background())

	dictInstance, err := dict.New(db)
	if err != nil {
		return fmt.Errorf("init dict failed: %w", err)
	}
	config.SetDict(dictInstance)
	glog.Infof(context.Background(), "[%s.initComponents] dict initialized", demo.AppName)

	settingInstance, err := gsetting.New(db)
	if err != nil {
		return fmt.Errorf("init setting failed: %w", err)
	}
	config.SetSetting(settingInstance)
	glog.Infof(context.Background(), "[%s.initComponents] setting initialized", demo.AppName)

	cronInstance, err := gcron.New(db, &config.Conf.Cron, distlock.NewRedisStorage(dbclient.RedisCli))
	if err != nil {
		return fmt.Errorf("init cron failed: %w", err)
	}
	config.SetCron(cronInstance)
	glog.Infof(context.Background(), "[%s.initComponents] cron initialized", demo.AppName)

	asyncInstance, err := gasync.NewServer(&config.Conf.Async, db)
	if err != nil {
		return fmt.Errorf("init async failed: %w", err)
	}
	config.SetAsync(asyncInstance)
	glog.Infof(context.Background(), "[%s.initComponents] async initialized", demo.AppName)

	return nil
}

func initTrace() error {
	provider, err := otlptracegrpc.NewGRPCProvider(context.Background(), demo.AppName, config.Conf.Server.Env, config.Conf.Trace)
	if err != nil {
		glog.Errorf(context.Background(), "[%s.initTrace] init trace failed, fallback to disabled mode, err:%v", demo.AppName, err)
		return nil
	}
	traceProvider = provider
	return nil
}

func shutdownTraceProvider() {
	if traceProvider == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := traceProvider.Shutdown(ctx); err != nil {
		glog.Errorf(context.Background(), "[%s.shutdownTraceProvider] shutdown fail, err:%v", demo.AppName, err)
	}
}
