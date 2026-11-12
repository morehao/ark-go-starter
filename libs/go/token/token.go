// Package token 提供令牌哈希与基于 Redis 的黑名单能力。
package token

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	"github.com/morehao/ark-go-starter/libs/go/dbclient"
	"github.com/morehao/golib/glog"
)

const (
	// TokenBlacklistKeyPrefix 访问令牌黑名单在 Redis 的 key 前缀。
	TokenBlacklistKeyPrefix = "token:blacklist:"
	// RefreshTokenBlacklistKeyPrefix 刷新令牌黑名单在 Redis 的 key 前缀。
	RefreshTokenBlacklistKeyPrefix = "refreshToken:blacklist:"
	// TokenExpireDuration 访问令牌黑名单的默认过期时长。
	TokenExpireDuration = 24 * time.Hour
	// RefreshTokenExpireDuration 刷新令牌黑名单的默认过期时长。
	RefreshTokenExpireDuration = 7 * 24 * time.Hour
)

// HashToken 返回 token 的 SHA-256 十六进制摘要，用作黑名单 key。
func HashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

// AddTokenToBlacklist 将访问令牌写入 Redis 黑名单，按 expireDuration 过期。
func AddTokenToBlacklist(ctx context.Context, token string, expireDuration time.Duration) error {
	token = strings.TrimPrefix(token, "Bearer ")
	if token == "" {
		return nil
	}
	key := TokenBlacklistKeyPrefix + HashToken(token)
	if err := dbclient.RedisCli.Set(ctx, key, "1", expireDuration).Err(); err != nil {
		glog.Errorf(ctx, "[token.AddTokenToBlacklist] Redis Set token fail, err:%v", err)
		return err
	}
	return nil
}

// AddRefreshTokenToBlacklist 将刷新令牌写入 Redis 黑名单，按默认过期时长过期。
func AddRefreshTokenToBlacklist(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	token = strings.TrimPrefix(token, "Bearer ")
	if token == "" {
		return nil
	}
	key := RefreshTokenBlacklistKeyPrefix + HashToken(token)
	if err := dbclient.RedisCli.Set(ctx, key, "1", RefreshTokenExpireDuration).Err(); err != nil {
		glog.Errorf(ctx, "[token.AddRefreshTokenToBlacklist] Redis Set refresh token fail, err:%v", err)
		return err
	}
	return nil
}

// IsTokenBlacklisted 判断访问令牌是否已在 Redis 黑名单中；Redis 未初始化时返回 false。
func IsTokenBlacklisted(ctx context.Context, token string) bool {
	if dbclient.RedisCli == nil {
		return false
	}
	token = strings.TrimPrefix(token, "Bearer ")
	if token == "" {
		return false
	}
	key := TokenBlacklistKeyPrefix + HashToken(token)
	exists, err := dbclient.RedisCli.Exists(ctx, key).Result()
	if err != nil {
		return false
	}
	return exists > 0
}
