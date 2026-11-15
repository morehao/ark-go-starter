package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	demo "github.com/morehao/ark-go-starter/apps/api"
	"github.com/morehao/ark-go-starter/apps/api/config"
	"github.com/morehao/ark-go-starter/pkg/dbclient"
	"github.com/morehao/golib/dict"
	"github.com/morehao/golib/distlock"
	_ "github.com/morehao/golib/distlock/driver/redis" // 注册 "redis" driver（跨进程互斥）
	"github.com/morehao/golib/glog"
	_ "github.com/morehao/golib/glog/driver/zap"
	"github.com/morehao/golib/gsetting"
	"github.com/morehao/golib/gtrace/otel"
	"github.com/morehao/golib/gtrace/otel/otlptracegrpc"
	"github.com/morehao/golib/task/gasync"
	"github.com/morehao/golib/task/gasync/manager"
	"github.com/morehao/golib/task/gasync/worker"
	"github.com/morehao/golib/task/gcron"
)

var traceProvider *otel.Provider

// distlocker 是 distlock 门面：redis driver 自建连接池，进程退出前必须 Close。
// 构造一次、全进程复用（cron 的分布式锁经 gcron.WithLocker 注入）。
var distlocker distlock.Locker

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

	redisCfg := config.Conf.RedisConfig
	locker, err := distlock.New(distlock.Config{
		Driver: distlock.DriverRedis,
		Redis: &distlock.RedisConfig{
			Service:  "demo-distlock",
			Addr:     redisCfg.Addr,
			Password: redisCfg.Password,
			DB:       redisCfg.DB,
		},
	})
	if err != nil {
		return fmt.Errorf("init distlock failed: %w", err)
	}
	distlocker = locker

	cronInstance, err := gcron.New(db, &config.Conf.Cron, gcron.WithLocker(locker))
	if err != nil {
		return fmt.Errorf("init cron failed: %w", err)
	}
	config.SetCron(cronInstance)
	glog.Infof(context.Background(), "[%s.initComponents] cron initialized", demo.AppName)

	// 任务中心门面：负责提交/查询/统计与建表（默认 AutoMigrate），供 ginasync 挂管理面。
	asyncManager, err := manager.New(&manager.Config{}, db)
	if err != nil {
		return fmt.Errorf("init async manager failed: %w", err)
	}
	// 执行运行时：复用 manager 内部的 DB Source 认领并执行任务。
	asyncWorker := worker.New(asyncManager.Source(), &worker.WorkerConfig{
		Concurrency: config.Conf.Async.Concurrency,
	})
	config.SetAsync(asyncManager, asyncWorker)
	glog.Infof(context.Background(), "[%s.initComponents] async initialized", demo.AppName)

	// 注册示例任务并启动调度：New 只做初始化，不 Start 的话注册的任务永远不执行，
	// 管理页面也永远是空列表。
	if err := registerDemoTasks(); err != nil {
		return err
	}
	startComponents()

	return nil
}

// registerDemoTasks 在 dev 环境注册示例任务，让「定时任务 / 异步任务」页面有真实的
// 可操作对象（列表、详情、运行记录）。
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

	asyncWorker := config.GetAsyncWorker()
	if asyncWorker == nil {
		return nil
	}
	// 任务中心的 handler 接收 *gasync.AsyncTask，payload 经 t.Bind 反序列化。
	if err := asyncWorker.Register("demo-echo", worker.HandlerFunc(func(ctx context.Context, t *gasync.AsyncTask) error {
		var payload []byte
		if err := t.Bind(&payload); err != nil {
			return worker.ErrNoRetry
		}
		glog.Infof(ctx, "[%s.demoEcho] payload:%s", demo.AppName, string(payload))
		return nil
	})); err != nil {
		return fmt.Errorf("register demo async task failed: %w", err)
	}

	// 提交一条示例任务，让「异步任务」页面有真实记录可查看/取消/重放。
	// 用固定 UniqKey 保证重启不重复堆积（命中即幂等返回既有任务）；用较远的 RunAt 让它
	// 停在 pending 而非被本进程 worker 立即消费，便于演示与 e2e 断言。
	asyncManager := config.GetAsyncManager()
	if asyncManager != nil {
		if _, _, err := asyncManager.Submit(context.Background(), &gasync.AsyncTaskSpec{
			Type:    "demo-echo",
			Payload: []byte(`"hello from demo"`),
			BizType: "demo",
			UniqKey: "demo:echo:sample",
			RunAt:   time.Now().Add(24 * time.Hour),
		}); err != nil && !errors.Is(err, gasync.ErrNotify) {
			return fmt.Errorf("submit demo async task failed: %w", err)
		}
	}

	return nil
}

// startComponents 启动定时任务调度与异步 worker。
//
// gcron.Start 与 worker.Start 均为非阻塞（Start 起执行槽后立即返回）。
func startComponents() {
	if scheduler := config.GetCron(); scheduler != nil {
		scheduler.Start()
		glog.Infof(context.Background(), "[%s.startComponents] cron started", demo.AppName)
	}
	if asyncWorker := config.GetAsyncWorker(); asyncWorker != nil {
		if err := asyncWorker.Start(context.Background()); err != nil {
			glog.Errorf(context.Background(), "[%s.startComponents] async worker start fail, err:%v", demo.AppName, err)
		} else {
			glog.Infof(context.Background(), "[%s.startComponents] async worker started", demo.AppName)
		}
	}
}

// shutdownComponents 停止定时任务调度与异步 worker（等待进行中的任务收尾），并关闭锁门面。
func shutdownComponents() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if scheduler := config.GetCron(); scheduler != nil {
		if err := scheduler.Stop(ctx); err != nil {
			glog.Errorf(context.Background(), "[%s.shutdownComponents] stop cron fail, err:%v", demo.AppName, err)
		}
	}
	if asyncWorker := config.GetAsyncWorker(); asyncWorker != nil {
		if err := asyncWorker.Shutdown(ctx); err != nil {
			glog.Errorf(context.Background(), "[%s.shutdownComponents] shutdown async fail, err:%v", demo.AppName, err)
		}
	}
	// 关闭顺序：先停调度/执行，再关闭锁门面自建的连接池。
	if distlocker != nil {
		if err := distlocker.Close(); err != nil {
			glog.Errorf(context.Background(), "[%s.shutdownComponents] close distlock fail, err:%v", demo.AppName, err)
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
