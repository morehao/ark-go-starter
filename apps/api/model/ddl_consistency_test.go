package model

import (
	"regexp"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm/schema"

	"github.com/morehao/ark-go-starter/apps/api/db"
)

// 本文件是「model ↔ DDL 逐列一致」的门禁。
//
// 背景：同一列在两处各写一遍——model 的 gorm tag 与 apps/api/db/table_init.sql。
// 两份漂移时，单元测试与 lint 都不会报（它们不读 DDL），错误要等到运行期才炸：
//   - model 多一列、DDL 没加   → 查询报 column "xxx" does not exist
//   - 两边类型/长度不一致      → 应用照跑，写入时才报 value too long for type ...
//   - DDL 有列、model 没有     → 不报错，但那一列永远没人写
//
// DDL 由 apps/api/db 包 go:embed 打进包，所以本用例不依赖测试工作目录、不连库。
//
// 比对口径（刻意的松紧边界）：
//   - 写了 type: tag 的列：类型、NOT NULL、DEFAULT、注释、主键标记全部严格比对；
//   - 没写 type: tag 的列（gorm.Model 隐含的 id/created_at/... 、time.Time 审计列）：
//     只要求「DDL 里存在」，不比对类型——gorm 的隐式映射由方言决定，硬比会把框架细节写死；
//   - DEFAULT：只在 model 声明了 default: 时严格比对；DB 侧自带默认值（now()、nextval()）
//     而 model 不声明是允许的（写入时由库赋值）；
//   - 反向：model 表对应的 DDL 不能有 model 里没有的列。

var (
	createTableRE = regexp.MustCompile(`(?i)^CREATE TABLE\s+(?:IF NOT EXISTS\s+)?([a-z_][a-z0-9_]*)`)
	commentColRE  = regexp.MustCompile(`(?i)^COMMENT ON COLUMN\s+([a-z_][a-z0-9_]*)\.([a-z_][a-z0-9_]*)\s+IS\s+'(.*)'\s*;`)
	constraintRE  = regexp.MustCompile(`(?i)^(?:CONSTRAINT|PRIMARY|UNIQUE|FOREIGN|CHECK|KEY|INDEX)\b`)
)

// tablesWithoutModel 列出 DDL 中刻意不建 Go model 的表；新增表若不建 model 必须先登记。
var tablesWithoutModel = map[string]string{
	"user_login_log": "当前无 Go model 与代码引用，保留 DDL 待用",
}

// ddlColumn 是 DDL 里一列的可比对属性。
type ddlColumn struct {
	typ        string // 归一化类型，如 varchar(255)、numeric(10,2)
	notNull    bool
	primaryKey bool
	hasDefault bool
	def        string // 归一化默认值
	comment    string
}

// ddlTable 是一张表在 DDL 里的形态。
type ddlTable struct {
	columns  map[string]*ddlColumn
	comments map[string]string // 列名 → COMMENT ON COLUMN 文本
}

// TestModelsMatchDDL 逐列比对 postgresModels 与建表 DDL。
func TestModelsMatchDDL(t *testing.T) {
	t.Parallel()

	tables := parseDDL(db.TableInitDDL)

	for _, dest := range postgresModels {
		entity, err := schema.Parse(dest, &sync.Map{}, schema.NamingStrategy{})
		require.NoError(t, err)

		tableName := entity.Table
		table, ok := tables[tableName]
		require.Truef(t, ok, "DDL（apps/api/db/table_init.sql）里没有表 %q", tableName)
		if !ok {
			continue
		}

		modelColumns := make(map[string]struct{}, len(entity.Fields))
		for _, field := range entity.Fields {
			modelColumns[field.DBName] = struct{}{}
			assertModelColumn(t, tableName, table, field)
		}

		for _, name := range sortedKeys(table.columns) {
			if _, ok := modelColumns[name]; !ok {
				t.Errorf("%s.%s 在 DDL 里有、model 里没有：这一列永远没人写，确认是漏了字段还是 DDL 该删",
					tableName, name)
			}
		}
	}
}

// TestDDLTablesAreModeled 要求 DDL 里每张表要么有 model 覆盖，要么在 tablesWithoutModel 登记。
func TestDDLTablesAreModeled(t *testing.T) {
	t.Parallel()

	tables := parseDDL(db.TableInitDDL)

	modeled := make(map[string]struct{}, len(postgresModels))
	for _, dest := range postgresModels {
		entity, err := schema.Parse(dest, &sync.Map{}, schema.NamingStrategy{})
		require.NoError(t, err)
		modeled[entity.Table] = struct{}{}
	}

	for _, name := range sortedKeys(tables) {
		_, hasModel := modeled[name]
		reason, acknowledged := tablesWithoutModel[name]
		if hasModel {
			assert.Falsef(t, acknowledged, "表 %q 已有 model，应从 tablesWithoutModel 白名单移除（登记原因：%s）", name, reason)
			continue
		}
		assert.Truef(t, acknowledged, "DDL 里的表 %q 没有 Go model：补 model，或在 tablesWithoutModel 写明原因", name)
	}
}

// TestDDLColumnCommentsInSync 要求每列的「行尾 -- 注释」与「COMMENT ON COLUMN」文本一致，
// 这是 backend-conventions「DDL 字段注释写两份、保持一致」的机械校验。
func TestDDLColumnCommentsInSync(t *testing.T) {
	t.Parallel()

	tables := parseDDL(db.TableInitDDL)

	for _, tableName := range sortedKeys(tables) {
		table := tables[tableName]
		for _, columnName := range sortedKeys(table.columns) {
			column := table.columns[columnName]
			metaComment, ok := table.comments[columnName]
			assert.Truef(t, ok, "%s.%s 缺少 COMMENT ON COLUMN", tableName, columnName)
			assert.NotEmptyf(t, column.comment, "%s.%s 缺少行尾 -- 注释", tableName, columnName)
			assert.Equalf(t, column.comment, metaComment,
				"%s.%s 的行尾注释与 COMMENT ON COLUMN 不一致：改一处要同步另一处", tableName, columnName)
		}
	}
}

// assertModelColumn 比对单个 model 字段与对应 DDL 列。
func assertModelColumn(t *testing.T, tableName string, table *ddlTable, field *schema.Field) {
	t.Helper()

	column, ok := table.columns[field.DBName]
	if !assert.Truef(t, ok, "%s.%s 在 model 里有、DDL 里没有：建表脚本缺这一列", tableName, field.DBName) {
		return
	}

	assert.Equalf(t, field.PrimaryKey, column.primaryKey,
		"%s.%s 的主键标记不一致：model=%v DDL=%v", tableName, field.DBName, field.PrimaryKey, column.primaryKey)

	if comment := field.TagSettings["COMMENT"]; comment != "" {
		assert.Equalf(t, comment, table.comments[field.DBName],
			"%s.%s 的 comment: tag 与 COMMENT ON COLUMN 不一致", tableName, field.DBName)
	}

	typ := modelColumnType(field)
	if typ == "" {
		return // 未显式声明类型的列只校验存在性，见文件头「比对口径」
	}

	assert.Equalf(t, typ, column.typ,
		"%s.%s 类型不一致：model=%s DDL=%s", tableName, field.DBName, typ, column.typ)

	wantNotNull := field.NotNull || field.PrimaryKey
	gotNotNull := column.notNull || column.primaryKey
	assert.Equalf(t, wantNotNull, gotNotNull,
		"%s.%s 的 NOT NULL 不一致：model=%v DDL=%v", tableName, field.DBName, wantNotNull, gotNotNull)

	if def, ok := field.TagSettings["DEFAULT"]; ok {
		assert.Truef(t, column.hasDefault, "%s.%s 在 model 声明了 default:%s，DDL 却没有 DEFAULT", tableName, field.DBName, def)
		assert.Equalf(t, normalizeDefault(def), column.def,
			"%s.%s 默认值不一致：model=%s DDL=%s", tableName, field.DBName, normalizeDefault(def), column.def)
	}
}

// modelColumnType 把 gorm tag 的 type/precision/scale 归一化成与 DDL 可比的形式，
// 如 type:numeric;precision:10;scale:2 → numeric(10,2)。未声明 type: 时返回空串。
func modelColumnType(field *schema.Field) string {
	raw := strings.TrimSpace(field.TagSettings["TYPE"])
	if raw == "" {
		return ""
	}
	typ := normalizeType(raw)
	precision := strings.TrimSpace(field.TagSettings["PRECISION"])
	scale := strings.TrimSpace(field.TagSettings["SCALE"])
	switch {
	case precision != "" && scale != "":
		typ += "(" + precision + "," + scale + ")"
	case precision != "":
		typ += "(" + precision + ")"
	}
	return typ
}

// parseDDL 解析建表脚本，返回表名 → 表定义。只认 CREATE TABLE 与 COMMENT ON COLUMN，
// 忽略注释行、CREATE INDEX / CREATE SEQUENCE 等语句。
func parseDDL(ddl string) map[string]*ddlTable {
	tables := make(map[string]*ddlTable)
	var current *ddlTable

	for raw := range strings.SplitSeq(ddl, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "--") {
			continue
		}

		if match := createTableRE.FindStringSubmatch(line); match != nil {
			current = &ddlTable{columns: make(map[string]*ddlColumn), comments: make(map[string]string)}
			tables[match[1]] = current
			continue
		}

		if current != nil {
			if strings.HasPrefix(line, ")") {
				current = nil
				continue
			}
			if strings.HasPrefix(line, "(") { // 建表语句左括号独占一行
				continue
			}
			if constraintRE.MatchString(line) {
				continue
			}
			name, column := parseColumn(line)
			current.columns[name] = column
			continue
		}

		if match := commentColRE.FindStringSubmatch(line); match != nil {
			if table, ok := tables[match[1]]; ok {
				table.comments[match[2]] = match[3]
			}
		}
	}
	return tables
}

// parseColumn 解析一行列定义，返回列名与归一化后的属性。
func parseColumn(line string) (string, *ddlColumn) {
	definition, comment := line, ""
	if before, after, found := strings.Cut(line, "--"); found {
		definition = strings.TrimSpace(before)
		comment = strings.TrimSpace(after)
	}

	definition = strings.TrimRight(strings.TrimSpace(definition), ",;")
	column := &ddlColumn{comment: comment}
	lower := strings.ToLower(definition)
	column.notNull = strings.Contains(lower, "not null")
	column.primaryKey = strings.Contains(lower, "primary key")

	// DEFAULT 之后的内容整体当作默认值表达式（本仓库 DDL 约定 DEFAULT 写在最后）。
	if idx := strings.Index(lower, " default "); idx >= 0 {
		column.hasDefault = true
		column.def = normalizeDefault(definition[idx+len(" default "):])
		definition = definition[:idx]
	}

	nameEnd := strings.IndexAny(definition, " \t")
	if nameEnd <= 0 {
		return strings.ToLower(strings.TrimSpace(definition)), column
	}
	name := strings.ToLower(definition[:nameEnd])

	typ := definition[nameEnd:]
	typ = strings.ReplaceAll(strings.ToLower(typ), "not null", " ")
	typ = strings.ReplaceAll(typ, "primary key", " ")
	column.typ = normalizeType(typ)

	return name, column
}

// normalizeType 统一大小写与空白：VARCHAR(255) / varchar( 255 ) 都归一成 varchar(255)。
func normalizeType(raw string) string {
	typ := strings.ToLower(strings.TrimSpace(raw))
	typ = strings.TrimSuffix(typ, ";")
	typ = strings.ReplaceAll(typ, " ", "")
	return typ
}

// normalizeDefault 归一化默认值：去引号、去 ::type 转换、统一小写，
// 让 model 的 default:” 与 DDL 的 DEFAULT ”、default:0 与 DEFAULT 0 可比。
func normalizeDefault(raw string) string {
	value := strings.ToLower(strings.TrimSpace(raw))
	value = strings.TrimSuffix(value, ";")
	if idx := strings.Index(value, "::"); idx >= 0 {
		value = strings.TrimSpace(value[:idx])
	}
	if len(value) >= 2 && strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'") {
		value = value[1 : len(value)-1]
	}
	if value == "null" {
		return ""
	}
	return value
}

// sortedKeys 返回 map 的有序 key，让失败信息稳定可读。
func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
