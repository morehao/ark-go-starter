// Package config 定义应用配置结构并负责从 YAML 加载配置。
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/morehao/golib/dbaccess/dbgorm"
	"github.com/morehao/golib/dbaccess/dbredis"
	"github.com/morehao/golib/dict"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/gsetting"
	"github.com/morehao/golib/gtrace"
	"github.com/morehao/golib/gutil"
	"github.com/morehao/golib/protocol/ghttp"
	"github.com/morehao/golib/storage"
	"github.com/morehao/golib/task/gasync/manager"
	"github.com/morehao/golib/task/gasync/worker"
	"github.com/morehao/golib/task/gcron"
)

// Conf 是进程级配置单例，由 LoadConfig/InitConf 填充。
var Conf *Config

// 进程级组件实例，在启动期初始化一次，供路由注册与后台运行使用。
var (
	dictInstance    *dict.Dict        // 字典组件实例
	settingInstance *gsetting.Setting // 设置组件实例
	cronInstance    *gcron.Scheduler  // 定时任务调度器实例
	asyncManager    *manager.Manager  // 异步任务中心门面（提交/查询/统计，供 ginasync 挂管理面）
	asyncWorker     *worker.Worker    // 异步任务执行运行时（消费任务中心里的任务）
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
	Async       AsyncConfig               `yaml:"async"` // 异步任务配置
	IAM         IAMConfig                 `yaml:"iam"`   // IAM 接入配置（可选，disabled/rp 两态）
}

// AsyncConfig 是异步任务（golib/task/gasync 任务中心）的本地配置。
//
// v1.36.3 起 gasync 从「asynq + Redis 队列」模型整体替换为「DB 认领 + 租约 + 管理面/执行
// 运行时」模型：任务落库（core_async_task*），执行侧从库认领，不再依赖 Redis 队列，因此这里
// 不再有 redis_addr 等连接项。Concurrency 用于配置本地执行 worker 的并发度。
type AsyncConfig struct {
	Concurrency int `yaml:"concurrency"` // 执行 worker 并发度（<=0 时用 worker 默认值 10）
}

// IAM 接入两态取值。
const (
	ModeDisabled = "disabled" // 不接入：零副作用，不初始化 IAM、不建表、不挂 /iam/* 路由
	ModeRP       = "rp"       // 对接外部 ark-iam：仅本地验签，不建库、不挂路由
)

// IAMConfig 是可选接入 ark-iam 的配置。
//
// 两态 mode（disabled 默认 / rp 对接外部 ark-iam 仅本地验签）决定接入形态。
// 均未接入（disabled）时零副作用：不初始化 IAM、不建表、不挂 /iam/* 路由，业务鉴权回落到
// 固定 dev 身份（仅 dev 环境，生产禁用）。
type IAMConfig struct {
	Mode     string `yaml:"mode"`     // 接入形态：disabled（默认）| rp
	Issuer   string `yaml:"issuer"`   // rp 形态：外部 ark-iam 的 issuer
	JWKSURL  string `yaml:"jwks_url"` // rp 形态的 JWKS 端点，空则按 issuer 推导
	Audience string `yaml:"audience"` // rp 形态需校验的 token aud（防其它应用 token 误接受）
}

// EffectiveMode 返回归一化（小写、去空白）后的接入形态。
//
// 规则：mode 非空 → 原样（小写/去空白）返回（含非法值，由调用方经 Validate 报错）；
// mode 为空 → disabled。
func (c IAMConfig) EffectiveMode() string {
	if mode := strings.ToLower(strings.TrimSpace(c.Mode)); mode != "" {
		return mode
	}
	return ModeDisabled
}

// Validate 校验接入形态是否合法（仅接受 disabled/rp）。
func (c IAMConfig) Validate() error {
	switch c.EffectiveMode() {
	case ModeDisabled, ModeRP:
		return nil
	default:
		return fmt.Errorf("invalid iam.mode %q: only disabled/rp supported", c.Mode)
	}
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

// SetAsync 设置进程级异步任务中心门面 + 执行运行时实例。
func SetAsync(mgr *manager.Manager, w *worker.Worker) {
	asyncManager = mgr
	asyncWorker = w
}

// GetAsyncManager 返回进程级异步任务中心门面（供 ginasync 挂管理面）。
func GetAsyncManager() *manager.Manager { return asyncManager }

// GetAsyncWorker 返回进程级异步任务执行运行时（供 start/shutdown）。
func GetAsyncWorker() *worker.Worker { return asyncWorker }

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
	applyRuntimePathOverrides(&cfg)
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
