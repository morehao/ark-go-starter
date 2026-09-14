# RAGForge Implementation Plan - Phase 1: Scaffold

> **For agentic workers:** Use subagent-driven-development to implement.
> See also: `docs/superpowers/specs/2026-05-19-ragforge-design.md`

**Goal:** Create `apps/ragforge` project skeleton with go.mod, cmd, config, app.go

**Architecture:** Standard ark-go-starter layered architecture

**Reference:** Source WeKnora at `/Users/morehao/Documents/study/go/go-ai/WeKnora/`

---

### Task 1.1: Create app scaffold

**Files:**
- Create: `apps/ragforge/go.mod`
- Create: `apps/ragforge/app.go`
- Create: `apps/ragforge/cmd/main.go`
- Create: `apps/ragforge/cmd/init.go`
- Create: `apps/ragforge/config/config.go`
- Create: `apps/ragforge/config/config.yaml`

- [ ] **Step 1: Create directories**

```bash
mkdir -p apps/ragforge/cmd apps/ragforge/config apps/ragforge/internal/router apps/ragforge/internal/middleware apps/ragforge/internal/engine apps/ragforge/model apps/ragforge/dao apps/ragforge/object/objrag apps/ragforge/internal/dto apps/ragforge/internal/controller apps/ragforge/internal/service apps/ragforge/docs apps/ragforge/scripts
```

- [ ] **Step 2: Create go.mod**

```bash
cd apps/ragforge && go mod init github.com/morehao/ark-go-starter/apps/ragforge
```

Then edit go.mod to add:

```
require (
	github.com/gin-gonic/gin v1.12.0
	github.com/morehao/golib v0.1.1
	github.com/stretchr/testify v1.11.1
	github.com/sashabaranov/go-openai v1.17.7+incompatible
	github.com/pgvector/pgvector-go v0.2.2
	gorm.io/gorm v1.31.1
	gorm.io/driver/postgres v1.5.11
)

replace github.com/morehao/golib => ../../../golib
```

- [ ] **Step 3: Create config/config.go**

```go
package config

import (
	"github.com/morehao/golib/dbgorm"
	"github.com/morehao/golib/dbredis"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/gtrace"
	"github.com/morehao/golib/gutil"
)

var Conf *Config

type Config struct {
	Server      Server                    `yaml:"server"`
	Log         map[string]glog.LogConfig `yaml:"log"`
	Trace       gtrace.TraceConfig        `yaml:"trace"`
	DBConfigs   []dbgorm.GormConfig       `yaml:"db_configs"`
	RedisConfig dbredis.RedisConfig       `yaml:"redis_config"`
}

type Server struct {
	Name string `yaml:"name"`
	Port int    `yaml:"port"`
	Env  string `yaml:"env"`
}

func InitConf(configPath string) {
	Conf = LoadConfig(configPath)
}

func LoadConfig(configPath string) *Config {
	var c Config
	gutil.LoadYamlConfig(configPath, &c)
	return &c
}
```

- [ ] **Step 4: Create config/config.yaml**

```yaml
server:
  name: ragforge
  port: 8082
  env: dev

log:
  default:
    level: debug
    format: console
    output: stdout

db_configs:
  - name: default
    driver: postgres
    dsn: postgres://postgres:123456@127.0.0.1:5432/ragforge?sslmode=disable
    max_open_conns: 50
    max_idle_conns: 10
    conn_max_lifetime: 30m

redis_config:
  addr: 127.0.0.1:6379
  password: ""
  db: 0
```

- [ ] **Step 5: Create app.go**

```go
package ragforge

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-go-starter/apps/ragforge/config"
	"github.com/morehao/ark-go-starter/apps/ragforge/internal/router"
	"github.com/morehao/ark-go-starter/pkg/ginserver"
)

const AppName = "ragforge"

func Routers(engine *gin.Engine) {
	routerGroups := ginserver.NewRouterGroups(engine, AppName, ginserver.Version{
		Name: "v1",
	})
	if config.Conf.Server.Env == "dev" {
		// gindocs.Register(engine.Group("/"+AppName), AppName)
	}
	router.RegisterRouter(routerGroups, AppName)
}
```

- [ ] **Step 6: Create cmd/main.go**

```go
package main

import (
	"fmt"

	"github.com/gin-gonic/gin"
	ragforge "github.com/morehao/ark-go-starter/apps/ragforge"
	"github.com/morehao/ark-go-starter/apps/ragforge/config"
	"github.com/morehao/golib/glog"
)

func main() {
	serverInit()

	gin.SetMode(gin.DebugMode)
	defer glog.Close()

	engine := gin.New()
	engine.Use(gin.Recovery())
	ragforge.Routers(engine)

	addr := fmt.Sprintf(":%d", config.Conf.Server.Port)
	engine.Run(addr)
}
```

- [ ] **Step 7: Create cmd/init.go**

```go
package main

import (
	"flag"

	"github.com/morehao/ark-go-starter/apps/ragforge/config"
	"github.com/morehao/ark-go-starter/pkg/dbclient"
	"github.com/morehao/golib/glog"
)

var configPath = flag.String("config", "config/config.yaml", "config file path")

func serverInit() {
	flag.Parse()
	config.InitConf(*configPath)
	glog.InitLogger(config.Conf.Log)
	dbclient.InitMultiDB(config.Conf.DBConfigs)
}
```

- [ ] **Step 8: Create stub router.go** (placeholder, will expand later)

```go
package router

import (
	"github.com/morehao/ark-go-starter/pkg/ginserver"
)

func RegisterRouter(groups *ginserver.RouterGroups, appName string) {
	// Routes will be added in subsequent phases
}
```

- [ ] **Step 9: Run go mod tidy**

```bash
cd apps/ragforge && go mod tidy
```

- [ ] **Step 10: Verify compilation**

```bash
cd apps/ragforge && go build ./...
```
Expected: Build succeeds.

- [ ] **Step 11: Commit**

```bash
git add apps/ragforge/
git commit -m "feat(ragforge): add project scaffold"
```
