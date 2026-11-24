package demo

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-go-starter/apps/api/config"
	"github.com/morehao/ark-iam/contract"
	"github.com/morehao/ark-iam/sdk/rp"
	"github.com/morehao/golib/glog"
)

// iamRuntime 保存已启动的 IAM 运行时及其对宿主暴露的能力（KeySource/Issuer）。
//
// 它实现 io.Closer，供 cmd/main.go defer 关闭；同时把验签公钥来源与 issuer 暴露给
// 鉴权中间件（同包 internal/middleware 通过 iam() 取用），使业务路由可本地验签。
//
// rp 形态持有一个远端 JWKS 密钥来源，通过 closer 统一释放。
type iamRuntime struct {
	keySource contract.KeySource
	issuer    string
	closer    io.Closer // rp=*rp.Keys；nil 安全
}

// Close 释放运行时持有的资源（rp 的后台刷新协程）。
func (r *iamRuntime) Close() error {
	if r == nil {
		return nil
	}
	if r.closer != nil {
		return r.closer.Close()
	}
	return nil
}

// currentIAM 保存当前进程内已启动的 IAM 运行时（disabled 时为 nil）。
var currentIAM *iamRuntime

// CurrentIAM 返回当前已启动的 IAM 运行时；未启用时返回 nil。
//
// 供鉴权中间件装配读取验签公钥来源与 issuer，从而对业务令牌本地验签。
func CurrentIAM() *IAMRuntime { return currentIAM }

// KeySource 返回运行时的验签公钥来源（rp=远端 JWKS）；未装配 auth 时为 nil。仅供装配/测试读取。
func (r *iamRuntime) KeySource() contract.KeySource {
	if r == nil {
		return nil
	}
	return r.keySource
}

// IAMRuntime 是对外只读的 IAM 运行时视图。
type IAMRuntime = iamRuntime

// SetupIAM 按 iam.mode 两态装配 IAM。
//
//   - disabled：零副作用——不初始化 IAM、不建表、不挂任何路由，直接返回 (nil, nil)，
//     业务鉴权回落到固定 dev 身份（见中间件装配）。
//   - rp：对接外部 ark-iam，仅本地验签。rp.NewKeys 启动期预算一次 JWKS（fail-fast），
//     不建库、不 Mount、不 RunMigrations。
//
// 非法 mode（如 Mode:"bogus"，或已下线的 op）经 Validate 报错返回。返回的 io.Closer 供
// cmd/main.go defer 关闭；disabled 时为 nil。
//
// 注意：op 下线后 disabled/rp 两态均不再向 engine 挂载任何路由，故 engine 参数当前未被使用；
// 保留该参数以维持装配点签名稳定（未来若 rp 需挂载 /config.json 等运行期配置可直接复用）。
func SetupIAM(_ *gin.Engine) (io.Closer, error) {
	cfg := config.Conf.IAM
	// 先校验再分派：非法 mode 明确报错，避免静默回落成 disabled。
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("demo.SetupIAM: invalid iam config: %w", err)
	}

	switch cfg.EffectiveMode() {
	case config.ModeDisabled:
		// 明确复位运行时，保证"关闭即无 IAM"的语义（避免上一次启用残留）。
		currentIAM = nil
		return nil, nil
	case config.ModeRP:
		return setupRP()
	default:
		// Validate 已拦截，这里仅为完备性兜底。
		return nil, fmt.Errorf("demo.SetupIAM: unsupported iam mode %q", cfg.EffectiveMode())
	}
}

// setupRP 装配 rp 形态：只构造远端 JWKS 验签密钥来源，不建库、不挂路由、不迁移。
//
// jwksURL 解析优先级：显式 cfg.JWKSURL > cfg.Issuer（由 rp 内部按标准 discovery →
// {issuer}/keys → {issuer}/.well-known/jwks.json 候选探测）。rp.NewKeys 启动期同步预取
// 一次（失败 fail-fast），故启动即可保证能验签。
func setupRP() (io.Closer, error) {
	cfg := config.Conf.IAM
	jwksURL := cfg.JWKSURL
	if jwksURL == "" {
		jwksURL = cfg.Issuer
	}
	if jwksURL == "" {
		return nil, errors.New("demo.SetupIAM: rp mode requires iam.jwks_url or iam.issuer")
	}

	ctx := context.Background()
	keys, err := rp.NewKeys(ctx, jwksURL)
	if err != nil {
		return nil, fmt.Errorf("demo.SetupIAM: rp.NewKeys failed: %w", err)
	}

	rt := &iamRuntime{keySource: keys, issuer: cfg.Issuer, closer: rpKeysCloser{keys}}
	currentIAM = rt
	glog.Infof(ctx, "[demo.SetupIAM] iam remote (rp) enabled, jwksURL:%s, issuer:%s", jwksURL, cfg.Issuer)
	return rt, nil
}

// rpKeysCloser 把 *rp.Keys（Close() 无返回值）适配成 io.Closer。
type rpKeysCloser struct{ keys *rp.Keys }

// Close 停止 rp 密钥来源的后台刷新协程，满足 io.Closer。
func (c rpKeysCloser) Close() error {
	if c.keys != nil {
		c.keys.Close()
	}
	return nil
}
