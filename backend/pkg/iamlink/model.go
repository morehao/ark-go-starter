// Package iamlink 提供 ark-iam 外部字符串身份与本站本地 BIGINT 用户ID 之间的映射能力。
//
// ark-iam 的 rp.Identity.UserID 是 string（如 "user-2222"），而本站 operator.Operator
// 与业务表审计列（created_by/updated_by/deleted_by）均为 BIGINT（uint）。本包用一张
// 映射表 biz_iam_user_map 把「外部 string 身份」与「本地 uint 主体」解耦：鉴权中间件解析出
// IAM userID 后调用 EnsureLocalUser 得到 local_uid，再写入 gcontext.KeyUserID 供业务侧按
// uint 读取，从而对现有 operator/业务表/前端零侵入。
//
// DDL 见 backend/scripts/sql/table_init.sql（biz_iam_user_map）。
package iamlink

import "time"

// 数据表名常量（禁止硬编码）。
const (
	// TableNameIAMUserMap 是 IAM 身份映射表的物理表名。
	TableNameIAMUserMap = "biz_iam_user_map"
)

// IAMUserMapEntity 是 IAM 身份映射表结构体。
//
// 该表不参与 AutoMigrate，建表只走 backend/scripts/sql/table_init.sql，字段与脚本一一对应。
// 不设 deleted_at：身份映射一旦建立即长期有效，不支持软删除。
type IAMUserMapEntity struct {
	IAMUserID string    `gorm:"column:iam_user_id;type:varchar(128);primaryKey;comment:IAM 侧字符串用户ID（rp.Identity.UserID）"`
	LocalUID  uint      `gorm:"column:local_uid;type:bigint;not null;comment:本地BIGINT用户ID（映射目标）"`
	PersonID  string    `gorm:"column:person_id;type:varchar(128);not null;default:'';comment:IAM 侧自然人ID（rp.Identity.PersonID）"`
	CreatedAt time.Time `gorm:"column:created_at;comment:创建时间"`
	UpdatedAt time.Time `gorm:"column:updated_at;comment:更新时间"`
}

// TableName 返回 IAM 身份映射表的物理表名。
func (IAMUserMapEntity) TableName() string {
	return TableNameIAMUserMap
}
