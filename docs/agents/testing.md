# 测试规范

> 上级入口：[AGENTS.md](../../AGENTS.md)　·　文档中心：[docs/agents/README.md](./README.md)

## 约定

- 测试文件放在同包或 `_test` 包中，**文件名与被测源文件同名**（`user.go` → `user_test.go`），不要按函数名命名
- 使用 `pkg/testsetup` 初始化测试环境：`testsetup.Initialize(testsetup.AppNameDemo)` / `testsetup.Close(testsetup.AppNameDemo)`；需要带身份的 ctx 用 `testsetup.NewContext(...)`。详见 [pkg/testsetup/README.md](../../pkg/testsetup/README.md)
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

## 端到端测试（e2e，两端覆盖）

项目提供前后端 e2e，作为集成测试之上的最外层保障。**e2e 与集成测试的 tag 不同**：

| 层 | 位置 | build tag / 命令 | 说明 |
| --- | --- | --- | --- |
| 后端集成级 e2e | `apps/api/internal/controller/ctrproduct/product_e2e_test.go` | `//go:build integration`，`make test-integration` | 进程内真栈（httptest）+ 真库，快、定位准 |
| 后端真端口 e2e | `apps/api/e2e/` | `//go:build e2e`，`make e2e-backend` | 真实 TCP 端口 + 真库，验证二进制可起、端口可通 |
| 前端 Playwright e2e | `tests/e2e/` | `make e2e-frontend` | 真浏览器 + 前端服务 + 真后端，覆盖核心用户路径 |

**硬性规则**：
- 后端 e2e 用例必须加 `//go:build e2e`，`go test ./...` 与 `make test` 默认不执行，不污染单元信号。
- 前端 e2e 的数据统一加 `e2e-` 前缀 + 时间戳保证唯一，并在 `tests/e2e/support/teardown.ts` 兜底清理，避免脏数据留在真实库。
- 前端 Playwright 依赖后端运行在 `http://localhost:8099`（vite `/v1` 代理转发）；由 Makefile / CI 负责拉起后端。

```bash
# 首次：安装前端依赖与 Playwright 浏览器
make e2e-install

# 本地一键两端 e2e（需本地 PG/Redis 就绪，后端 8099 会由 e2e 自行/由你启动）
make e2e

# 分别运行
make e2e-backend  APP=demo   # 后端真端口 e2e（用例自起随机端口）
make e2e-frontend            # 前端 Playwright（需先启动后端 8099，Playwright 自动拉起前端）

# 依赖预检（探测本地 PG/Redis 端口）
make e2e-deps-check
```

> 复用本地 PG/Redis 的策略：e2e 沿用 `apps/api/config/config.yaml` 指向的本地实例，
> 未就绪时 `make e2e-deps-check` 会明确报错，不自动起库。

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

# 在仓库根按模块单独跑（apps/api 与 pkg 各自为独立 module）
cd pkg && go test -race ./...
cd pkg && go test -race -tags=integration ./...

# 运行单个测试函数
go test ./apps/api/internal/service/svcproduct -run TestGeneratePassword -v
```

## 环境依赖

- 默认 `go test ./...` / `make test` **不依赖任何外部服务**；连库用例已被 `//go:build integration` 排除。
- `make test-integration` 需要本地 PostgreSQL(5432) / Redis(6379)，且库表已按 `tools/sql/table_init.sql` 初始化；未启动时失败属环境问题而非代码问题。
- 本项目不连接 Elasticsearch。

## 相关文档

- 命令大全：[commands.md](./commands.md)
- 代码规范：[backend-conventions.md](./backend-conventions.md)
