package demo

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-go-starter/demo/config"
	"github.com/morehao/ark-iam/contract"
	pkgconfig "github.com/morehao/ark-iam/pkg/config"
	"github.com/morehao/ark-iam/sdk/rp"
	"github.com/morehao/ark-iam/sdk/server"
	"github.com/morehao/golib/glog"
)

// iamRuntime 保存已启动的 IAM 运行时及其对宿主暴露的能力（KeySource/Issuer）。
//
// 它实现 io.Closer，供 cmd/main.go defer 关闭；同时把验签公钥来源与 issuer 暴露给
// 鉴权中间件（同包 internal/middleware 通过 iam() 取用），使业务路由可本地验签。
//
// op 形态持有内嵌 OP 的 app（连接与路由），rp 形态只持有一个远端 JWKS 密钥来源，
// 两者通过 closer 统一释放。
type iamRuntime struct {
	app       *server.App
	keySource contract.KeySource
	issuer    string
	closer    io.Closer // op=app，rp=*rp.Keys；nil 安全
}

// Close 释放运行时持有的资源（内嵌 OP 的 DB/Redis 连接或 rp 的后台刷新协程）。
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

// KeySource 返回运行时的验签公钥来源（op=进程内 OP key set，rp=远端 JWKS）；
// 未装配 auth 时为 nil。仅供装配/测试读取。
func (r *iamRuntime) KeySource() contract.KeySource {
	if r == nil {
		return nil
	}
	return r.keySource
}

// IAMRuntime 是对外只读的 IAM 运行时视图。
type IAMRuntime = iamRuntime

// SetupIAM 按 iam.mode 三态装配 IAM，并（op 形态下）挂载到 engine。
//
//   - disabled：零副作用——不初始化 IAM、不建表、不挂任何路由，直接返回 (nil, nil)，
//     业务鉴权回落到固定 dev 身份（见中间件装配）。
//   - op：本进程内嵌 OIDC Provider。server.New 建立实例（DB 复用主库/独立 DSN，Redis 复用/独立），
//     RunMigrations 建 iam_ 前缀表，Mount 挂 /v1/{platform,tenant,auth} + /oidc + /install，
//     MountConfig 只注册 /iam/config.json（前端登录 UI 由 npm 包提供，Go 侧不再托管静态资源）。
//   - rp：对接外部 ark-iam，仅本地验签。rp.NewKeys 启动期预算一次 JWKS（fail-fast），
//     不建库、不 Mount、不 RunMigrations。
//
// 非法 mode（如 Mode:"bogus"）经 Validate 报错返回。返回的 io.Closer 供 cmd/main.go
// defer 关闭；disabled 时为 nil。
func SetupIAM(engine *gin.Engine) (io.Closer, error) {
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
	case config.ModeOP:
		return setupOP(engine)
	case config.ModeRP:
		return setupRP()
	default:
		// Validate 已拦截，这里仅为完备性兜底。
		return nil, fmt.Errorf("demo.SetupIAM: unsupported iam mode %q", cfg.EffectiveMode())
	}
}

// setupOP 装配并启动内嵌 OP（op 形态）：建库、迁移、挂载业务路由与 /config.json。
func setupOP(engine *gin.Engine) (io.Closer, error) {
	cfg := config.Conf.IAM
	prefix := cfg.GetPrefix()
	dsn := resolveIAMDSN(cfg)
	if dsn == "" {
		return nil, errors.New("demo.SetupIAM: iam dsn is empty and no db_configs to fall back")
	}
	issuer := cfg.Issuer
	if issuer == "" {
		issuer = fmt.Sprintf("http://localhost:%s%s/oidc", config.Conf.Server.Port, prefix)
	}

	app, err := server.New(server.Config{
		Env:          config.Conf.Server.Env,
		DB:           server.DBConfig{DSN: dsn, Service: "iam"},
		Redis:        resolveIAMRedis(cfg),
		Domains:      server.DefaultConsoleDomains,
		SameDatabase: true,
		// Raw 承载 OP 所需的完整配置：issuer 必须与 Mount 前缀一致（{prefix}/oidc），
		// 否则 OP 公布的 issuer 与实际端点不符，RP 侧验签会因 iss 不匹配失败。
		Raw: buildRawConfig(issuer, cfg),
	})
	if err != nil {
		return nil, fmt.Errorf("demo.SetupIAM: server.New failed: %w", err)
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
	// 只注册 /config.json（运行期配置）；前端登录/管理台 UI 由 npm @morehao/ark-iam-react 提供。
	if err := app.MountConfig(engine, prefix); err != nil {
		_ = app.Close()
		return nil, fmt.Errorf("demo.SetupIAM: MountConfig failed: %w", err)
	}

	rt := &iamRuntime{app: app, keySource: app.KeySource(), issuer: issuer, closer: app}
	currentIAM = rt
	glog.Infof(ctx, "[demo.SetupIAM] iam embedded (op) enabled, prefix:%s, issuer:%s", prefix, issuer)
	return rt, nil
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
func resolveIAMRedis(cfg config.IAMConfig) *server.RedisConfig {
	addr := cfg.RedisAddr
	password := cfg.RedisPassword
	if addr == "" {
		addr = config.Conf.RedisConfig.Addr
		password = config.Conf.RedisConfig.Password
	}
	if addr == "" {
		return nil
	}
	return &server.RedisConfig{Service: "iam", Addr: addr, Password: password}
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
	// FrontendLoginURL 是内嵌登录页地址：SSO 无会话时 `/iam/oidc/sso-login` 会 302 到
	// `{FrontendLoginURL}?authRequestID=...`。**必须显式设置**且带挂载前缀（登录页 SPA 在前端
	// `@morehao/ark-iam-react` 包提供，挂载在 `{prefix}/login`）；留空会导致 Location 退化成
	// 相对地址、丢失 authRequestID。
	// 优先用显式配置的前端基址（前后端分离部署，登录页在前端端口）；未配置时回退 issuer 的 host
	// （同源部署）。见 loginURL。
	raw.OIDC.FrontendLoginURL = loginURL(cfg.FrontendURL, issuer, cfg.GetPrefix())
	raw.OIDC.CookieSecure = cfg.CookieSecure
	raw.OIDC.SigningPrivateKeyPEM = cfg.SigningKey
	raw.OIDC.EncryptionKey = cfg.EncryptionKey
	// 引导令牌透传给 OP：CheckBootstrapToken 读取的 config.Conf 正是这份 Raw
	// （apps/auth 在 Mount 时 authconfig.Set(raw)），故必须在此设值，否则 config 里的
	// install.bootstrapToken 不会生效、只有 env BOOTSTRAP_TOKEN 可用。
	raw.Install.BootstrapToken = config.Conf.Install.BootstrapToken
	// 内置控制台 platform_admin_web 的回调白名单：显式配置时覆盖 ark-iam 内置缺省（本地 4001），
	// 让不同端口启动的前端（如 vite 自动退到 3001）也能命中。留空则保持 ark-iam 缺省不动。
	if len(cfg.RedirectURIs) > 0 {
		raw.OIDC.Consoles.PlatformAdminWeb.RedirectURIs = cfg.RedirectURIs
	}
	return raw
}

// loginURL 计算内嵌登录页地址 `{base}{prefix}/login`（与 config.json 的 loginPath 一致）。
//
// base 取值优先级：
//  1. frontendURL 显式配置的前端基址（前后端分离部署，登录页 SPA 由前端提供，如 http://localhost:3000）；
//  2. 空则回退 issuer 的 origin（同源部署，OP 与前端同端口）。
//
// issuer 形如 `http://localhost:8099/iam/oidc`。
func loginURL(frontendURL, issuer, prefix string) string {
	p := prefix
	if p == "" || p == "/" {
		p = ""
	} else {
		p = "/" + strings.Trim(p, "/")
	}
	if base := strings.TrimRight(strings.TrimSpace(frontendURL), "/"); base != "" {
		return base + p + "/login"
	}
	if u, err := url.Parse(issuer); err == nil && u.Scheme != "" && u.Host != "" {
		return u.Scheme + "://" + u.Host + p + "/login"
	}
	return strings.TrimSuffix(issuer, "/") + p + "/login"
}
