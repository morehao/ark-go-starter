-- ============================================================================
-- 初始化脚本（PostgreSQL）
--
-- 本项目以 PostgreSQL 为样板库。首次使用：
--   createdb demo
--   psql -d demo -f backend/scripts/sql/table_init.sql
--
-- 注意：product 表名在 PostgreSQL 中非关键字，SQL 中可直接写 product（无需像保留字那样加双引号）。
-- Go 侧 model.TableNameProduct 是**不带引号**的逻辑表名，传给 gormdao.NewDao 即可，
-- 生成的 SQL 形如 SELECT count(*) FROM "product" WHERE "product"."deleted_at" IS NULL。
--
-- 其余表（core_file、core_file_upload）由 golib/filestore 在应用启动时 AutoMigrate 创建。
--
-- 字段注释规范（见 docs/agents/backend-conventions.md）：每个字段都必须有行内 `--` 注释，
-- 并在建表后用 `COMMENT ON COLUMN` 把同一份说明写入库元数据（psql 里 \d+ 表名 可见）。
-- 两者必须保持一致，新增/改名字段时同步修改。
-- ============================================================================

-- 数据库若尚未创建，可先执行（已存在时会报错，可忽略）：
-- CREATE DATABASE demo;

-- ----------------------------------------------------------------------------
-- 商品表
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS product
(
    id          BIGSERIAL     PRIMARY KEY,               -- 自增ID
    name        VARCHAR(255)  NOT NULL DEFAULT '',       -- 商品名称
    category_id BIGINT        NOT NULL DEFAULT 0,        -- 分类id
    price       NUMERIC(10,2) NOT NULL DEFAULT 0,        -- 价格
    created_at  TIMESTAMPTZ   NOT NULL DEFAULT now(),    -- 创建时间
    updated_at  TIMESTAMPTZ   NOT NULL DEFAULT now(),    -- 更新时间（由 GORM 维护）
    deleted_at  TIMESTAMPTZ            DEFAULT NULL,     -- 删除时间（软删除）
    created_by  BIGINT        NOT NULL DEFAULT 0,        -- 创建人id
    updated_by  BIGINT        NOT NULL DEFAULT 0,        -- 更新人id
    deleted_by  BIGINT        NOT NULL DEFAULT 0         -- 删除人id
);

CREATE INDEX IF NOT EXISTS idx_product_deleted_at ON product (deleted_at);

COMMENT ON TABLE product IS '商品表';
COMMENT ON COLUMN product.id IS '自增ID';
COMMENT ON COLUMN product.name IS '商品名称';
COMMENT ON COLUMN product.category_id IS '分类id';
COMMENT ON COLUMN product.price IS '价格';
COMMENT ON COLUMN product.created_at IS '创建时间';
COMMENT ON COLUMN product.updated_at IS '更新时间（由 GORM 维护）';
COMMENT ON COLUMN product.deleted_at IS '删除时间（软删除）';
COMMENT ON COLUMN product.created_by IS '创建人id';
COMMENT ON COLUMN product.updated_by IS '更新人id';
COMMENT ON COLUMN product.deleted_by IS '删除人id';

-- ----------------------------------------------------------------------------
-- 用户登录日志表
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS user_login_log
(
    id         BIGSERIAL    PRIMARY KEY,               -- 自增ID
    user_id    BIGINT       NOT NULL,                  -- 用户ID
    login_ip   VARCHAR(32)           DEFAULT NULL,     -- 登录IP地址
    user_agent VARCHAR(512)          DEFAULT NULL,     -- 用户代理信息
    login_time TIMESTAMPTZ  NOT NULL DEFAULT now(),    -- 登录时间
    created_at TIMESTAMPTZ  NOT NULL DEFAULT now(),    -- 创建时间
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT now(),    -- 更新时间（由 GORM 维护）
    deleted_at TIMESTAMPTZ           DEFAULT NULL,     -- 删除时间
    created_by BIGINT       NOT NULL,                  -- 创建人ID
    updated_by BIGINT       NOT NULL,                  -- 更新人ID
    deleted_by BIGINT                DEFAULT NULL      -- 删除人ID
);

CREATE INDEX IF NOT EXISTS idx_user_login_log_user_id ON user_login_log (user_id);
CREATE INDEX IF NOT EXISTS idx_user_login_log_login_time ON user_login_log (login_time);
CREATE INDEX IF NOT EXISTS idx_user_login_log_deleted_at ON user_login_log (deleted_at);

COMMENT ON TABLE user_login_log IS '用户登录日志表';
COMMENT ON COLUMN user_login_log.id IS '自增ID';
COMMENT ON COLUMN user_login_log.user_id IS '用户ID';
COMMENT ON COLUMN user_login_log.login_ip IS '登录IP地址';
COMMENT ON COLUMN user_login_log.user_agent IS '用户代理信息';
COMMENT ON COLUMN user_login_log.login_time IS '登录时间';
COMMENT ON COLUMN user_login_log.created_at IS '创建时间';
COMMENT ON COLUMN user_login_log.updated_at IS '更新时间（由 GORM 维护）';
COMMENT ON COLUMN user_login_log.deleted_at IS '删除时间';
COMMENT ON COLUMN user_login_log.created_by IS '创建人ID';
COMMENT ON COLUMN user_login_log.updated_by IS '更新人ID';
COMMENT ON COLUMN user_login_log.deleted_by IS '删除人ID';

-- ----------------------------------------------------------------------------
-- IAM 身份映射表（ark-iam 接入，S-A0）
--
-- ark-iam 的 rp.Identity 身份 ID 是 string（如 "user-2222"），而本站 operator.Operator
-- 与业务表审计列 created_by/updated_by/deleted_by 均为 BIGINT（uint）。本表把
-- 「外部 IAM string 身份」与「本地 BIGINT 主体」解耦：中间件解析出 IAM userID 后经
-- EnsureLocalUser 映射出 local_uid，再写入 gcontext.KeyUserID 供业务侧按 uint 读取。
--
-- local_uid 由独立序列 biz_iam_user_map_local_uid_seq 原子分配，避免并发首次登录撞号；
-- 写入走 INSERT ... ON CONFLICT (iam_user_id) DO UPDATE ... RETURNING local_uid，
-- 已存在则原样返回其既有 local_uid（幂等）。
-- ----------------------------------------------------------------------------
CREATE SEQUENCE IF NOT EXISTS biz_iam_user_map_local_uid_seq
    AS BIGINT START WITH 1 INCREMENT BY 1;         -- local_uid 分配序列（从 1 起）

CREATE TABLE IF NOT EXISTS biz_iam_user_map
(
    iam_user_id VARCHAR(128) PRIMARY KEY,                                        -- IAM 侧字符串用户ID（rp.Identity.UserID）
    local_uid   BIGINT       NOT NULL DEFAULT nextval('biz_iam_user_map_local_uid_seq'), -- 本地 BIGINT 用户ID（映射目标）
    person_id   VARCHAR(128) NOT NULL DEFAULT '',                                -- IAM 侧自然人ID（rp.Identity.PersonID）
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),                             -- 创建时间
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT now()                              -- 更新时间
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_biz_iam_user_map_local_uid ON biz_iam_user_map (local_uid);

COMMENT ON TABLE biz_iam_user_map IS 'IAM身份映射表（IAM string 用户ID ↔ 本地 BIGINT 用户ID）';
COMMENT ON COLUMN biz_iam_user_map.iam_user_id IS 'IAM 侧字符串用户ID（rp.Identity.UserID）';
COMMENT ON COLUMN biz_iam_user_map.local_uid IS '本地BIGINT用户ID（映射目标）';
COMMENT ON COLUMN biz_iam_user_map.person_id IS 'IAM 侧自然人ID（rp.Identity.PersonID）';
COMMENT ON COLUMN biz_iam_user_map.created_at IS '创建时间';
COMMENT ON COLUMN biz_iam_user_map.updated_at IS '更新时间';
