// Package testsetup 为测试提供应用环境初始化（配置、日志、DB、Redis）。
package testsetup

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/morehao/golib/gconstant"
	"github.com/morehao/golib/gutil"
)

// Initializer 是测试环境的初始化器：Initialize 建立依赖（配置/日志/DB/Redis），Close 释放资源。
//
// 历史上这两个类型是 golib `biz/testkit` 的类型别名。golib v1.34.5 起把测试基建迁到
// `gtest`（`biz/testkit` 已删除），新 API 以 `testing.TB` + AppBoot 为中心，与本案
// 「按 appName 注册并在包内按需初始化」的调用方式不同；这里改为本包自包含实现，
// 保持既有调用方（Initialize/Close/NewContext）零改动。
type Initializer interface {
	Initialize() error
	Close() error
}

// InitializerFunc 按需构造 Initializer（首次 Initialize 时调用一次）。
type InitializerFunc func() (Initializer, error)

// Option 在构造测试 Gin 上下文后追加定制（如注入 request context）。
type Option func(ctx *gin.Context)

var (
	mu sync.RWMutex
	// initializerCreators 是「应用名 → 初始化器构造器」的注册表（内置 demo，其余按需 RegisterApp）。
	initializerCreators = map[string]InitializerFunc{
		AppNameDemo: newDemoappInitializer,
	}
	// appInitializers 保存已初始化的实例，供 Close 释放。
	appInitializers = make(map[string]Initializer)
	// initOnce 保证每个应用名的初始化只执行一次。
	initOnce = make(map[string]*sync.Once)
)

func init() {
	gin.SetMode(gin.TestMode)
}

// RegisterApp 注册应用名到初始化函数的映射（已注册过则幂等跳过）。
func RegisterApp(appName string, initFunc InitializerFunc) {
	mu.Lock()
	defer mu.Unlock()
	if _, exists := initializerCreators[appName]; exists {
		return
	}
	initializerCreators[appName] = initFunc
}

// Initialize 初始化指定应用名对应的测试环境（并发安全，仅首次生效）。
// 创建或初始化失败直接 panic——测试环境起不来时继续跑没有意义。
func Initialize(appName string) {
	once := onceFor(appName)

	once.Do(func() {
		mu.RLock()
		creator, ok := initializerCreators[appName]
		mu.RUnlock()
		if !ok {
			panic(fmt.Sprintf("app %s not registered", appName))
		}

		initializer, err := creator()
		if err != nil {
			panic(fmt.Sprintf("create initializer for app %s failed: %v", appName, err))
		}
		if err := initializer.Initialize(); err != nil {
			panic(fmt.Sprintf("initialize app %s failed: %v", appName, err))
		}

		mu.Lock()
		appInitializers[appName] = initializer
		mu.Unlock()
	})
}

// onceFor 取（必要时创建）指定应用名的 sync.Once。
func onceFor(appName string) *sync.Once {
	mu.Lock()
	defer mu.Unlock()

	once, ok := initOnce[appName]
	if !ok {
		once = &sync.Once{}
		initOnce[appName] = once
	}
	return once
}

// Close 关闭指定应用名的测试环境资源；appName 为空时关闭全部。
func Close(appName string) {
	if appName == "" {
		closeAll()
		return
	}

	mu.RLock()
	initializer, ok := appInitializers[appName]
	mu.RUnlock()
	if !ok {
		return
	}

	_ = initializer.Close()

	mu.Lock()
	defer mu.Unlock()
	delete(appInitializers, appName)
	delete(initOnce, appName)
}

// closeAll 关闭全部已初始化实例（用于 TestMain 收尾）。
func closeAll() {
	mu.Lock()
	defer mu.Unlock()

	for _, initializer := range appInitializers {
		_ = initializer.Close()
	}
	appInitializers = make(map[string]Initializer)
	initOnce = make(map[string]*sync.Once)
}

// Init 是 Initialize 的简写。
func Init(appName string) {
	Initialize(appName)
}

// Done 是 Close 的简写。
func Done(appName string) {
	Close(appName)
}

// WithContext 把外层 context 注入测试 Gin 上下文的 Request。
func WithContext(ctx context.Context) Option {
	return func(gc *gin.Context) {
		if ctx != nil {
			gc.Request = gc.Request.WithContext(ctx)
		}
	}
}

// NewContext 构造一个带测试请求的 Gin 上下文（自动补 requestID）。
func NewContext(opts ...Option) *gin.Context {
	ctx := &gin.Context{
		Request: &http.Request{
			URL:    &url.URL{},
			Header: http.Header{},
		},
	}
	ctx.Request = ctx.Request.WithContext(context.Background())

	for _, opt := range opts {
		opt(ctx)
	}

	if _, exists := ctx.Get(gconstant.KeyAppRequestID); !exists {
		ctx.Set(gconstant.KeyAppRequestID, gutil.GenUUID())
	}

	return ctx
}
