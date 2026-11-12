package main

import (
	"context"
	"fmt"
	"time"

	demo "github.com/morehao/ark-go-starter/apps/api"
	"github.com/morehao/ark-go-starter/apps/api/config"
	"github.com/morehao/ark-go-starter/libs/go/dbclient"
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

	asyncCfg := config.Conf.Async.ToGasync()
	asyncInstance, err := gasync.NewServer(&asyncCfg, db)
	if err != nil {
		return fmt.Errorf("init async failed: %w", err)
	}
	config.SetAsync(asyncInstance)
	glog.Infof(context.Background(), "[%s.initComponents] async initialized", demo.AppName)

	// 注册示例任务并启动调度：New/NewServer 只做初始化，不 Start/Run 的话
	// 注册的任务永远不会执行，管理页面也永远是空列表。
	if err := registerDemoTasks(); err != nil {
		return err
	}
	startComponents()

	return nil
}

// registerDemoTasks 在 dev 环境注册示例任务，让「定时任务 / 异步任务」页面有真实的
// 可操作对象（列表、详情、运行记录、启停开关）。
//
// 仅 dev 注册：线上不应产生示例调度与示例队列消费。处理器只打日志、不做业务操作。
// 任务 ID / 类型名带 demo- 前缀，避免与业务任务混淆。
func registerDemoTasks() error {
	if config.Conf.Server.Env != "dev" {
		return nil
	}

	scheduler := config.GetCron()
	if scheduler == nil {
		return nil
	}
	if err := scheduler.Register(gcron.Task{
		ID:          "demo-heartbeat",
		BizType:     "demo",
		Name:        "示例：心跳任务",
		TaskType:    "demo",
		Spec:        "*/5 * * * *", // 5 段 cron（未开启 WithSeconds），每 5 分钟一次
		Description: "示例任务：每 5 分钟打印一条日志，用于演示启停、运行记录等运维流程",
		Handler: func(ctx context.Context) error {
			glog.Infof(ctx, "[%s.demoHeartbeat] tick", demo.AppName)
			return nil
		},
	}); err != nil {
		return fmt.Errorf("register demo cron task failed: %w", err)
	}

	asyncServer := config.GetAsync()
	if asyncServer == nil {
		return nil
	}
	if err := asyncServer.Register("demo-echo", func(ctx context.Context, payload []byte) error {
		glog.Infof(ctx, "[%s.demoEcho] payload:%s", demo.AppName, string(payload))
		return nil
	}); err != nil {
		return fmt.Errorf("register demo async task failed: %w", err)
	}

	return nil
}

// startComponents 启动定时任务调度与异步 worker。
//
// gcron.Start 非阻塞；gasync.Run 阻塞，因此放到 goroutine 里跑（进程退出时随之结束）。
func startComponents() {
	if scheduler := config.GetCron(); scheduler != nil {
		scheduler.Start()
		glog.Infof(context.Background(), "[%s.startComponents] cron started", demo.AppName)
	}
	if asyncServer := config.GetAsync(); asyncServer != nil {
		go func() {
			if err := asyncServer.Run(); err != nil {
				glog.Errorf(context.Background(), "[%s.startComponents] async run fail, err:%v", demo.AppName, err)
			}
		}()
		glog.Infof(context.Background(), "[%s.startComponents] async worker started", demo.AppName)
	}
}

// shutdownComponents 停止定时任务调度与异步 worker（等待进行中的任务收尾）。
func shutdownComponents() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if scheduler := config.GetCron(); scheduler != nil {
		if err := scheduler.Stop(ctx); err != nil {
			glog.Errorf(context.Background(), "[%s.shutdownComponents] stop cron fail, err:%v", demo.AppName, err)
		}
	}
	if asyncServer := config.GetAsync(); asyncServer != nil {
		if err := asyncServer.ShutdownContext(ctx); err != nil {
			glog.Errorf(context.Background(), "[%s.shutdownComponents] shutdown async fail, err:%v", demo.AppName, err)
		}
	}
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
