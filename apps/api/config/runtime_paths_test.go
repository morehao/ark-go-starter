package config

import (
	"testing"

	"github.com/morehao/golib/glog"
	"github.com/stretchr/testify/assert"
)

const (
	testLogName     = "default"
	testLogInstance = "pod-a"
	testLogService  = "demo"
)

// TestApplyLogInstanceSuffix 覆盖多实例日志文件名加实例标识逻辑（glog 升级去日期目录后，
// 多副本共享目录需每实例独立文件，否则 lumberjack 跨进程滚动会丢日志）。
func TestApplyLogInstanceSuffix(t *testing.T) {
	cases := []struct {
		name     string
		logs     map[string]glog.LogConfig
		instance string
		want     map[string]string // key: log name, value: 该 log 下第一个 file writer 的 FileName
	}{
		{
			name: "instance 为空不改（单实例默认行为）",
			logs: map[string]glog.LogConfig{
				testLogName: {Service: testLogService, Writers: []glog.WriterConfig{{Type: glog.WriterFile}}},
			},
			instance: "",
			want:     map[string]string{testLogName: ""},
		},
		{
			name: "显式 FileName 带实例后缀",
			logs: map[string]glog.LogConfig{
				testLogName: {Writers: []glog.WriterConfig{{Type: glog.WriterFile, FileName: "app.log"}}},
			},
			instance: testLogInstance,
			want:     map[string]string{testLogName: "app-pod-a.log"},
		},
		{
			name: "未配 FileName 时用 service 作基名",
			logs: map[string]glog.LogConfig{
				testLogName: {Service: testLogService, Writers: []glog.WriterConfig{{Type: glog.WriterFile}}},
			},
			instance: testLogInstance,
			want:     map[string]string{testLogName: "demo-pod-a.log"},
		},
		{
			name: "service 为空时回退 DefaultServiceName",
			logs: map[string]glog.LogConfig{
				testLogName: {Writers: []glog.WriterConfig{{Type: glog.WriterFile}}},
			},
			instance: testLogInstance,
			want:     map[string]string{testLogName: glog.DefaultServiceName + "-pod-a.log"},
		},
		{
			name: "console writer 不动",
			logs: map[string]glog.LogConfig{
				testLogName: {Service: testLogService, Writers: []glog.WriterConfig{
					{Type: glog.WriterConsole},
					{Type: glog.WriterFile},
				}},
			},
			instance: testLogInstance,
			want:     map[string]string{testLogName: "demo-pod-a.log"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ApplyLogInstanceSuffix(tc.logs, tc.instance)
			for name, wantFile := range tc.want {
				lc := tc.logs[name]
				var got string
				for _, w := range lc.Writers {
					if w.Type == glog.WriterFile {
						got = w.FileName
						break
					}
				}
				assert.Equal(t, wantFile, got, "log %q FileName", name)
			}
		})
	}
}

// TestApplyLogInstanceSuffix_Idempotent 验证重复调用不会重复叠加实例后缀。
func TestApplyLogInstanceSuffix_Idempotent(t *testing.T) {
	logs := map[string]glog.LogConfig{
		testLogName: {Service: testLogService, Writers: []glog.WriterConfig{{Type: glog.WriterFile}}},
	}
	ApplyLogInstanceSuffix(logs, testLogInstance)
	ApplyLogInstanceSuffix(logs, testLogInstance)
	assert.Equal(t, "demo-pod-a.log", logs[testLogName].Writers[0].FileName)
}

// TestLogInstanceSuffix 验证环境变量优先级：APP_LOG_INSTANCE > HOSTNAME。
func TestLogInstanceSuffix(t *testing.T) {
	t.Setenv(EnvAppLogInstance, "")
	t.Setenv("HOSTNAME", "host-x")
	assert.Equal(t, "host-x", LogInstanceSuffix())

	t.Setenv(EnvAppLogInstance, "inst-y")
	assert.Equal(t, "inst-y", LogInstanceSuffix(), "APP_LOG_INSTANCE 优先于 HOSTNAME")
}
