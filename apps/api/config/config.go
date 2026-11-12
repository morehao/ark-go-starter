// Package config 定义应用配置结构并负责从 YAML 加载配置。
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

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
	Cron        gcron.Config              `yaml:"cron"`    // 定时任务配置
	Async       AsyncConfig               `yaml:"async"`   // 异步任务配置
	IAM         IAMConfig                 `yaml:"iam"`     // 内嵌 IAM 配置
	Install     InstallConfig             `yaml:"install"` // 首次初始化引导配置（op）
}

// InstallConfig 是首次初始化引导（POST /iam/install/initialize）的配置。
//
// BootstrapToken 是该写接口的门禁令牌（等价于「创建平台管理员」的权限）。
// 生效优先级：环境变量 BOOTSTRAP_TOKEN > 本配置项。仅用于 dev，生产必须用环境变量/密钥管理
// 覆盖并移除；初始化完成后即可作废。见 docs/agents/quickstart.md。
type InstallConfig struct {
	BootstrapToken string `yaml:"bootstrapToken"` // 引导令牌（env BOOTSTRAP_TOKEN 优先）
}

// AsyncConfig 是异步任务（golib/gasync）的配置。
//
// golib 的 gasync.Config 未声明 yaml tag，yaml.v3 默认按字段名小写（redisaddr）
// 匹配，导致 config.yaml 里 snake_case 的 redis_addr 无法生效、启动时报
// "gasync: redis addr is empty"。这里用带 yaml tag 的本地结构承接配置，
// 再转换为 gasync.Config，既保持配置文件 snake_case 风格，又避免运行时踩坑。
type AsyncConfig struct {
	RedisAddr     string        `yaml:"redis_addr"`     // Redis 地址
	RedisPassword string        `yaml:"redis_password"` // Redis 密码
	RedisDB       int           `yaml:"redis_db"`       // Redis 库号
	Concurrency   int           `yaml:"concurrency"`    // 并发度（<=0 时用 gasync 默认）
	Timeout       time.Duration `yaml:"timeout"`        // 任务超时（<=0 时用 gasync 默认）
}

// ToGasync 转换为 golib gasync.Config。
func (c AsyncConfig) ToGasync() gasync.Config {
	cfg := gasync.Config{
		RedisAddr:     c.RedisAddr,
		RedisPassword: c.RedisPassword,
		RedisDB:       c.RedisDB,
		Concurrency:   c.Concurrency,
		Timeout:       c.Timeout,
	}
	return cfg
}

// IAM 接入三态取值。Mode 为空时按旧 Enable 字段归一（Enable=true → ModeOP）。
const (
	ModeDisabled = "disabled" // 不接入：零副作用，不初始化 IAM、不建表、不挂 /iam/* 路由
	ModeOP       = "op"       // 本进程内嵌 OIDC Provider：建库、挂 /iam/* 路由、签发/验签
	ModeRP       = "rp"       // 对接外部 ark-iam：仅本地验签，不建库、不挂路由
)

// IAMConfig 是可选接入 ark-iam 的配置。
//
// 三态 mode（disabled 默认 / op 本进程内嵌 OP / rp 对接外部 ark-iam 仅本地验签）决定接入形态。
// 兼容旧字段 Enable：mode 为空且 Enable=true 时按 op 处理，否则 disabled。
// 均未接入（disabled）时零副作用：不初始化 IAM、不建表、不挂 /iam/* 路由，业务鉴权回落到
// 固定 dev 身份（仅 dev 环境，生产禁用）。
type IAMConfig struct {
	Mode          string `yaml:"mode"`           // 接入形态：disabled（默认）| op | rp
	Enable        bool   `yaml:"enable"`         // 旧二态开关（兼容）：mode 为空且为 true 时按 op
	Prefix        string `yaml:"prefix"`         // IAM 挂载路由前缀，缺省 /iam
	DSN           string `yaml:"dsn"`            // IAM 数据库 DSN，空则复用主库第一个 db_configs
	RedisAddr     string `yaml:"redis_addr"`     // IAM Redis 地址，空则复用 redis_config
	RedisPassword string `yaml:"redis_password"` // IAM Redis 密码，空则复用 redis_config
	Issuer        string `yaml:"issuer"`         // OIDC issuer，op 空则按端口推导；rp 为外部 ark-iam 的 issuer
	JWKSURL       string `yaml:"jwks_url"`       // rp 形态的 JWKS 端点，空则按 issuer 推导
	Audience      string `yaml:"audience"`       // rp 形态需校验的 token aud（防其它应用 token 误接受）
	SigningKey    string `yaml:"signing_key"`    // 签名私钥（PEM），非 dev 必填（op）
	EncryptionKey string `yaml:"encryption_key"` // 加密密钥，非 dev 必填（op）
	CookieSecure  bool   `yaml:"cookie_secure"`  // SSO 会话 cookie 的 Secure 标志，生产（HTTPS）必须 true（op）
	// RedirectURIs 是内置控制台 platform_admin_web 的 OAuth 回调白名单（完整 URL，可多个），
	// op 形态下写入该 client 的 redirect_uris（seed 时生效；已初始化的库需手动更新库或清库重建）。
	// 留空则沿用 ark-iam 内置缺省（本地开发默认 http://localhost:4001/auth/callback）。
	RedirectURIs []string `yaml:"redirect_uris"`
	// FrontendURL 是前端应用基址（如 http://localhost:3000），op 形态下用于拼内嵌登录页地址
	// `{frontend_url}{prefix}/login`。前后端分离部署时**必须**配置，否则登录页会被导向 OP
	// 自身的 host（后端端口），而登录页 SPA 实际由前端提供，导致 /iam/login 404。
	// 留空则回退用 issuer 的 host（同源部署场景）。
	FrontendURL string `yaml:"frontend_url"`
}

// EffectiveMode 返回归一化（小写、去空白）后的接入形态。
//
// 规则：mode 非空 → 校验只接受 disabled/op/rp（非法值原样返回，由调用方经 Validate 报错）；
// mode 为空且 Enable=true → op；否则 disabled。
func (c IAMConfig) EffectiveMode() string {
	if mode := strings.ToLower(strings.TrimSpace(c.Mode)); mode != "" {
		return mode
	}
	if c.Enable {
		return ModeOP
	}
	return ModeDisabled
}

// Validate 校验接入形态是否合法（仅接受 disabled/op/rp）。disabled 恒合法。
func (c IAMConfig) Validate() error {
	switch c.EffectiveMode() {
	case ModeDisabled, ModeOP, ModeRP:
		return nil
	default:
		return fmt.Errorf("invalid iam.mode %q: must be one of %q|%q|%q", c.Mode, ModeDisabled, ModeOP, ModeRP)
	}
}

// GetPrefix 返回归一化后的 IAM 路由前缀（保证以 "/" 开头且不以 "/" 结尾）。
func (c IAMConfig) GetPrefix() string {
	prefix := c.Prefix
	if prefix == "" {
		return "/iam"
	}
	if prefix[0] != '/' {
		prefix = "/" + prefix
	}
	for len(prefix) > 1 && prefix[len(prefix)-1] == '/' {
		prefix = prefix[:len(prefix)-1]
	}
	return prefix
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
