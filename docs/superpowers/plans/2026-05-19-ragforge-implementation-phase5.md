# RAGForge Implementation Plan - Phase 5: Remaining Business Modules

> Requires Phases 1-4 to be complete.

---

### Task 5.1: Knowledge Document Module

**Files:**
- Create: `apps/ragforge/internal/dto/dtoknowledge/request.go`
- Create: `apps/ragforge/internal/dto/dtoknowledge/response.go`
- Create: `apps/ragforge/internal/service/svcknowledge/knowledge.go`
- Create: `apps/ragforge/internal/controller/ctrknowledge/knowledge.go`
- Create: `apps/ragforge/internal/router/knowledge.go`

- [ ] **Step 1: Create dto/dtoknowledge/request.go**

```go
package dtoknowledge

import "github.com/morehao/golib/gobject"

type KnowledgeCreateFileReq struct {
	KbID  uint   `form:"kbID" validate:"required" label:"知识库ID"`
	Title string `form:"title" label:"文档标题"`
}

type KnowledgeCreateURLReq struct {
	KbID      uint   `json:"kbID" validate:"required" label:"知识库ID"`
	SourceURL string `json:"sourceURL" validate:"required" label:"URL地址"`
	Title     string `json:"title" label:"文档标题"`
}

type KnowledgeCreateManualReq struct {
	KbID    uint   `json:"kbID" validate:"required" label:"知识库ID"`
	Title   string `json:"title" validate:"required" label:"文档标题"`
	Content string `json:"content" validate:"required" label:"文档内容"`
}

type KnowledgeUpdateReq struct {
	ID      uint   `json:"id" validate:"required" label:"知识文档ID"`
	Title   string `json:"title"`
	Content string `json:"content"`
}

type KnowledgeDeleteReq struct {
	ID uint `json:"id" validate:"required" label:"知识文档ID"`
}

type KnowledgeDetailReq struct {
	ID uint `form:"id" validate:"required" label:"知识文档ID"`
}

type KnowledgePageListReq struct {
	gobject.PageQuery
	KbID        uint   `form:"kbID"`
	Type        string `form:"type"`
	ParseStatus string `form:"parseStatus"`
}

type KnowledgeReparseReq struct {
	ID uint `json:"id" validate:"required" label:"知识文档ID"`
}
```

- [ ] **Step 2: Create dto/dtoknowledge/response.go**

```go
package dtoknowledge

import "time"

type KnowledgeCreateResp struct {
	ID uint `json:"id"`
}

type KnowledgeDetailResp struct {
	ID          uint      `json:"id"`
	KbID        uint      `json:"kbID"`
	Type        string    `json:"type"`
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	FileURL     string    `json:"fileURL"`
	SourceURL   string    `json:"sourceURL"`
	ParseStatus string    `json:"parseStatus"`
	FileSize    int64     `json:"fileSize"`
	CreatorID   uint      `json:"creatorID"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type KnowledgePageListResp struct {
	List  []KnowledgeDetailResp `json:"list"`
	Total int64                 `json:"total"`
}
```

- [ ] **Step 3: Create service/svcknowledge/knowledge.go**

```go
package svcknowledge

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-go-starter/apps/ragforge/dao"
	"github.com/morehao/ark-go-starter/apps/ragforge/internal/dto/dtoknowledge"
	"github.com/morehao/ark-go-starter/apps/ragforge/model"
	"github.com/morehao/golib/genericdao"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/gutil"
)

type KnowledgeSvc interface {
	CreateFromFile(ctx *gin.Context, req *dtoknowledge.KnowledgeCreateFileReq) (*dtoknowledge.KnowledgeCreateResp, error)
	CreateFromURL(ctx *gin.Context, req *dtoknowledge.KnowledgeCreateURLReq) (*dtoknowledge.KnowledgeCreateResp, error)
	CreateManual(ctx *gin.Context, req *dtoknowledge.KnowledgeCreateManualReq) (*dtoknowledge.KnowledgeCreateResp, error)
	Update(ctx *gin.Context, req *dtoknowledge.KnowledgeUpdateReq) error
	Delete(ctx *gin.Context, req *dtoknowledge.KnowledgeDeleteReq) error
	Detail(ctx *gin.Context, req *dtoknowledge.KnowledgeDetailReq) (*dtoknowledge.KnowledgeDetailResp, error)
	PageList(ctx *gin.Context, req *dtoknowledge.KnowledgePageListReq) (*dtoknowledge.KnowledgePageListResp, error)
	Reparse(ctx *gin.Context, req *dtoknowledge.KnowledgeReparseReq) error
}

type knowledgeSvc struct {
	knowledgeDao *dao.KnowledgeDao
}

var _ KnowledgeSvc = (*knowledgeSvc)(nil)

func NewKnowledgeSvc() KnowledgeSvc {
	return &knowledgeSvc{
		knowledgeDao: dao.NewKnowledgeDao(),
	}
}

func (svc *knowledgeSvc) CreateFromFile(ctx *gin.Context, req *dtoknowledge.KnowledgeCreateFileReq) (*dtoknowledge.KnowledgeCreateResp, error) {
	file, header, err := ctx.Request.FormFile("file")
	if err != nil {
		glog.Errorf(ctx, "[svcknowledge.CreateFromFile] FormFile fail, err:%v", err)
		return nil, err
	}
	defer file.Close()

	entity := &model.KnowledgeEntity{
		KbID:        req.KbID,
		TenantID:    1,
		Type:        model.KnowledgeTypeFile,
		Title:       req.Title,
		FileURL:     "", // TODO: save file to storage
		FileSize:    header.Size,
		ParseStatus: model.ParseStatusPending,
		CreatorID:   1,
	}
	if err := svc.knowledgeDao.Insert(ctx, entity); err != nil {
		glog.Errorf(ctx, "[svcknowledge.CreateFromFile] Insert fail, err:%v", err)
		return nil, err
	}
	return &dtoknowledge.KnowledgeCreateResp{ID: entity.ID}, nil
}

func (svc *knowledgeSvc) CreateFromURL(ctx *gin.Context, req *dtoknowledge.KnowledgeCreateURLReq) (*dtoknowledge.KnowledgeCreateResp, error) {
	entity := &model.KnowledgeEntity{
		KbID:        req.KbID,
		TenantID:    1,
		Type:        model.KnowledgeTypeURL,
		Title:       req.Title,
		SourceURL:   req.SourceURL,
		ParseStatus: model.ParseStatusPending,
		CreatorID:   1,
	}
	if err := svc.knowledgeDao.Insert(ctx, entity); err != nil {
		glog.Errorf(ctx, "[svcknowledge.CreateFromURL] Insert fail, err:%v", err)
		return nil, err
	}
	return &dtoknowledge.KnowledgeCreateResp{ID: entity.ID}, nil
}

func (svc *knowledgeSvc) CreateManual(ctx *gin.Context, req *dtoknowledge.KnowledgeCreateManualReq) (*dtoknowledge.KnowledgeCreateResp, error) {
	entity := &model.KnowledgeEntity{
		KbID:        req.KbID,
		TenantID:    1,
		Type:        model.KnowledgeTypeManual,
		Title:       req.Title,
		Content:     req.Content,
		ParseStatus: model.ParseStatusSuccess,
		CreatorID:   1,
	}
	if err := svc.knowledgeDao.Insert(ctx, entity); err != nil {
		glog.Errorf(ctx, "[svcknowledge.CreateManual] Insert fail, err:%v", err)
		return nil, err
	}
	return &dtoknowledge.KnowledgeCreateResp{ID: entity.ID}, nil
}

func (svc *knowledgeSvc) Update(ctx *gin.Context, req *dtoknowledge.KnowledgeUpdateReq) error {
	updates := map[string]interface{}{}
	if req.Title != "" {
		updates["title"] = req.Title
	}
	if req.Content != "" {
		updates["content"] = req.Content
	}
	if err := svc.knowledgeDao.UpdateByID(ctx, req.ID, updates); err != nil {
		glog.Errorf(ctx, "[svcknowledge.Update] fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return err
	}
	return nil
}

func (svc *knowledgeSvc) Delete(ctx *gin.Context, req *dtoknowledge.KnowledgeDeleteReq) error {
	return svc.knowledgeDao.Delete(ctx, req.ID, 1)
}

func (svc *knowledgeSvc) Detail(ctx *gin.Context, req *dtoknowledge.KnowledgeDetailReq) (*dtoknowledge.KnowledgeDetailResp, error) {
	entity, err := svc.knowledgeDao.GetByID(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	return &dtoknowledge.KnowledgeDetailResp{
		ID:          entity.ID,
		KbID:        entity.KbID,
		Type:        string(entity.Type),
		Title:       entity.Title,
		Content:     entity.Content,
		FileURL:     entity.FileURL,
		SourceURL:   entity.SourceURL,
		ParseStatus: string(entity.ParseStatus),
		FileSize:    entity.FileSize,
		CreatorID:   entity.CreatorID,
		CreatedAt:   entity.CreatedAt,
		UpdatedAt:   entity.UpdatedAt,
	}, nil
}

func (svc *knowledgeSvc) PageList(ctx *gin.Context, req *dtoknowledge.KnowledgePageListReq) (*dtoknowledge.KnowledgePageListResp, error) {
	cond := &dao.KnowledgeCond{
		BaseCond:      genericdao.NewBaseCond(&req.PageQuery),
		KbID:          req.KbID,
		TenantID:      1,
		KnowledgeType: req.Type,
		ParseStatus:   req.ParseStatus,
	}
	list, total, err := svc.knowledgeDao.GetPageListByCond(ctx, cond)
	if err != nil {
		return nil, err
	}
	items := make([]dtoknowledge.KnowledgeDetailResp, len(list))
	for i, item := range list {
		items[i] = dtoknowledge.KnowledgeDetailResp{
			ID:          item.ID,
			KbID:        item.KbID,
			Type:        string(item.Type),
			Title:       item.Title,
			ParseStatus: string(item.ParseStatus),
			FileSize:    item.FileSize,
			CreatorID:   item.CreatorID,
			CreatedAt:   item.CreatedAt,
		}
	}
	return &dtoknowledge.KnowledgePageListResp{List: items, Total: total}, nil
}

func (svc *knowledgeSvc) Reparse(ctx *gin.Context, req *dtoknowledge.KnowledgeReparseReq) error {
	return svc.knowledgeDao.UpdateByID(ctx, req.ID, map[string]interface{}{
		"parse_status": model.ParseStatusPending,
	})
}
```

- [ ] **Step 4: Create controller/ctrknowledge/knowledge.go**

```go
package ctrknowledge

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-go-starter/apps/ragforge/internal/dto/dtoknowledge"
	"github.com/morehao/ark-go-starter/apps/ragforge/internal/service/svcknowledge"
	"github.com/morehao/ark-go-starter/pkg/gincontext"
)

type KnowledgeCtr interface {
	CreateFromFile(ctx *gin.Context)
	CreateFromURL(ctx *gin.Context)
	CreateManual(ctx *gin.Context)
	Update(ctx *gin.Context)
	Delete(ctx *gin.Context)
	Detail(ctx *gin.Context)
	PageList(ctx *gin.Context)
	Reparse(ctx *gin.Context)
}

type knowledgeCtr struct {
	knowledgeSvc svcknowledge.KnowledgeSvc
}

var _ KnowledgeCtr = (*knowledgeCtr)(nil)

func NewKnowledgeCtr() KnowledgeCtr {
	return &knowledgeCtr{
		knowledgeSvc: svcknowledge.NewKnowledgeSvc(),
	}
}

func (ctr *knowledgeCtr) CreateFromFile(ctx *gin.Context) {
	var req dtoknowledge.KnowledgeCreateFileReq
	if err := ctx.ShouldBind(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.knowledgeSvc.CreateFromFile(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

func (ctr *knowledgeCtr) CreateFromURL(ctx *gin.Context) {
	var req dtoknowledge.KnowledgeCreateURLReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.knowledgeSvc.CreateFromURL(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

func (ctr *knowledgeCtr) CreateManual(ctx *gin.Context) {
	var req dtoknowledge.KnowledgeCreateManualReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.knowledgeSvc.CreateManual(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

func (ctr *knowledgeCtr) Update(ctx *gin.Context) {
	var req dtoknowledge.KnowledgeUpdateReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.knowledgeSvc.Update(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, nil)
}

func (ctr *knowledgeCtr) Delete(ctx *gin.Context) {
	var req dtoknowledge.KnowledgeDeleteReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.knowledgeSvc.Delete(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, nil)
}

func (ctr *knowledgeCtr) Detail(ctx *gin.Context) {
	var req dtoknowledge.KnowledgeDetailReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.knowledgeSvc.Detail(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

func (ctr *knowledgeCtr) PageList(ctx *gin.Context) {
	var req dtoknowledge.KnowledgePageListReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.knowledgeSvc.PageList(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

func (ctr *knowledgeCtr) Reparse(ctx *gin.Context) {
	var req dtoknowledge.KnowledgeReparseReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.knowledgeSvc.Reparse(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, nil)
}
```

- [ ] **Step 5: Create router/knowledge.go**

```go
package router

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-go-starter/apps/ragforge/internal/controller/ctrknowledge"
)

func knowledgeRouter(group *gin.RouterGroup) {
	ctr := ctrknowledge.NewKnowledgeCtr()
	group.POST("/knowledge/createFile", ctr.CreateFromFile)
	group.POST("/knowledge/createUrl", ctr.CreateFromURL)
	group.POST("/knowledge/createManual", ctr.CreateManual)
	group.POST("/knowledge/update", ctr.Update)
	group.POST("/knowledge/delete", ctr.Delete)
	group.GET("/knowledge/detail", ctr.Detail)
	group.POST("/knowledge/pageList", ctr.PageList)
	group.POST("/knowledge/reparse", ctr.Reparse)
}
```

- [ ] **Step 6: Register knowledge routes in router.go**

Add `knowledgeRouter(v1RouterGroup)` after `kbRouter`.

- [ ] **Step 7: Verify compilation**

```bash
cd apps/ragforge && go build ./...
```

- [ ] **Step 8: Commit**

```bash
git add apps/ragforge/internal/dto/dtoknowledge/ apps/ragforge/internal/service/svcknowledge/ apps/ragforge/internal/controller/ctrknowledge/ apps/ragforge/internal/router/
git commit -m "feat(ragforge): add knowledge document module"
```

---

### Task 5.2: Chunk Module with pgvector Semantic Search

Add `apps/ragforge/dao/chunk_search.go` for pgvector operations, then create chunk service.

**Files:**
- Create: `apps/ragforge/internal/dto/dtochunk/request.go`
- Create: `apps/ragforge/internal/dto/dtochunk/response.go`
- Create: `apps/ragforge/internal/service/svcchunk/chunk.go`
- Create: `apps/ragforge/internal/controller/ctrchunk/chunk.go`
- Create: `apps/ragforge/internal/router/chunk.go`

- [ ] Step 1: Create dto/dtochunk/request.go & response.go (following dtokb pattern)

- [ ] Step 2: Create service/svcchunk/chunk.go with semantic search using pgvector

```go
package svcchunk

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-go-starter/apps/ragforge/dao"
	"github.com/morehao/ark-go-starter/apps/ragforge/internal/dto/dtochunk"
	"github.com/morehao/ark-go-starter/apps/ragforge/internal/engine"
	"github.com/morehao/ark-go-starter/apps/ragforge/model"
	"github.com/morehao/ark-go-starter/pkg/dbclient"
	"github.com/morehao/golib/genericdao"
	"github.com/morehao/golib/glog"
	"github.com/pgvector/pgvector-go"
)

type ChunkSearchReq struct {
	KbID     uint   `json:"kbID" validate:"required" label:"知识库ID"`
	Query    string `json:"query" validate:"required" label:"查询文本"`
	TopK     int    `json:"topK"`
}

type ChunkSearchResp struct {
	List []ChunkSearchItem `json:"list"`
}

type ChunkSearchItem struct {
	ID          uint    `json:"id"`
	KnowledgeID uint    `json:"knowledgeID"`
	Content     string  `json:"content"`
	Score       float64 `json:"score"`
	SeqID       int     `json:"seqID"`
}

type ChunkSvc interface {
	PageList(ctx *gin.Context, req *dtochunk.ChunkPageListReq) (*dtochunk.ChunkPageListResp, error)
	Update(ctx *gin.Context, req *dtochunk.ChunkUpdateReq) error
	Delete(ctx *gin.Context, req *dtochunk.ChunkDeleteReq) error
	Search(ctx *gin.Context, req *ChunkSearchReq) (*ChunkSearchResp, error)
}

type chunkSvc struct {
	chunkDao          *dao.ChunkDao
	embeddingProvider engine.EmbeddingProvider
}

func NewChunkSvc(embeddingProvider engine.EmbeddingProvider) ChunkSvc {
	return &chunkSvc{
		chunkDao:          dao.NewChunkDao(),
		embeddingProvider: embeddingProvider,
	}
}

func (svc *chunkSvc) Search(ctx *gin.Context, req *ChunkSearchReq) (*ChunkSearchResp, error) {
	if req.TopK <= 0 {
		req.TopK = 10
	}
	embedding, err := svc.embeddingProvider.CreateEmbedding(ctx, &engine.EmbeddingRequest{
		Inputs: []string{req.Query},
	})
	if err != nil {
		glog.Errorf(ctx, "[svcchunk.Search] CreateEmbedding fail, err:%v", err)
		return nil, err
	}
	if len(embedding.Embeddings) == 0 {
		return &ChunkSearchResp{List: []ChunkSearchItem{}}, nil
	}
	vector := pgvector.NewVector(embedding.Embeddings[0])
	db := dbclient.DefaultDB(ctx)
	var chunks []model.ChunkEntity
	if err := db.Where("kb_id = ? AND tenant_id = ? AND deleted_at IS NULL", req.KbID, 1).
		Order("vector <-> ?", vector).
		Limit(req.TopK).
		Find(&chunks).Error; err != nil {
		return nil, err
	}
	items := make([]ChunkSearchItem, len(chunks))
	for i, c := range chunks {
		items[i] = ChunkSearchItem{
			ID:          c.ID,
			KnowledgeID: c.KnowledgeID,
			Content:     c.Content,
			SeqID:       c.SeqID,
		}
	}
	return &ChunkSearchResp{List: items}, nil
}

// Standard CRUD methods follow the same pattern as previous modules
```

- [ ] Step 3: Create controller/ctrchunk/chunk.go (following same pattern as ctrkb)

- [ ] Step 4: Create router/chunk.go and register in router.go

---

### Task 5.3: FAQ Module

**Files:**
- Create: `apps/ragforge/internal/dto/dtofaq/request.go`
- Create: `apps/ragforge/internal/dto/dtofaq/response.go`
- Create: `apps/ragforge/internal/service/svcfaq/faq.go`
- Create: `apps/ragforge/internal/controller/ctrfaq/faq.go`
- Create: `apps/ragforge/internal/router/faq.go`

Follow the same CRUD pattern as KB module. FAQ-specific operations:
- `Create` with question + answer + similar_questions
- `Search` - semantic search on FAQ entries
- `Import` - batch import FAQ entries

### Task 5.4: Session + Message Modules

**Files:**
- Create: `apps/ragforge/internal/dto/dtosession/request.go` + `response.go`
- Create: `apps/ragforge/internal/service/svcsession/session.go`
- Create: `apps/ragforge/internal/controller/ctrsession/session.go`
- Create: `apps/ragforge/internal/router/session.go`
- Create: `apps/ragforge/internal/dto/dtomessage/request.go` + `response.go`
- Create: `apps/ragforge/internal/service/svcmessage/message.go`
- Create: `apps/ragforge/internal/controller/ctrmessage/message.go`
- Create: `apps/ragforge/internal/router/message.go`

Session CRUD + GenerateTitle. Message CRUD + List by session.

### Task 5.5: QA Service (Knowledge Chat)

**Files:**
- Create: `apps/ragforge/internal/service/svcqa/qa.go`
- Create: `apps/ragforge/internal/controller/ctrsession/knowledge_chat.go` (or add to session controller)

This is the core RAG pipeline:

```go
package svcqa

type QASvc interface {
	KnowledgeChat(ctx *gin.Context, sessionID uint, query string) error
}

type qaSvc struct {
	embeddingProvider engine.EmbeddingProvider
	llmProvider       engine.LLMProvider
	chunkSvc          *svcchunk.ChunkSvc
	messageDao        *dao.MessageDao
	sessionDao        *dao.SessionDao
}

func (svc *qaSvc) KnowledgeChat(ctx *gin.Context, sessionID uint, query string) error {
	// 1. Save user message
	// 2. Generate embedding for query
	// 3. Search top-k chunks from KB
	// 4. Build prompt with context
	// 5. Call LLM for answer
	// 6. Save assistant message
	// 7. Return answer (stream SSE or full response)
	return nil
}
```

### Task 5.6: Model + VectorStore + Tag Modules

**Files:** Create DTO, Service, Controller, Router for each:
- `internal/dto/dtomodel/`, `internal/service/svcmodel/`, `internal/controller/ctrmodel/`, `internal/router/model.go`
- `internal/dto/dtovectorstore/`, `internal/service/svcvectorstore/`, `internal/controller/ctrvectorstore/`, `internal/router/vector_store.go`
- `internal/dto/dtotag/`, `internal/service/svctag/`, `internal/controller/ctrtag/`, `internal/router/tag.go`

Each follows the same CRUD pattern as the KB module.

### Task 5.7: Integrate all routers in router.go

```go
package router

import (
	"github.com/morehao/ark-go-starter/pkg/ginserver"
)

func RegisterRouter(groups *ginserver.RouterGroups, appName string) {
	v1RouterGroup := groups.MustGetGroup("v1")

	kbRouter(v1RouterGroup)
	knowledgeRouter(v1RouterGroup)
	chunkRouter(v1RouterGroup)
	faqRouter(v1RouterGroup)
	sessionRouter(v1RouterGroup)
	messageRouter(v1RouterGroup)
	modelRouter(v1RouterGroup)
	vectorStoreRouter(v1RouterGroup)
	tagRouter(v1RouterGroup)
}
```

### Task 5.8: Middleware and Integration

**Files:**
- Create: `apps/ragforge/internal/middleware/auth.go`

```go
package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-go-starter/pkg/token"
)

func Auth() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		claims, err := token.ParseToken(ctx)
		if err != nil {
			ctx.AbortWithStatusJSON(401, gin.H{"code": 401, "msg": "unauthorized"})
			return
		}
		ctx.Set("userID", claims.UserID)
		ctx.Set("tenantID", claims.TenantID)
		ctx.Next()
	}
}
```

Update `app.go` to register middleware:

```go
func Routers(engine *gin.Engine) {
	routerGroups := ginserver.NewRouterGroups(engine, AppName, ginserver.Version{
		Name: "v1",
		Middlewares: []gin.HandlerFunc{
			middleware.Auth(),
		},
	})
	// ...
}
```

### Task 5.9: Final verification

```bash
cd apps/ragforge && go build ./...
go vet ./apps/ragforge/...
```

### Task 5.10: Update Makefile

Add to `apps/ragforge/scripts/Makefile` (and/or root Makefile):

```makefile
ragforge_build:
	go build -o bin/ragforge ./apps/ragforge/cmd/

ragforge_run:
	go run ./apps/ragforge/cmd/ -config apps/ragforge/config/config.yaml

ragforge_test:
	go test ./apps/ragforge/...

.PHONY: ragforge_build ragforge_run ragforge_test
```

### Task 5.11: Final commit

```bash
git add apps/ragforge/
git commit -m "feat(ragforge): complete ragforge app with all business modules"
```
