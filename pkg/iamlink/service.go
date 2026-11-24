package iamlink

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/morehao/ark-go-starter/pkg/dbclient"
	"github.com/morehao/golib/glog"
)

// userMapCacheKeyPrefix 是 (issuer, iam_user_id) → local_uid 映射在 Redis 的 key 前缀。
const userMapCacheKeyPrefix = "iamlink:user:"

// userMapCacheTTL 是映射缓存的有效期。命中即省一次 DB 查询；未命中回源后回填。
const userMapCacheTTL = 30 * time.Minute

// Svc 是 IAM 身份映射服务：把外部 IAM 字符串用户ID 映射为本地 BIGINT 用户ID。
type Svc struct {
	dao *MapDao
}

// NewSvc 构造 IAM 身份映射服务。
func NewSvc() *Svc {
	return &Svc{dao: NewMapDao()}
}

// cacheKey 组装 (issuer, iamUserID) 的 Redis 缓存键。
//
// 用冒号分隔 issuer 与 iamUserID 并各占一段，避免「issuer+iamUserID」直接拼接产生的歧义
// （如 issuer="a"、id="bc" 与 issuer="ab"、id="c" 拼出同一串）；不同 issuer 下相同 sub 的
// 缓存键必不相同。
func cacheKey(issuer, iamUserID string) string {
	return userMapCacheKeyPrefix + issuer + ":" + iamUserID
}

// EnsureLocalUser 返回 (issuer, iamUserID) 对应的本地 uint 用户ID；不存在则原子创建映射。
//
// 查询顺序：Redis 缓存 → 命中直接返回；未命中 → DB 幂等 upsert（首次会新建 local_uid，
// 已存在则返回既有值）→ 回填缓存。personID 用于在插入/更新时落库自然人ID，可为空。
//
// issuer 取装配层已知的 cfg.issuer（中间件已用它精确校验 token 的 iss），不从 token 或
// rp.Identity 读取：不同 IDP 下相同 sub 必须映射到不同的本地用户，故 issuer 是映射键的一部分。
//
// 该函数是鉴权中间件把 rp.Identity(string) 映射为 operator.Operator(uint) 的唯一入口。
func (s *Svc) EnsureLocalUser(ctx context.Context, issuer, iamUserID, personID string) (uint, error) {
	if iamUserID == "" {
		return 0, errors.New("iamUserID is empty")
	}

	key := cacheKey(issuer, iamUserID)
	if uid, ok := s.getCache(ctx, key); ok {
		return uid, nil
	}

	localUID, err := s.dao.Ensure(ctx, issuer, iamUserID, personID)
	if err != nil {
		glog.Errorf(ctx, "[iamlink.Svc.EnsureLocalUser] dao Ensure fail, err:%v, issuer:%s, iamUserID:%s", err, issuer, iamUserID)
		return 0, err
	}
	s.setCache(ctx, key, localUID)
	return localUID, nil
}

// getCache 从 Redis 读取映射；缓存未初始化/未命中/异常均返回 ok=false（回源 DB）。
func (s *Svc) getCache(ctx context.Context, key string) (uint, bool) {
	if dbclient.RedisCli == nil {
		return 0, false
	}
	raw, err := dbclient.RedisCli.Get(ctx, key).Result()
	if err != nil {
		return 0, false
	}
	uid, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0, false
	}
	return uint(uid), true
}

// setCache 把映射写入 Redis；缓存未初始化或写失败只记日志、不影响主流程。
func (s *Svc) setCache(ctx context.Context, key string, localUID uint) {
	if dbclient.RedisCli == nil {
		return
	}
	if err := dbclient.RedisCli.Set(ctx, key, strconv.FormatUint(uint64(localUID), 10), userMapCacheTTL).Err(); err != nil {
		glog.Errorf(ctx, "[iamlink.Svc.setCache] Redis Set fail, err:%v, key:%s", err, key)
	}
}
