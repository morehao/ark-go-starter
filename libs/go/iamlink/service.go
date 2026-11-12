package iamlink

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/morehao/ark-go-starter/libs/go/dbclient"
	"github.com/morehao/golib/glog"
)

// userMapCacheKeyPrefix 是 iam_user_id → local_uid 映射在 Redis 的 key 前缀。
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

// EnsureLocalUser 返回 iamUserID 对应的本地 uint 用户ID；不存在则原子创建映射。
//
// 查询顺序：Redis 缓存 → 命中直接返回；未命中 → DB 幂等 upsert（首次会新建 local_uid，
// 已存在则返回既有值）→ 回填缓存。personID 用于在插入/更新时落库自然人ID，可为空。
//
// 该函数是鉴权中间件把 rp.Identity(string) 映射为 operator.Operator(uint) 的唯一入口。
func (s *Svc) EnsureLocalUser(ctx context.Context, iamUserID, personID string) (uint, error) {
	if iamUserID == "" {
		return 0, errors.New("iamUserID is empty")
	}

	cacheKey := userMapCacheKeyPrefix + iamUserID
	if uid, ok := s.getCache(ctx, cacheKey); ok {
		return uid, nil
	}

	localUID, err := s.dao.Ensure(ctx, iamUserID, personID)
	if err != nil {
		glog.Errorf(ctx, "[iamlink.Svc.EnsureLocalUser] dao Ensure fail, err:%v, iamUserID:%s", err, iamUserID)
		return 0, err
	}
	s.setCache(ctx, cacheKey, localUID)
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
