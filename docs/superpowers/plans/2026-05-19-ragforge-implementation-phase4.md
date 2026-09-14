# RAGForge Implementation Plan - Phase 4: Knowledge Base Module

> Requires Phases 1-3 to be complete.

---

### Task 4.1: KB DTOs + Service + Controller + Router

**Files:**
- Create: `apps/ragforge/internal/dto/dtokb/request.go`
- Create: `apps/ragforge/internal/dto/dtokb/response.go`
- Create: `apps/ragforge/internal/service/svckb/kb.go`
- Create: `apps/ragforge/internal/controller/ctrkb/kb.go`
- Create: `apps/ragforge/internal/router/kb.go`

- [ ] **Step 1: Create dto/dtokb/request.go**

```go
package dtokb

import "github.com/morehao/golib/gobject"

type KBCreateReq struct {
	Name        string `json:"name" validate:"required" label:"知识库名称"`
	Description string `json:"description"`
}

type KBUpdateReq struct {
	ID          uint   `json:"id" validate:"required" label:"知识库ID"`
	Name        string `json:"name" validate:"required" label:"知识库名称"`
	Description string `json:"description"`
}

type KBDetailReq struct {
	ID uint `form:"id" validate:"required" label:"知识库ID"`
}

type KBDeleteReq struct {
	ID uint `json:"id" validate:"required" label:"知识库ID"`
}

type KBPageListReq struct {
	gobject.PageQuery
	Name string `form:"name"`
}
```

- [ ] **Step 2: Create dto/dtokb/response.go**

```go
package dtokb

import "time"

type KBCreateResp struct {
	ID uint `json:"id"`
}

type KBDetailResp struct {
	ID          uint      `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	KBType      string    `json:"kbType"`
	CreatorID   uint      `json:"creatorID"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type KBPageListResp struct {
	List  []KBDetailResp `json:"list"`
	Total int64          `json:"total"`
}
```

- [ ] **Step 3: Create service/svckb/kb.go**

```go
package svckb

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-go-starter/apps/ragforge/dao"
	"github.com/morehao/ark-go-starter/apps/ragforge/internal/dto/dtokb"
	"github.com/morehao/ark-go-starter/apps/ragforge/model"
	"github.com/morehao/golib/genericdao"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/gutil"
)

type KBSvc interface {
	Create(ctx *gin.Context, req *dtokb.KBCreateReq) (*dtokb.KBCreateResp, error)
	Update(ctx *gin.Context, req *dtokb.KBUpdateReq) error
	Delete(ctx *gin.Context, req *dtokb.KBDeleteReq) error
	Detail(ctx *gin.Context, req *dtokb.KBDetailReq) (*dtokb.KBDetailResp, error)
	PageList(ctx *gin.Context, req *dtokb.KBPageListReq) (*dtokb.KBPageListResp, error)
}

type kbSvc struct {
	kbDao *dao.KnowledgeBaseDao
}

var _ KBSvc = (*kbSvc)(nil)

func NewKBSvc() KBSvc {
	return &kbSvc{
		kbDao: dao.NewKnowledgeBaseDao(),
	}
}

func (svc *kbSvc) Create(ctx *gin.Context, req *dtokb.KBCreateReq) (*dtokb.KBCreateResp, error) {
	entity := &model.KnowledgeBaseEntity{
		Name:        req.Name,
		Description: req.Description,
		TenantID:    1,
		CreatorID:   1,
	}
	if err := svc.kbDao.Insert(ctx, entity); err != nil {
		glog.Errorf(ctx, "[svckb.Create] Insert fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, err
	}
	return &dtokb.KBCreateResp{ID: entity.ID}, nil
}

func (svc *kbSvc) Detail(ctx *gin.Context, req *dtokb.KBDetailReq) (*dtokb.KBDetailResp, error) {
	entity, err := svc.kbDao.GetByID(ctx, req.ID)
	if err != nil {
		glog.Errorf(ctx, "[svckb.Detail] GetByID fail, err:%v, id:%d", err, req.ID)
		return nil, err
	}
	return &dtokb.KBDetailResp{
		ID:          entity.ID,
		Name:        entity.Name,
		Description: entity.Description,
		KBType:      string(entity.KBType),
		CreatorID:   entity.CreatorID,
		CreatedAt:   entity.CreatedAt,
		UpdatedAt:   entity.UpdatedAt,
	}, nil
}

func (svc *kbSvc) Update(ctx *gin.Context, req *dtokb.KBUpdateReq) error {
	if err := svc.kbDao.UpdateByID(ctx, req.ID, map[string]interface{}{
		"name":        req.Name,
		"description": req.Description,
	}); err != nil {
		glog.Errorf(ctx, "[svckb.Update] UpdateByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return err
	}
	return nil
}

func (svc *kbSvc) Delete(ctx *gin.Context, req *dtokb.KBDeleteReq) error {
	if err := svc.kbDao.Delete(ctx, req.ID, 1); err != nil {
		glog.Errorf(ctx, "[svckb.Delete] Delete fail, err:%v, id:%d", err, req.ID)
		return err
	}
	return nil
}

func (svc *kbSvc) PageList(ctx *gin.Context, req *dtokb.KBPageListReq) (*dtokb.KBPageListResp, error) {
	cond := &dao.KnowledgeBaseCond{
		BaseCond: genericdao.NewBaseCond(&req.PageQuery),
		TenantID: 1,
		Name:     req.Name,
	}
	list, total, err := svc.kbDao.GetPageListByCond(ctx, cond)
	if err != nil {
		glog.Errorf(ctx, "[svckb.PageList] GetPageListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, err
	}
	items := make([]dtokb.KBDetailResp, len(list))
	for i, item := range list {
		items[i] = dtokb.KBDetailResp{
			ID:          item.ID,
			Name:        item.Name,
			Description: item.Description,
			KBType:      string(item.KBType),
			CreatorID:   item.CreatorID,
			CreatedAt:   item.CreatedAt,
			UpdatedAt:   item.UpdatedAt,
		}
	}
	return &dtokb.KBPageListResp{List: items, Total: total}, nil
}
```

- [ ] **Step 4: Create controller/ctrkb/kb.go**

```go
package ctrkb

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-go-starter/apps/ragforge/internal/dto/dtokb"
	"github.com/morehao/ark-go-starter/apps/ragforge/internal/service/svckb"
	"github.com/morehao/ark-go-starter/pkg/gincontext"
)

type KBCtr interface {
	Create(ctx *gin.Context)
	Update(ctx *gin.Context)
	Delete(ctx *gin.Context)
	Detail(ctx *gin.Context)
	PageList(ctx *gin.Context)
}

type kbCtr struct {
	kbSvc svckb.KBSvc
}

var _ KBCtr = (*kbCtr)(nil)

func NewKBCtr() KBCtr {
	return &kbCtr{
		kbSvc: svckb.NewKBSvc(),
	}
}

func (ctr *kbCtr) Create(ctx *gin.Context) {
	var req dtokb.KBCreateReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.kbSvc.Create(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

func (ctr *kbCtr) Update(ctx *gin.Context) {
	var req dtokb.KBUpdateReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.kbSvc.Update(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, nil)
}

func (ctr *kbCtr) Delete(ctx *gin.Context) {
	var req dtokb.KBDeleteReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.kbSvc.Delete(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, nil)
}

func (ctr *kbCtr) Detail(ctx *gin.Context) {
	var req dtokb.KBDetailReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.kbSvc.Detail(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

func (ctr *kbCtr) PageList(ctx *gin.Context) {
	var req dtokb.KBPageListReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.kbSvc.PageList(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}
```

- [ ] **Step 5: Create router/kb.go**

```go
package router

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-go-starter/apps/ragforge/internal/controller/ctrkb"
)

func kbRouter(group *gin.RouterGroup) {
	ctr := ctrkb.NewKBCtr()
	group.POST("/kb/create", ctr.Create)
	group.POST("/kb/delete", ctr.Delete)
	group.POST("/kb/update", ctr.Update)
	group.GET("/kb/detail", ctr.Detail)
	group.POST("/kb/pageList", ctr.PageList)
}
```

- [ ] **Step 6: Register KB routes in router.go**

Update `apps/ragforge/internal/router/router.go`:

```go
package router

import (
	"github.com/morehao/ark-go-starter/pkg/ginserver"
)

func RegisterRouter(groups *ginserver.RouterGroups, appName string) {
	v1RouterGroup := groups.MustGetGroup("v1")

	kbRouter(v1RouterGroup)
}
```

- [ ] **Step 7: Verify compilation**

```bash
cd apps/ragforge && go build ./...
```
Expected: Build succeeds.

- [ ] **Step 8: Commit**

```bash
git add apps/ragforge/internal/dto/dtokb/ apps/ragforge/internal/service/svckb/ apps/ragforge/internal/controller/ctrkb/ apps/ragforge/internal/router/
git commit -m "feat(ragforge): add knowledge base module"
```
