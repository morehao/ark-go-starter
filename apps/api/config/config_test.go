package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testIAMDefaultPrefix = "/iam"

// TestIAMConfig_EffectiveMode 覆盖三态 mode 的归一化、旧 enable 兼容与非法值处理。
func TestIAMConfig_EffectiveMode(t *testing.T) {
	cases := []struct {
		name string
		cfg  IAMConfig
		want string
	}{
		{"默认零值 → disabled", IAMConfig{}, ModeDisabled},
		{"显式 mode:disabled", IAMConfig{Mode: ModeDisabled}, ModeDisabled},
		{"mode:op", IAMConfig{Mode: ModeOP}, ModeOP},
		{"mode:rp", IAMConfig{Mode: ModeRP}, ModeRP},
		{"大小写归一 OP", IAMConfig{Mode: "OP"}, ModeOP},
		{"空白与大写归一 \"  RP \"", IAMConfig{Mode: "  RP "}, ModeRP},
		{"旧 enable:true → op", IAMConfig{Enable: true}, ModeOP},
		{"enable:false → disabled", IAMConfig{Enable: false}, ModeDisabled},
		{"mode 优先于 enable:true", IAMConfig{Mode: ModeRP, Enable: true}, ModeRP},
		{"非法 mode 原样返回（交由 Validate 报错）", IAMConfig{Mode: "Bogus"}, "bogus"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.cfg.EffectiveMode())
		})
	}
}

// TestIAMConfig_Validate 覆盖合法/非法 mode 的校验结果。
func TestIAMConfig_Validate(t *testing.T) {
	t.Run("默认 disabled 合法", func(t *testing.T) {
		assert.NoError(t, IAMConfig{}.Validate())
	})
	t.Run("op 合法", func(t *testing.T) {
		assert.NoError(t, IAMConfig{Mode: ModeOP}.Validate())
	})
	t.Run("rp 合法", func(t *testing.T) {
		assert.NoError(t, IAMConfig{Mode: ModeRP}.Validate())
	})
	t.Run("旧 enable:true 合法", func(t *testing.T) {
		assert.NoError(t, IAMConfig{Enable: true}.Validate())
	})
	t.Run("非法 mode 报错", func(t *testing.T) {
		err := IAMConfig{Mode: "bogus"}.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid iam.mode")
	})
	t.Run("非法 mode 即便 enable:true 也报错", func(t *testing.T) {
		require.Error(t, IAMConfig{Mode: "bogus", Enable: true}.Validate())
	})
}

// TestIAMConfig_GetPrefix 回归：保留既有前缀归一逻辑（不以 mode 改造为转移）。
func TestIAMConfig_GetPrefix(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"", testIAMDefaultPrefix},
		{testIAMDefaultPrefix, testIAMDefaultPrefix},
		{"iam", testIAMDefaultPrefix},
		{testIAMDefaultPrefix + "/", testIAMDefaultPrefix},
		{"/custom/", "/custom"},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, IAMConfig{Prefix: tc.in}.GetPrefix())
	}
}
