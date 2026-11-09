package demo

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-go-starter/demo/config"
	"github.com/morehao/ark-iam/iam"
	pkgconfig "github.com/morehao/ark-iam/pkg/config"
	"github.com/morehao/ark-iam/sdk/rp"
	"github.com/morehao/golib/glog"
)

// iamRuntime 保存已启动的内嵌 IAM 实例及其对宿主暴露的能力（KeySource/Issuer）。
//
// 它实现 io.Closer，供 cmd/main.go defer 关闭；同时把 OP 的进程内公钥来源与 issuer
// 暴露给鉴权中间件（同包 internal/middleware 通过 iam() 取用），使业务路由可本地验签。
type iamRuntime struct {
	app       *iam.App
	keySource rp.KeySource
	issuer    string
}

// Close 释放内嵌 IAM 持有的 DB/Redis 连接，满足 io.Closer。
func (r *iamRuntime) Close() error {
	if r == nil || r.app == nil {
		return nil
	}
	return r.app.Close()
}

// currentIAM 保存当前进程内已启动的内嵌 IAM 运行时（Enable=false 时为 nil）。
var currentIAM *iamRuntime

// CurrentIAM 返回当前已启动的内嵌 IAM 运行时；未启用时返回 nil。
//
// 供鉴权中间件装配读取 OP 公钥来源与 issuer，从而对业务令牌本地验签。
func CurrentIAM() *IAMRuntime { return currentIAM }

// IAMRuntime 是对外只读的内嵌 IAM 运行时视图。
type IAMRuntime = iamRuntime

// SetupIAM 按配置可选地启动内嵌 IAM 并挂载到 engine。
//
// Enable=false 时零副作用：不初始化 IAM、不建表、不挂任何路由，直接返回 (nil, nil)，
// 业务鉴权回落到固定 dev 身份（见中间件装配）。Enable=true 时：
//  1. iam.New 建立 IAM 实例（DB 复用主库/独立 DSN，Redis 复用/独立）；
//  2. RunMigrations 建 iam_ 前缀表（同库撞名检测）；
//  3. Mount 在 {prefix} 下挂 /v1/{platform,tenant,auth} + /oidc + /install；
//  4. MountFrontend 在 {prefix} 下托管内嵌登录/初始化页。
//
// 返回的 io.Closer 供 cmd/main.go defer 关闭；未启用时为 nil。
func SetupIAM(engine *gin.Engine) (io.Closer, error) {
	cfg := config.Conf.IAM
	if !cfg.Enable {
		// 明确复位运行时，保证"关闭即无 IAM"的语义（避免上一次启用残留）。
		currentIAM = nil
		return nil, nil
	}

	prefix := cfg.GetPrefix()
	dsn := resolveIAMDSN(cfg)
	if dsn == "" {
		return nil, errors.New("demo.SetupIAM: iam dsn is empty and no db_configs to fall back")
	}
	issuer := cfg.Issuer
	if issuer == "" {
		issuer = fmt.Sprintf("http://localhost:%s%s/oidc", config.Conf.Server.Port, prefix)
	}

	app, err := iam.New(iam.Config{
		Env:          config.Conf.Server.Env,
		DB:           iam.DBConfig{DSN: dsn, Service: "iam"},
		Redis:        resolveIAMRedis(cfg),
		Domains:      iam.DefaultConsoleDomains,
		SameDatabase: true,
		// Raw 承载 OP 所需的完整配置：issuer 必须与 Mount 前缀一致（{prefix}/oidc），
		// 否则 OP 公布的 issuer 与实际端点不符，RP 侧验签会因 iss 不匹配失败。
		Raw: buildRawConfig(issuer, cfg),
	})
	if err != nil {
		return nil, fmt.Errorf("demo.SetupIAM: iam.New failed: %w", err)
	}

	ctx := context.Background()
	if err := app.RunMigrations(ctx); err != nil {
		_ = app.Close()
		return nil, fmt.Errorf("demo.SetupIAM: RunMigrations failed: %w", err)
	}

	if err := app.Mount(engine.Group(prefix)); err != nil {
		_ = app.Close()
		return nil, fmt.Errorf("demo.SetupIAM: Mount failed: %w", err)
	}
	if err := app.MountFrontend(engine, prefix); err != nil {
		_ = app.Close()
		return nil, fmt.Errorf("demo.SetupIAM: MountFrontend failed: %w", err)
	}

	rt := &iamRuntime{app: app, keySource: app.KeySource(), issuer: issuer}
	currentIAM = rt
	glog.Infof(ctx, "[demo.SetupIAM] iam embedded enabled, prefix:%s, issuer:%s", prefix, issuer)
	return rt, nil
}

// resolveIAMDSN 解析 IAM 数据库 DSN：显式配置优先，否则复用主库（db_configs 第一个）。
func resolveIAMDSN(cfg config.IAMConfig) string {
	if cfg.DSN != "" {
		return cfg.DSN
	}
	if len(config.Conf.DBConfigs) > 0 {
		return config.Conf.DBConfigs[0].URL
	}
	return ""
}

// resolveIAMRedis 解析 IAM Redis 配置：未配置地址时返回 nil（由 IAM 自行处理/不启用缓存）。
func resolveIAMRedis(cfg config.IAMConfig) *iam.RedisConfig {
	addr := cfg.RedisAddr
	password := cfg.RedisPassword
	if addr == "" {
		addr = config.Conf.RedisConfig.Addr
		password = config.Conf.RedisConfig.Password
	}
	if addr == "" {
		return nil
	}
	return &iam.RedisConfig{Service: "iam", Addr: addr, Password: password}
}

// buildRawConfig 组装 OP 所需的完整配置（issuer/签名密钥/cookie 等）。
//
// dev 环境签名密钥/加密密钥可缺省（IAM 自动生成/放行）；非 dev 由 IAM 的
// validateNonDevSecurityConfig 强制要求显式配置（fail-closed）。
func buildRawConfig(issuer string, cfg config.IAMConfig) *pkgconfig.Config {
	raw := &pkgconfig.Config{}
	raw.Server.Env = config.Conf.Server.Env
	raw.Server.Port = config.Conf.Server.Port
	raw.OIDC.Issuer = issuer
	raw.OIDC.CookieSecure = cfg.CookieSecure
	raw.OIDC.SigningPrivateKeyPEM = cfg.SigningKey
	raw.OIDC.EncryptionKey = cfg.EncryptionKey
	return raw
}
