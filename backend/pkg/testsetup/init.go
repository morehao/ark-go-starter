// Package testsetup 为测试提供应用环境初始化（配置、日志、DB、Redis）。
package testsetup

import (
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/morehao/golib/biz/testkit"
)

// Initializer 是 golib testkit.Initializer 的类型别名，便于调用方少引一层包。
type Initializer = testkit.Initializer

// InitializerFunc 是 golib testkit.InitializerFunc 的类型别名。
type InitializerFunc = testkit.InitializerFunc

var (
	mu                  sync.RWMutex
	initializerCreators = map[string]InitializerFunc{
		AppNameDemo: newDemoappInitializer,
	}
	registeredApps = make(map[string]bool)
)

func init() {
	gin.SetMode(gin.TestMode)
}

// RegisterApp 注册应用名到初始化函数的映射。
func RegisterApp(appName string, initFunc InitializerFunc) {
	mu.Lock()
	defer mu.Unlock()
	if registeredApps[appName] {
		return
	}
	testkit.RegisterInitializer(appName, initFunc)
	registeredApps[appName] = true
}

// Initialize 初始化指定应用名对应的测试环境（并发安全，仅首次生效）。
func Initialize(appName string) {
	mu.Lock()
	if !registeredApps[appName] {
		if creator, ok := initializerCreators[appName]; ok {
			testkit.RegisterInitializer(appName, creator)
			registeredApps[appName] = true
		}
	}
	mu.Unlock()
	testkit.Initialize(appName)
}

// Close 关闭指定应用名的测试环境资源。
func Close(appName string) {
	testkit.Close(appName)
}

// Init 是 Initialize 的简写。
func Init(appName string) {
	Initialize(appName)
}

// Done 是 Close 的简写。
func Done(appName string) {
	Close(appName)
}

// NewContext 构造一个带测试请求的 Gin 上下文。
func NewContext(opts ...testkit.Option) *gin.Context {
	return testkit.NewContext(opts...)
}
