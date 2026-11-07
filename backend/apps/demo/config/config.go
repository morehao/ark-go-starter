// Package config 定义应用配置结构并负责从 YAML 加载配置。
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/morehao/golib/dbaccess/dbgorm"
	"github.com/morehao/golib/dbaccess/dbredis"
	"github.com/morehao/golib/dict"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/gsetting"
	"github.com/morehao/golib/gtrace"
	"github.com/morehao/golib/gutil"
	"github.com/morehao/golib/protocol/ghttp"
	"github.com/morehao/golib/storage"
	"github.com/morehao/golib/task/gasync"
	"github.com/morehao/golib/task/gcron"
)

// Conf 是进程级配置单例，由 LoadConfig/InitConf 填充。
var Conf *Config

// 进程级组件实例，在启动期初始化一次，供路由注册使用。
var (
	dictInstance    *dict.Dict        // 字典组件实例
	settingInstance *gsetting.Setting // 设置组件实例
	cronInstance    *gcron.Scheduler  // 定时任务调度器实例
	asyncInstance   *gasync.Server    // 异步任务服务端实例
)

// Config 是应用的全部配置项。
type Config struct {
	Server      Server                    `yaml:"server"`
	Log         map[string]glog.LogConfig `yaml:"log"`
	Trace       gtrace.TraceConfig        `yaml:"trace"`
	DBConfigs   []dbgorm.Config           `yaml:"db_configs"`
	RedisConfig dbredis.RedisConfig       `yaml:"redis_config"`
	Client      Client                    `yaml:"client"`
	FileStorage FileStorage               `yaml:"file_storage"`
	Cron        gcron.Config              `yaml:"cron"`  // 定时任务配置
	Async       gasync.Config             `yaml:"async"` // 异步任务配置
}

// Server 是 HTTP 服务配置。
type Server struct {
	Name string `yaml:"name"` // 服务名称
	Port string `yaml:"port"` // 服务端口
	Env  string `yaml:"env"`  // 环境变量
}

// Client 是外部 HTTP 客户端配置。
type Client struct {
	HTTPBingo *ghttp.Client `yaml:"httpbingo"`
}

// FileStorage 文件上传存储配置（对应 golib ginupload / filestore）。
type FileStorage struct {
	Driver  storage.DriverType `yaml:"driver"`  // 存储驱动类型
	Bucket  string             `yaml:"bucket"`  // 存储桶
	Storage storage.Config     `yaml:"storage"` // 驱动通用配置
}

// SetDict 设置进程级字典组件实例。
func SetDict(instance *dict.Dict) { dictInstance = instance }

// GetDict 返回进程级字典组件实例。
func GetDict() *dict.Dict { return dictInstance }

// SetSetting 设置进程级设置组件实例。
func SetSetting(instance *gsetting.Setting) { settingInstance = instance }

// GetSetting 返回进程级设置组件实例。
func GetSetting() *gsetting.Setting { return settingInstance }

// SetCron 设置进程级定时任务调度器实例。
func SetCron(instance *gcron.Scheduler) { cronInstance = instance }

// GetCron 返回进程级定时任务调度器实例。
func GetCron() *gcron.Scheduler { return cronInstance }

// SetAsync 设置进程级异步任务服务端实例。
func SetAsync(instance *gasync.Server) { asyncInstance = instance }

// GetAsync 返回进程级异步任务服务端实例。
func GetAsync() *gasync.Server { return asyncInstance }

// InitConf 初始化配置（运行时使用，自动查找配置文件）.
func InitConf() {
	configPath := getConfigPath()
	LoadConfig(configPath)
}

// LoadConfig 从指定路径加载配置并设置全局变量（测试环境可直接使用）.
func LoadConfig(configPath string) {
	fmt.Println("Load config file:", configPath)

	var cfg Config
	gutil.LoadYamlConfig(configPath, &cfg)
	Conf = &cfg
}

// getConfigPath 获取配置文件路径，优先级：环境变量 > 相对路径 > 绝对路径
func getConfigPath() string {
	// 优先使用环境变量
	if configPath := os.Getenv("APP_CONFIG_PATH"); configPath != "" {
		return configPath
	}

	// 尝试相对路径（从当前工作目录）
	relativePath := "../config/config.yaml"
	if fileExists(relativePath) {
		return relativePath
	}

	// 获取可执行文件所在目录的上级目录
	execPath, err := os.Executable()
	if err == nil {
		// 可执行文件目录的上级目录/config/config.yaml
		absPath := filepath.Join(filepath.Dir(execPath), "..", "config", "config.yaml")
		if fileExists(absPath) {
			return absPath
		}
	}

	// 默认返回相对路径
	return relativePath
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
