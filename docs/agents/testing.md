# 测试规范

> 上级入口：[AGENTS.md](../../AGENTS.md)　·　文档中心：[docs/agents/README.md](./README.md)

## 约定

- 测试文件放在同包或 `_test` 包中，**文件名与被测源文件同名**（`user.go` → `user_test.go`），不要按函数名命名
- 使用 `pkg/testsetup` 初始化测试环境：`testsetup.Initialize(testsetup.AppNameDemo)` / `testsetup.Close(testsetup.AppNameDemo)`；需要带身份的 ctx 用 `testsetup.NewContext(...)`。详见 [pkg/testsetup/README.md](../../backend/pkg/testsetup/README.md)
- 使用标准库 `testing` 包和 `testify`（`assert` / `require`）

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

## 集成测试必须隔离（硬性要求）

**凡依赖 PostgreSQL / Redis / 外部 HTTP 服务的测试，文件首行必须加 `//go:build integration`。**

这是硬性要求，原因：`go test ./...` 在无 DB 环境会把整个仓库测红，掩盖真实单元测试信号，也让 CI 与本地表现不一致。

```go
//go:build integration

// 依赖本地 PostgreSQL/Redis，属集成测试：默认 `go test ./...` 不运行，
// 需 `go test -tags=integration ./...`。

package dbclient_test

import "testing"

func TestDbcheckVerify(t *testing.T) {
    testsetup.Initialize(testsetup.AppNameDemo)
    // ...
}
```

判断标准：只要调用了 `testsetup.Initialize`（会连库）或发起真实网络请求，就要加 tag。

## 断言约定

| 场景 | 写法 | 原因 |
| --- | --- | --- |
| 错误断言 | `assert.NoError(t, err)` / `require.NoError(t, err)` | **不要** `assert.Nil(t, err)`（`testifylint` 会报 `error-nil`） |
| 前置条件（失败即停） | `require.*` | 后续断言依赖该结果，`assert` 会继续执行导致空指针 |
| 一般断言 | `assert.*` | 一次跑到尾，暴露更多问题 |

## 并行

- 不依赖全局可变状态、不启动服务的**纯单元测试**应加 `t.Parallel()`；
- 依赖 `testsetup.Initialize` 的集成测试**不要**加（共享全局初始化的单例）。

## 运行方式

```bash
# 单元测试（pkg + 指定应用，无需 DB/Redis）
make test APP=demo

# 集成测试（需本地 PostgreSQL(5432)/Redis(6379)）
make test-integration APP=demo

# 在 backend/ 下按模块单独跑
cd backend/pkg && go test -race ./...
cd backend/pkg && go test -race -tags=integration ./...

# 运行单个测试函数
go test ./apps/demo/internal/service/svcuser -run TestGeneratePassword -v
```

## 环境依赖

- 默认 `go test ./...` / `make test` **不依赖任何外部服务**；连库用例已被 `//go:build integration` 排除。
- `make test-integration` 需要本地 PostgreSQL(5432) / Redis(6379)，且库表已按 `backend/scripts/sql/table_init.sql` 初始化；未启动时失败属环境问题而非代码问题。
- 本项目不连接 Elasticsearch。

## 相关文档

- 命令大全：[commands.md](./commands.md)
- 代码规范：[backend-conventions.md](./backend-conventions.md)
