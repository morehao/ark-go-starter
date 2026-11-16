package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/morehao/golib/glog"
)

// 运行时路径环境变量（D10）：日志、存储目录改为「环境变量优先，缺省落仓库根 .local/」，
// 使 make run / cd apps/api/cmd && go run . / 容器内 /app 三种启动方式得到同一结果，
// 运行时产物不再按 CWD 相对路径散落到仓库任意深度。
const (
	EnvAppLogDir      = "APP_LOG_DIR"      // 日志目录；缺省 <repoRoot>/.local/log
	EnvAppStorageDir  = "APP_STORAGE_DIR"  // 存储目录；缺省 <repoRoot>/.local/storage
	EnvAppLogInstance = "APP_LOG_INSTANCE" // 日志实例标识；多实例/多副本共享日志目录时必须设置，用于生成每实例独立的日志文件名
)

// RepoRoot 从本文件位置向上回溯找到包含 go.work 的目录（仓库根）。
func RepoRoot() string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return ""
	}
	dir := filepath.Dir(filename)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.work")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// DefaultLogDir 返回日志目录：APP_LOG_DIR 优先，否则 <repoRoot>/.local/log。
func DefaultLogDir() string {
	if d := os.Getenv(EnvAppLogDir); d != "" {
		return d
	}
	if root := RepoRoot(); root != "" {
		return filepath.Join(root, ".local", "log")
	}
	return filepath.Join(".local", "log")
}

// DefaultStorageDir 返回存储目录：APP_STORAGE_DIR 优先，否则 <repoRoot>/.local/storage。
func DefaultStorageDir() string {
	if d := os.Getenv(EnvAppStorageDir); d != "" {
		return d
	}
	if root := RepoRoot(); root != "" {
		return filepath.Join(root, ".local", "storage")
	}
	return filepath.Join(".local", "storage")
}

// ApplyLogDirOverride 把所有日志配置里的文件 writer 目录统一改为 logDir。
// 只改 type=file 的 writer；console 等其它类型不动。logDir 为空时不改。
func ApplyLogDirOverride(logs map[string]glog.LogConfig, logDir string) {
	if logDir == "" {
		return
	}
	for name, lc := range logs {
		for i := range lc.Writers {
			if lc.Writers[i].Type == glog.WriterFile {
				lc.Writers[i].Dir = logDir
			}
		}
		logs[name] = lc
	}
}

// LogInstanceSuffix 返回日志实例标识：APP_LOG_INSTANCE 优先，其次 HOSTNAME（K8s 下即 pod 名）。
// 二者都为空时返回空串，表示单实例场景、不改文件名。
func LogInstanceSuffix() string {
	if s := os.Getenv(EnvAppLogInstance); s != "" {
		return s
	}
	return os.Getenv("HOSTNAME")
}

// ApplyLogInstanceSuffix 给每个文件 writer 的文件名加上实例标识，避免多实例/多副本
// 共享同一日志目录时写同一个文件导致丢日志（glog 升级后日志不再按天分目录，多进程写
// 同一文件由 lumberjack 滚动会互相覆盖）。instance 为空时不改（单实例默认行为不变）。
// 生成的文件名为 <name>-<instance><suffix>.log（suffix 由 glog 追加，_full/_wf）。
func ApplyLogInstanceSuffix(logs map[string]glog.LogConfig, instance string) {
	if instance == "" {
		return
	}
	for name, lc := range logs {
		for i := range lc.Writers {
			if lc.Writers[i].Type != glog.WriterFile {
				continue
			}
			base := lc.Writers[i].FileName
			if base == "" {
				// 未显式配置时沿用 EffectiveFileName 的默认（<service>.log）；
				// service 为空时 glog 用 DefaultServiceName，这里保持一致。
				svc := lc.Service
				if svc == "" {
					svc = glog.DefaultServiceName
				}
				base = svc + ".log"
			}
			ext := filepath.Ext(base)
			baseName := base[:len(base)-len(ext)]
			// 幂等：已带本实例后缀则跳过，避免重复调用叠加（如配置重载）。
			if strings.HasSuffix(baseName, "-"+instance) {
				continue
			}
			lc.Writers[i].FileName = baseName + "-" + instance + ext
		}
		logs[name] = lc
	}
}

// applyRuntimePathOverrides 在加载配置后应用运行时路径覆盖（日志目录 + 存储目录）。
func applyRuntimePathOverrides(cfg *Config) {
	ApplyLogDirOverride(cfg.Log, DefaultLogDir())
	ApplyLogInstanceSuffix(cfg.Log, LogInstanceSuffix())
	// 存储目录：仅当 APP_STORAGE_DIR 显式设置时覆盖；否则沿用 config 里的 base_dir
	// （真实数据搬迁到 .local/storage 属 P2-T6，届时同步改 config.base_dir）。
	if d := os.Getenv(EnvAppStorageDir); d != "" {
		cfg.FileStorage.Storage.BaseDir = d
	}
}
