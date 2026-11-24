package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIAMConfig_EffectiveMode 覆盖两态 mode 的归一化与非法值处理。
func TestIAMConfig_EffectiveMode(t *testing.T) {
	cases := []struct {
		name string
		cfg  IAMConfig
		want string
	}{
		{"默认零值 → disabled", IAMConfig{}, ModeDisabled},
		{"显式 mode:disabled", IAMConfig{Mode: ModeDisabled}, ModeDisabled},
		{"mode:rp", IAMConfig{Mode: ModeRP}, ModeRP},
		{"空白与大写归一 \"  RP \"", IAMConfig{Mode: "  RP "}, ModeRP},
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
	t.Run("rp 合法", func(t *testing.T) {
		assert.NoError(t, IAMConfig{Mode: ModeRP}.Validate())
	})
	t.Run("op 被拒绝", func(t *testing.T) {
		require.Error(t, IAMConfig{Mode: "op"}.Validate())
	})
	t.Run("非法 mode 报错", func(t *testing.T) {
		err := IAMConfig{Mode: "bogus"}.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid iam.mode")
	})
}
