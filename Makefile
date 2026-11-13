# Go应用程序Makefile

# ============================================================
# 构建相关变量
# ============================================================
APP        =
# 应用逻辑标识(APP) 与 目录名(AppDir) 解耦（D11）：demo 的 Go 目录已迁为 apps/api。
# 新增应用时：若目录名与逻辑名不同，在此补一条映射即可。
APP_DIR    = $(if $(filter demo,$(APP)),api,$(APP))
BINARY     = $(APP)
MAIN_DIR   = ./apps/$(APP_DIR)/cmd
BUILD_DIR  = ./output/build
VERSION    = $(shell date +%Y%m%d%H%M%S)-$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

APP_CONFIG_PATH = /app/config.yaml

# go命令的环境变量
GO_ENV = CGO_ENABLED=0 GOPROXY=https://goproxy.cn,direct

# Dockerfile 位置（构建上下文为仓库根）
DOCKERFILE = ./deploy/docker/api.Dockerfile

# 注意：本项目所有目标都**不执行** `go work sync`。
# 原因（go 1.27.3 实测）：go.work 只 use 了 ./apps/api 与 ./pkg（go.work 在仓库根），`go work sync` 会把
# demo/go.mod 里显式 require 的 ark-iam 模块（如 github.com/morehao/ark-iam/pkg）当作
# 冗余依赖从 go.mod 删掉（且未补进 go.work.sum），随后 go build / go test 直接报
# "imports ... from implicitly required module"，干净检出下 `make build` / `make test` 必挂。
# 依赖刷新请改用各模块内的 `go mod tidy` / `go mod download`（见 deps、tidy 目标）。
# 相关排查记录：.dsh/docs/notes/2026-11-10-go-work-local-modules.md

# ============================================================
# Docker 相关变量
# ============================================================
# 获取 git tag（如果存在），不存在时为空字符串
GIT_TAG    = $(shell git describe --tags --exact-match 2>/dev/null || echo "")
# 获取 commit hash 短格式
GIT_COMMIT = $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
# 构建镜像 tag：如果有 git tag 则使用 tag-commit，否则使用 commit
DOCKER_TAG        = $(if $(GIT_TAG),$(GIT_TAG)-$(GIT_COMMIT),$(GIT_COMMIT))
# 完整的镜像名称：appname:tag
DOCKER_IMAGE      = $(APP):$(DOCKER_TAG)
# 容器对外暴露端口（可通过命令行覆盖，例如：make docker-run APP=foo PORT=9090）
PORT              = 8099

# ============================================================
# 伪目标
# ============================================================
.PHONY: all build build-env clean run lint lint-frontend test test-integration swag codegen \
        e2e e2e-backend e2e-frontend e2e-install e2e-deps-check contracts-verify \
        docker-build docker-run check-image \
        list-apps deps tidy update-dep \
        dev-all dev-frontend stop-frontend stop-dev \
        quickstart quickstart-local quickstart-verify help

# ============================================================
# 通用入口：清理、依赖、构建并运行
# ============================================================
all: clean deps build run

# ============================================================
# 工具函数
# ============================================================

# 验证 APP 参数是否有效，同时列出可用应用供参考
define validate_app
	@if [ -z "$(APP)" ]; then \
		echo "❌ 请使用 APP=<名称> 指定要操作的应用程序，例如：make build APP=demo"; \
		echo "   可用的应用程序：$$(ls ./apps 2>/dev/null | tr '\n' ' ')"; \
		exit 1; \
	fi
	@if [ ! -d "./apps/$(APP_DIR)" ]; then \
		echo "❌ 应用程序 '$(APP)'（目录 apps/$(APP_DIR)）不存在"; \
		echo "   可用的应用程序：$$(ls ./apps 2>/dev/null | tr '\n' ' ')"; \
		exit 1; \
	fi
endef

# ============================================================
# 构建
# ============================================================

# 构建应用程序（本机环境）
build:
	$(call validate_app)
	@echo "🔨 正在构建应用程序 $(APP)..."
	@mkdir -p $(BUILD_DIR)
	@go build -ldflags="-X 'main.BuildVersion=$(VERSION)'" -o output/build/$(BINARY) $(MAIN_DIR)
	@echo "✅ 构建完成: ./output/build/$(BINARY)"

# 为指定环境构建（默认注入 CGO_ENABLED=0，适合容器/Linux 交叉编译）
build-env:
	$(call validate_app)
	@echo "🔨 正在为 [$(GO_ENV)] 构建 $(APP)..."
	@mkdir -p $(BUILD_DIR)
	@$(GO_ENV) go build -ldflags="-X 'main.BuildVersion=$(VERSION)'" -o output/build/$(BINARY) $(MAIN_DIR)
	@echo "✅ 构建完成: ./output/build/$(BINARY)"

# ============================================================
# 清理
# ============================================================
clean:
	@echo "🧹 正在清理构建目录..."
	@rm -rf $(BUILD_DIR)
	@echo "✅ 已清理构建目录"

# ============================================================
# 运行 & 测试
# ============================================================

# 直接用 go run 启动（开发调试用）
# 注意：必须在 apps/<appdir>/cmd 目录下启动——config（../config/config.yaml）、
# 日志、存储等相对路径（缺省 .local/）均按该 CWD 与配置设计。
run:
	$(call validate_app)
	@echo "🚀 正在运行应用程序 $(APP)..."
	@cd apps/$(APP_DIR)/cmd && go run .

# 运行单元测试（pkg 公共层 + 指定应用；不含 //go:build integration 的用例）
test:
	$(call validate_app)
	@echo "🧪 正在运行测试..."
	@go test ./pkg/... ./apps/$(APP_DIR)/... -v

# 运行集成测试（需本地 PostgreSQL(5432)/Redis(6379) 就绪）
test-integration:
	$(call validate_app)
	@echo "🔗 正在运行集成测试..."
	@go test -tags=integration ./pkg/... ./apps/$(APP_DIR)/... -v

# ============================================================
# E2E 端到端测试
# ============================================================

# 依赖预检：探测本地 PostgreSQL(5432)/Redis(6379) 是否就绪（e2e 复用本地实例）
e2e-deps-check:
	@ok=1; \
	if ! nc -z 127.0.0.1 5432 2>/dev/null; then echo "❌ PostgreSQL(127.0.0.1:5432) 未就绪，请先启动本地 PG 并执行 apps/api/db/table_init.sql"; ok=0; fi; \
	if ! nc -z 127.0.0.1 6379 2>/dev/null; then echo "❌ Redis(127.0.0.1:6379) 未就绪，请先启动本地 Redis"; ok=0; fi; \
	[ $$ok -eq 1 ] && echo "✅ 本地 PG/Redis 就绪" || exit 1

# 后端真端口端到端测试（需 PG/Redis；--tags=e2e 与单元/集成测试隔离）
e2e-backend:
	$(call validate_app)
	@echo "🌐 正在运行后端 e2e（真端口）..."
	@go test -tags=e2e ./apps/$(APP_DIR)/e2e/... -v -count=1

# 前端 Playwright 端到端测试（需后端 8099 已在运行；前端服务由 playwright webServer 自动拉起）
e2e-frontend:
	@echo "🎭 正在运行前端 Playwright e2e..."
	@cd tests/e2e && pnpm exec playwright test

# 两端 e2e 全量
e2e: e2e-deps-check e2e-backend e2e-frontend

# 首次安装 Playwright 浏览器依赖
e2e-install:
	@echo "📦 正在安装前端依赖与 Playwright 浏览器..."
	@pnpm install
	@cd tests/e2e && pnpm exec playwright install --with-deps chromium

# ============================================================
# 依赖管理
# ============================================================

# 同步依赖：逐模块 go mod download + tidy（不使用 go work sync，原因见文件头注释）
deps:
	@echo "📦 正在下载依赖项..."
	@find ./apps ./pkg -name "go.mod" -not -path "*/vendor/*" | while read modfile; do \
		dir=$$(dirname $$modfile); \
		echo "  => $$dir"; \
		(cd $$dir && $(GO_ENV) go mod download && go mod tidy); \
	done
	@echo "✅ 依赖项已更新"

# 仅执行 go mod tidy（不下载全量依赖，速度更快）
tidy:
	@find ./apps ./pkg -name "go.mod" -not -path "*/vendor/*" | while read modfile; do \
		dir=$$(dirname $$modfile); \
		echo "==> Tidy in $$dir"; \
		(cd $$dir && go mod tidy); \
	done

# 升级指定依赖包（仅在引用该包的模块中执行）
# 用法：make update-dep ARGS="-u github.com/morehao/golib@v0.1.1"
update-dep:
	@if [ -z "$(ARGS)" ]; then \
		echo "❌ 请提供 ARGS 参数"; \
		echo "   用法：make update-dep ARGS=\"-u github.com/morehao/golib@v0.1.1\""; \
		exit 1; \
	fi
	@find ./apps ./pkg -name "go.mod" -not -path "*/vendor/*" | while read modfile; do \
		dir=$$(dirname $$modfile); \
		pkg=$$(echo "$(ARGS)" | grep -oE '[a-zA-Z0-9._/-]+/[a-zA-Z0-9._/-]+' | head -1); \
		if grep -q "$$pkg" "$$modfile"; then \
			echo "==> Updating in $$dir"; \
			(cd $$dir && go get $(ARGS) && go mod tidy); \
		fi \
	done
	@$(MAKE) tidy

# ============================================================
# 代码生成
# ============================================================

# 生成 Swagger 文档
swag:
	$(call validate_app)
	@echo "📚 正在生成 Swagger 文档..."
	@which swag > /dev/null 2>&1 || (echo "⚠️  swag 未安装，正在安装..." && go install github.com/swaggo/swag/cmd/swag@latest)
	@swag init \
		--parseDependency \
		--parseInternal \
		--parseDependencyLevel 2 \
		-g app.go \
		--dir apps/$(APP_DIR) \
		--output apps/$(APP_DIR)/docs \
		--outputTypes go \
		--instanceName $(APP)
	@echo "✅ Swagger 文档已生成：apps/$(APP_DIR)/docs"

# gocli 版本：模板已切换为 RESTful kebab-case 复数资源路由风格
CLI_VERSION := v1.32.6
CLI_PKG     := github.com/morehao/gocli

# 代码生成（API / module / model 等）
# 用法：make codegen APP=demoapp COMMAND=api
codegen:
	$(call validate_app)
	@if [ -z "$(COMMAND)" ]; then \
		echo "❌ 请使用 COMMAND 参数指定生成命令，例如：make codegen APP=$(APP) COMMAND=api"; \
		echo "   支持的命令：api, module, model"; \
		exit 1; \
		fi
	@if ! command -v gocli >/dev/null 2>&1; then \
		echo "⚠️  未检测到 gocli，正在安装 $(CLI_VERSION)..."; \
		go install $(CLI_PKG)@$(CLI_VERSION); \
	else \
		INSTALLED_VER=$$(go version -m $$(command -v gocli) 2>/dev/null | awk -v pkg="$(CLI_PKG)" '$$0 ~ "^[ \t]+mod[ \t]+" pkg "[ \t]" {print $$3; exit}'); \
		echo "🔍 已安装的 gocli 版本: [$$INSTALLED_VER] 目标版本: $(CLI_VERSION)"; \
		if [ -n "$$INSTALLED_VER" ] && [ "$$INSTALLED_VER" = "$(CLI_VERSION)" ]; then \
			echo "✅ gocli 版本已是最新"; \
		else \
			echo "⚠️  版本不匹配，重新安装 $(CLI_VERSION)..."; \
			go install $(CLI_PKG)@$(CLI_VERSION); \
		fi; \
	fi
	@echo "🔧 开始生成代码：APP=$(APP)，COMMAND=$(COMMAND)"
	@gocli generate $(COMMAND) -a $(APP)

# ============================================================
# Docker
# ============================================================

# 构建 Docker 镜像（构建上下文为仓库根：.dockerignore 负责排除 node_modules/log/output/storage 等）
docker-build:
	$(call validate_app)
	@echo "🐳 正在构建 $(APP) 的 Docker 镜像..."
	@echo "   镜像名称  : $(DOCKER_IMAGE)"
	@echo "   Git Tag   : $(if $(GIT_TAG),$(GIT_TAG),无)"
	@echo "   Git Commit: $(GIT_COMMIT)"
	@docker buildx build -t $(DOCKER_IMAGE) -f $(DOCKERFILE) .
	@echo "✅ Docker 镜像 $(DOCKER_IMAGE) 构建完成"

# 运行 Docker 容器（镜像不存在时自动构建）
# 用法：make docker-run APP=demoapp [PORT=9090]
docker-run: check-image
	@echo "🚀 正在运行 $(APP) 容器..."
	@echo "   使用镜像 : $(DOCKER_IMAGE)"
	@echo "   端口映射 : $(PORT):$(PORT)"
	-@docker rm -f $(APP) 2>/dev/null || true
	@docker run -d \
		--name $(APP) \
		-e APP_CONFIG_PATH=$(APP_CONFIG_PATH) \
		-p $(PORT):$(PORT) \
		$(DOCKER_IMAGE)
	@echo "✅ 容器 $(APP) 已启动，服务地址：http://localhost:$(PORT)"

# 检查镜像是否存在，不存在才构建（保留缓存）
check-image:
	$(call validate_app)
	@if [ -z "$$(docker images -q $(DOCKER_IMAGE) 2>/dev/null)" ]; then \
		echo "📦 镜像 $(DOCKER_IMAGE) 不存在，开始构建..."; \
		$(MAKE) docker-build APP=$(APP); \
	else \
		echo "✅ 镜像 $(DOCKER_IMAGE) 已存在，跳过构建"; \
	fi

# ============================================================
# 一键跑通（clone 即用）
# ============================================================

# 一键跑通（有 Docker）：自动起 PG/Redis → 建库建表 → IAM 初始化 → 起前后端。
# 幂等，可重复执行不报错；端口被占自动降级复用外部依赖。
# 用法：make quickstart [APP=demo] [IAM_ADMIN_USER=admin IAM_ADMIN_PASS=***]
quickstart:
	@bash scripts/quickstart.sh

# 一键跑通（无 Docker / 复用本机已就绪的 PG 5432 与 Redis 6379）
quickstart-local:
	@SKIP_DEPS=1 bash scripts/quickstart.sh

# 一键跑通验收：对已在运行的环境跑 IAM 登录全流程 e2e。
# quickstart 以 op 模式启动（登录强制），故这里跑的是「未登录→跳登录页→账密登录→回跳
# 业务页→带 token 调受保护接口」这条主链路（specs/iam-login.spec.ts），它同时验证登录门后
# 的商品管理页与受保护接口可用。其余业务页（商品/字典/文件/异步/定时）的细粒度回归在
# `make e2e`（iam.mode=disabled 的 OFF 全量）中覆盖。
# 前置：先 make quickstart 把后端 8099 / 前端 3000 起好且 IAM 已初始化。
# 用法：make quickstart-verify [IAM_ADMIN_USER=admin IAM_ADMIN_PASS=***]
quickstart-verify:
	@echo "🎭 运行 quickstart 验收 e2e（E2E_IAM=1 · iam-login 主链路）..."
	@cd tests/e2e && \
		E2E_IAM=1 \
		E2E_IAM_ADMIN_USER="$${IAM_ADMIN_USER:-admin}" \
		E2E_IAM_ADMIN_PASS="$${IAM_ADMIN_PASS:-Adm1n@Strong!Pass}" \
		pnpm exec playwright test specs/iam-login.spec.ts

# ============================================================
# 前端开发
# ============================================================

# 停止本地开发进程：后端（go run / 编译产物 exe/cmd）、前端 vite/pnpm，以及占用
# 8099（后端）与 3000-3010（前端可能漂移的端口段）的残留进程。幂等，无进程时安全退出。
# 供 dev-all 每次启动前自动清理，避免重复启动导致 8099 端口冲突 panic 或前端端口漂移。
stop-dev:
	@echo "🧹 启动前清理既有开发进程..."; \
	pkill -KILL -f "go run \." 2>/dev/null; \
	pkill -KILL -f "exe/cmd" 2>/dev/null; \
	pkill -KILL -f "make dev-all" 2>/dev/null; \
	pkill -KILL -f "vite" 2>/dev/null; \
	pkill -KILL -f "pnpm --filter @ark-go-starter/demo-web dev" 2>/dev/null; \
	for p in 8099 3000 3001 3002 3003 3004 3005 3006 3007 3008 3009 3010; do \
		pids="$$(lsof -tiTCP:$$p -sTCP:LISTEN 2>/dev/null)"; \
		if [ -n "$$pids" ]; then kill -KILL $$pids 2>/dev/null; fi; \
	done; \
	sleep 1; \
	echo "✅ 清理完成"

# 同时启动后端与前端开发服务（后端后台运行、verbose 日志写入 .local/log/dev-all-backend.log；
# 前端前台输出；探测到前端端口后再等待后端 :8099 健康检查通过，随后打印明确的访问地址，
# 避免「前端已就绪、后端还在编译/迁移」的窗口期访问导致 /iam 代理 ECONNREFUSED；Ctrl+C 一并停止）
# 用法：make dev-all [APP=demo]
# 注意：启动前会先调用 stop-dev 杀掉既有的后端/前端进程，保证端口干净再启动。
dev-all: stop-dev
	@app="$(APP)"; appdir="$(APP_DIR)"; app=$${app:-demo}; appdir=$${appdir:-demo}; \
	if [ ! -d "./apps/$$appdir" ]; then \
		echo "❌ 应用程序 '$$app'（目录 apps/$$appdir）不存在"; \
		echo "   可用的应用程序：$$(ls ./apps 2>/dev/null | tr '\n' ' ')"; \
		exit 1; \
	fi; \
	mkdir -p ./.local/log; \
	echo "🚀 正在启动后端 [$$app] 与前端（后端日志：.local/log/dev-all-backend.log）..."; \
	before_ports="$$(lsof -nP -iTCP -sTCP:LISTEN 2>/dev/null | awk 'NR>1 {print $$9}' | grep -oE ':[0-9]+$$' | tr -d ':' | sort -u | tr '\n' ' ')"; \
	( cd apps/$$appdir/cmd && go run . ) > ./.local/log/dev-all-backend.log 2>&1 & \
	backend_pid=$$!; \
	trap 'echo ""; echo "🛑 正在停止后端 [$$app]..."; kill $$backend_pid 2>/dev/null; wait $$backend_pid 2>/dev/null; echo "✅ 已停止"; exit 0' INT TERM; \
	( fe_port=""; \
	for _ in $$(seq 1 120); do \
		for p in 3000 3001 3002 3003 3004 3005 3006 3007 3008 3009 3010; do \
			if echo " $$before_ports " | grep -q " $$p "; then \
				continue; \
			fi; \
			if nc -z -w 1 127.0.0.1 $$p >/dev/null 2>&1 || nc -z -w 1 ::1 $$p >/dev/null 2>&1; then \
				fe_port=$$p; \
				break; \
			fi; \
		done; \
		[ -n "$$fe_port" ] && break; \
		sleep 1; \
	done; \
	if [ -z "$$fe_port" ]; then \
		echo ""; \
		echo "⚠️  未探测到前端端口（3000-3010），请确认前端是否启动成功。"; \
		exit 0; \
	fi; \
	echo ""; \
	echo "⏳ 前端已就绪(:$$fe_port)，等待后端 :8099 就绪（首次 go run 需编译+迁移，可能较久）..."; \
	backend_ready=0; \
	for _ in $$(seq 1 180); do \
		if curl -s -o /dev/null -w '%{http_code}' http://127.0.0.1:8099/v1/$$app/health 2>/dev/null | grep -q '^[2-5][0-9][0-9]$$'; then \
			backend_ready=1; break; \
		fi; \
		if nc -z -w 1 127.0.0.1 8099 >/dev/null 2>&1; then \
			backend_ready=1; break; \
		fi; \
		sleep 1; \
	done; \
	echo ""; \
	echo "=============================================="; \
	echo "  🌐 前端访问地址: http://localhost:$$fe_port"; \
	echo "  🔌 后端接口:     http://localhost:8099"; \
	echo "  📄 后端日志:     .local/log/dev-all-backend.log"; \
	if [ "$$backend_ready" = "1" ]; then \
		echo "  ✅ 前后端均已就绪，可以访问"; \
	else \
		echo "  ⚠️  后端 :8099 仍未就绪（详见上方日志），此时访问前端可能报 /iam 代理错误"; \
		echo "     可在后端起来后刷新浏览器，日志：.local/log/dev-all-backend.log"; \
	fi; \
	echo "=============================================="; \
	echo ""; \
	) & \
	watcher_pid=$$!; \
	( pnpm dev ); \
	kill $$watcher_pid 2>/dev/null; \
	kill $$backend_pid 2>/dev/null; wait $$backend_pid 2>/dev/null; \
	echo "✅ 前后端开发服务已全部停止"

# 启动前端开发服务（开发调试用）
dev-frontend:
	@echo "🚀 正在启动前端开发服务..."
	@pnpm dev

# 停止前端开发服务（本地测试用）
stop-frontend:
	@echo "🛑 正在停止前端服务..."
	@pkill -f "vite.*ark-go-starter" 2>/dev/null && echo "✅ 已停止" || echo "⚠️  没有运行中的前端服务"

# ============================================================
# 其他工具
# ============================================================

# 列出所有可用的应用程序
list-apps:
	@echo "📂 可用的应用程序:"
	@ls -1 ./apps

# 运行代码检查工具（在 workspace 各模块目录内分别执行）
# 说明：go.work 在仓库根，DiskPath 形如 ./apps/api、./pkg，均相对仓库根，
# 故在仓库根（make 默认 cwd）读取并 cd 到各模块。
lint:
	@echo "🔍 正在运行代码检查工具..."
	@go work edit -json | grep -o '"DiskPath": "[^"]*"' | awk '{gsub(/"/, "", $$2); print $$2}' | while read mod; do \
		echo "  => lint $$mod"; \
		(cd $$mod && golangci-lint run ./...); \
	done

# 运行前端 ESLint（apps/web 与 packages/* 源码；e2e workspace 独立，未纳入）
lint-frontend:
	@echo "🔍 正在运行前端 ESLint..."
	@pnpm lint

# ============================================================
# 契约校验（P5 试点）
# ============================================================
# 校验 contracts/openapi spec 合法性 + 生成物（pkg/client、packages/api-client）与 spec 一致。
# 任一不一致即 exit 非 0（CI 阻断）。详见 contracts/verify.sh 头部说明。
contracts-verify:
	@bash contracts/verify.sh

# ============================================================
# 帮助信息
# ============================================================
help:
	@echo ""
	@echo "🆘 可用命令："
	@echo ""
	@echo "  构建 & 运行"
	@echo "    make build     APP=<名称>              本机环境构建"
	@echo "    make build-env APP=<名称>              指定环境变量构建（适合交叉编译）"
	@echo "    make run       APP=<名称>              go run 直接启动（开发调试）"
	@echo "    make clean                             清理构建产物"
	@echo "    make all       APP=<名称>              clean → deps → build → run"
	@echo ""
	@echo "  测试 & 检查"
	@echo "    make test      APP=<名称>              运行单元测试（pkg + 指定应用）"
	@echo "    make test-integration APP=<名称>     运行集成测试（需 PG/Redis）"
	@echo "    make e2e       [APP=<名称>]            两端端到端测试（后端真端口 + 前端 Playwright）"
	@echo "    make e2e-backend APP=<名称>          后端真端口 e2e（需 PG/Redis）"
	@echo "    make e2e-frontend                      前端 Playwright e2e（需后端 8099 已运行）"
	@echo "    make e2e-install                       安装前端依赖与 Playwright 浏览器（首次）"
	@echo "    make e2e-deps-check                    探测本地 PG/Redis 是否就绪"
	@echo "    make lint                              运行 golangci-lint"
	@echo "    make lint-frontend                     运行前端 ESLint（apps/web + packages/*）"
	@echo "    make contracts-verify                  契约校验（spec 合法 + Go/TS 生成物与 spec 一致）"
	@echo ""
	@echo "  依赖管理"
	@echo "    make deps                              同步 workspace 并更新全部模块依赖"
	@echo "    make tidy                              对所有模块执行 go mod tidy"
	@echo "    make update-dep ARGS=\"-u <pkg@ver>\"   升级指定依赖包"
	@echo ""
	@echo "  代码生成"
	@echo "    make swag      APP=<名称>              生成 Swagger 文档"
	@echo "    make codegen   APP=<名称> COMMAND=<api|module|model>  生成代码"
	@echo ""
	@echo "  Docker"
	@echo "    make docker-build APP=<名称>           构建 Docker 镜像"
	@echo "    make docker-run   APP=<名称> [PORT=N]  运行容器（镜像不存在时自动构建）"
	@echo ""
	@echo "  前端"
	@echo "    make dev-all       [APP=<名称>]        同时启动前后端（启动前先清理旧进程，Ctrl+C 一并停止）"
	@echo "    make dev-frontend                      启动前端开发服务"
	@echo "    make stop-frontend                     停止前端开发服务（本地测试）"
	@echo "    make stop-dev                          停止本地开发进程（后端+前端+端口残留，幂等）"
	@echo ""
	@echo "  一键跑通（clone 即用，仅 dev）"
	@echo "    make quickstart                        有 Docker：自动起 PG/Redis + 建表 + 初始化 IAM + 起前后端"
	@echo "    make quickstart-local                  无 Docker：复用本机 PG(5432)/Redis(6379)"
	@echo "    make quickstart-verify                 对运行中的环境跑 e2e 验收（E2E_IAM=1）"
	@echo ""
	@echo "  其他"
	@echo "    make list-apps                         列出所有可用的应用程序"
	@echo "    make help                              显示此帮助信息"
	@echo ""
	@echo "📝 Docker 镜像标签规则："
	@echo "   有 git tag  →  <tag>-<commit>   示例：demo:v1.0.0-abc1234"
	@echo "   无 git tag  →  <commit>         示例：demo:abc1234"
	@echo ""
