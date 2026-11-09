package iamlink

import (
	"context"
	"errors"
	"fmt"

	"github.com/morehao/ark-go-starter/pkg/dbclient"
	"github.com/morehao/golib/glog"
	"gorm.io/gorm"
)

// MapDao 是 IAM 身份映射表的数据访问对象。
//
// 该表无软删除列、主键是 iam_user_id（string），且首次登录需 INSERT ... ON CONFLICT ... RETURNING
// 的原子 upsert，故不复用 golib 的通用 gormdao（其默认假设 deleted_at 软删除与自增主键），直接用 gorm 实现。
type MapDao struct {
	getDB func(ctx context.Context) *gorm.DB
}

// NewMapDao 构造 IAM 身份映射表 DAO（默认取 demo 应用库连接）。
func NewMapDao() *MapDao {
	return &MapDao{getDB: dbclient.DemoDB}
}

// GetByIAMUserID 按 IAM 字符串用户ID 查询映射；不存在返回 (nil, nil)。
func (d *MapDao) GetByIAMUserID(ctx context.Context, iamUserID string) (*IAMUserMapEntity, error) {
	var entity IAMUserMapEntity
	err := d.getDB(ctx).Where("iam_user_id = ?", iamUserID).Take(&entity).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query biz_iam_user_map by iam_user_id failed: %w", err)
	}
	return &entity, nil
}

// ensureSQL 是幂等 upsert 语句：iam_user_id 已存在则刷新 person_id/updated_at 并原样返回既有
// local_uid；不存在则插入一行，local_uid 由列默认值 nextval('biz_iam_user_map_local_uid_seq')
// 原子分配。单条语句完成，天然并发安全。
const ensureSQL = `INSERT INTO biz_iam_user_map (iam_user_id, person_id)
VALUES (?, ?)
ON CONFLICT (iam_user_id) DO UPDATE
   SET person_id = EXCLUDED.person_id, updated_at = now()
RETURNING local_uid`

// Ensure 原子地确保 iamUserID 有映射并返回其 local_uid。
func (d *MapDao) Ensure(ctx context.Context, iamUserID, personID string) (uint, error) {
	var localUID uint
	if err := d.getDB(ctx).Raw(ensureSQL, iamUserID, personID).Scan(&localUID).Error; err != nil {
		return 0, fmt.Errorf("upsert biz_iam_user_map failed: %w", err)
	}
	if localUID == 0 {
		glog.Errorf(ctx, "[iamlink.mapDao.Ensure] got zero local_uid after upsert, iamUserID:%s", iamUserID)
		return 0, errors.New("iam user map returned zero local_uid")
	}
	return localUID, nil
}
