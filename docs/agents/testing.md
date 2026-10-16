# 测试规范

> 上级入口：[AGENTS.md](../../AGENTS.md)　·　文档中心：[docs/agents/README.md](./README.md)

## 约定

- 测试文件放在同包或 `_test` 包中
- 使用 `pkg/testsetup` 初始化测试环境：`TestMain` 中调 `testsetup.Initialize(testsetup.AppNameDemo)` / `testsetup.Close(testsetup.AppNameDemo)`；需要带身份的 ctx 用 `testsetup.NewContext(...)`。详见 [pkg/testsetup/README.md](../../backend/pkg/testsetup/README.md)
- 使用标准库 `testing` 包和 `testify/assert`

```go
package svcuser

import (
    "testing"

    "github.com/morehao/golib/gcrypto"
)

func TestGeneratePassword(t *testing.T) {
    hash, err := gcrypto.GeneratePasswordHash("password")
    if err != nil {
        t.Fatalf("GeneratePasswordHash failed: %v", err)
    }
    if err := gcrypto.ComparePasswordHash(hash, "password"); err != nil {
        t.Errorf("ComparePasswordHash failed: %v", err)
    }
}
```

## 运行方式

```bash
# 运行指定应用的测试（推荐，项目根目录执行）
make test APP=demo

# 运行单个测试函数（在 backend/ 目录下执行）
go test ./apps/demo/internal/service/svcuser -run TestGeneratePassword -v
```

## 环境依赖

`pkg/dbclient/dbcheck_test.go` 依赖本地 PostgreSQL(5432) / Redis(6379) / ES(9200) 服务；未启动对应服务时该包测试会失败，属环境依赖问题而非代码问题。数据库初始化脚本见 `backend/scripts/sql/table_init.sql`（PostgreSQL DDL）。

## 相关文档

- 命令大全：[commands.md](./commands.md)
- 代码规范：[backend-conventions.md](./backend-conventions.md)
