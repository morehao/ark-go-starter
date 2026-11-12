package config

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/morehao/golib/glog"
)

// 运行时路径环境变量（D10）：日志、存储目录改为「环境变量优先，缺省落仓库根 .local/」，
// 使 make run / cd apps/api/cmd && go run . / 容器内 /app 三种启动方式得到同一结果，
// 运行时产物不再按 CWD 相对路径散落到仓库任意深度。
const (
	EnvAppLogDir     = "APP_LOG_DIR"     // 日志目录；缺省 <repoRoot>/.local/log
	EnvAppStorageDir = "APP_STORAGE_DIR" // 存储目录；缺省 <repoRoot>/.local/storage
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

// applyRuntimePathOverrides 在加载配置后应用运行时路径覆盖（日志目录 + 存储目录）。
func applyRuntimePathOverrides(cfg *Config) {
	ApplyLogDirOverride(cfg.Log, DefaultLogDir())
	// 存储目录：仅当 APP_STORAGE_DIR 显式设置时覆盖；否则沿用 config 里的 base_dir
	// （真实数据搬迁到 .local/storage 属 P2-T6，届时同步改 config.base_dir）。
	if d := os.Getenv(EnvAppStorageDir); d != "" {
		cfg.FileStorage.Storage.BaseDir = d
	}
}
