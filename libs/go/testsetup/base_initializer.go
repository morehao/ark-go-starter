package testsetup

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/morehao/ark-go-starter/libs/go/dbclient"
	"github.com/morehao/golib/dbaccess/dbgorm"
	"github.com/morehao/golib/dbaccess/dbredis"
	"github.com/morehao/golib/glog"

	// 注册 zap 日志驱动：glog.InitLogger 按驱动名 "zap" 创建 logger，必须 blank import 触发 init。
	_ "github.com/morehao/golib/glog/driver/zap"
	"github.com/morehao/golib/gutil"
)

type baseAppInitializer struct {
	AppName    string
	ConfigPath string

	Log         map[string]glog.LogConfig
	DBConfigs   []dbgorm.Config
	RedisConfig dbredis.RedisConfig
}

// AppConfig 是测试初始化所需的配置集合。
type AppConfig struct {
	Log         map[string]glog.LogConfig `yaml:"log"`
	DBConfigs   []dbgorm.Config           `yaml:"db_configs"`
	RedisConfig dbredis.RedisConfig       `yaml:"redis_config"`
}

func newBaseAppInitializer(appName string) (*baseAppInitializer, error) {
	configPath := findConfigPath(appName)
	if configPath == "" {
		return nil, fmt.Errorf("cannot find config path for app: %s", appName)
	}

	return &baseAppInitializer{
		AppName:    appName,
		ConfigPath: configPath,
	}, nil
}

// repoRoot 从本包源文件位置向上回溯，找到包含 go.work 的目录（仓库根）。
// 用「向上找 go.work」代替「固定上溯 N 级」，避免 libs/go/testsetup 这类深层目录
// 在模块搬迁后层数变化导致路径拼错。
func repoRoot() string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return ""
	}
	dir := filepath.Dir(filename) // .../libs/go/testsetup
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.work")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir { // 到达文件系统根仍未找到
			return ""
		}
		dir = parent
	}
}

// resolveAppDir 返回应用标识(appName)对应的目录名：优先环境变量 APP_DIR，
// 其次查 appDirByName 映射，最后回退为 appName 本身（兼容目录名==逻辑名的旧应用）。
func resolveAppDir(appName string) string {
	if dir := os.Getenv("APP_DIR"); dir != "" {
		return dir
	}
	if dir, ok := appDirByName[appName]; ok {
		return dir
	}
	return appName
}

// appConfigDir 返回指定应用 config 目录的绝对路径：<repoRoot>/apps/<AppDir>/config。
func appConfigDir(appName string) string {
	root := repoRoot()
	if root == "" {
		return ""
	}
	return filepath.Join(root, "apps", resolveAppDir(appName), "config")
}

// findConfigPath 返回应用生产配置文件路径：<repoRoot>/apps/<AppDir>/config/config.yaml。
func findConfigPath(appName string) string {
	dir := appConfigDir(appName)
	if dir == "" {
		return ""
	}
	return filepath.Join(dir, "config.yaml")
}

// testLogDir 返回测试日志目录：APP_LOG_DIR 优先，否则 <repoRoot>/.local/log。
func testLogDir() string {
	if d := os.Getenv("APP_LOG_DIR"); d != "" {
		return d
	}
	if root := repoRoot(); root != "" {
		return filepath.Join(root, ".local", "log")
	}
	return filepath.Join(".local", "log")
}

// applyTestLogDir 把所有日志配置里的文件 writer 目录统一改为 testLogDir()。
func applyTestLogDir(logs map[string]glog.LogConfig) {
	dir := testLogDir()
	if dir == "" {
		return
	}
	for name, lc := range logs {
		for i := range lc.Writers {
			if lc.Writers[i].Type == glog.WriterFile {
				lc.Writers[i].Dir = dir
			}
		}
		logs[name] = lc
	}
}

// TestConfigPath 返回测试应加载的配置文件路径：
//   - 若同一 config 目录下存在 config.test.yaml（测试专用，通常把 iam.mode 设为 disabled，
//     使集成/e2e 走 DemoIdentity 的 off 路径），优先返回它；
//   - 否则回退到生产 config.yaml。
//
// 这样测试用独立配置覆盖 iam.mode，无需改动生产 config.yaml / config.prod.yaml。
func TestConfigPath(appName string) string {
	dir := appConfigDir(appName)
	if dir == "" {
		return ""
	}
	testCfg := filepath.Join(dir, "config.test.yaml")
	if _, err := os.Stat(testCfg); err == nil {
		return testCfg
	}
	return filepath.Join(dir, "config.yaml")
}

func (i *baseAppInitializer) Init() error {
	if _, err := os.Stat(i.ConfigPath); err != nil {
		return fmt.Errorf("config file not found: %s, error: %w", i.ConfigPath, err)
	}

	var cfg AppConfig
	gutil.LoadYamlConfig(i.ConfigPath, &cfg)

	// 运行时日志目录环境变量化（D10）：测试也统一写到 <repoRoot>/.local/log（或 APP_LOG_DIR），
	// 避免 config 里 CWD 相对路径（如 ../../../log）在测试包目录(CWD=包目录)下解析到越界目录。
	applyTestLogDir(cfg.Log)

	i.Log = cfg.Log
	i.DBConfigs = cfg.DBConfigs
	i.RedisConfig = cfg.RedisConfig

	if err := i.initLog(); err != nil {
		return fmt.Errorf("init log: %w", err)
	}

	if err := i.initResources(); err != nil {
		return fmt.Errorf("init resources: %w", err)
	}

	return nil
}

func (i *baseAppInitializer) initLog() error {
	logCfg, ok := i.Log["default"]
	if !ok {
		for _, c := range i.Log {
			logCfg = c
			break
		}
	}
	return glog.InitLogger(&logCfg)
}

func (i *baseAppInitializer) initResources() error {
	var gormLogConfig *glog.LogConfig
	if c, ok := i.Log["gorm"]; ok {
		gormLogConfig = &c
	}
	if err := dbclient.InitMultiDB(i.DBConfigs, gormLogConfig); err != nil {
		return fmt.Errorf("init db: %w", err)
	}

	if i.RedisConfig.Addr != "" {
		var redisLogConfig *glog.LogConfig
		if c, ok := i.Log["redis"]; ok {
			redisLogConfig = &c
		}
		if err := dbclient.InitRedis(i.RedisConfig, redisLogConfig); err != nil {
			return fmt.Errorf("init redis: %w", err)
		}
	}

	return nil
}

func (i *baseAppInitializer) Close() error {
	if err := glog.Close(); err != nil {
		return fmt.Errorf("close logger: %w", err)
	}
	return nil
}
