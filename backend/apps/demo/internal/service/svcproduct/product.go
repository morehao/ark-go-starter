// Package svcproduct 是商品管理服务层：承载业务规则、事务与错误转换。
package svcproduct

import (
	"context"
	"errors"

	"github.com/morehao/ark-go-starter/demo/dao"
	"github.com/morehao/ark-go-starter/demo/internal/dto/dtoproduct"
	"github.com/morehao/ark-go-starter/demo/model"
	"github.com/morehao/ark-go-starter/demo/object/objproduct"
	"github.com/morehao/ark-go-starter/pkg/code"
	"github.com/morehao/ark-go-starter/pkg/dbclient"
	"github.com/morehao/golib/biz/gobject"
	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/gutil"
	"gorm.io/gorm"
)

// ProductSvc 是商品管理服务接口。
type ProductSvc interface {
	Create(ctx context.Context, req *dtoproduct.ProductCreateReq) (*dtoproduct.ProductCreateResp, error)
	CreateInTx(ctx context.Context, req *dtoproduct.ProductCreateInTxReq) (*dtoproduct.ProductCreateInTxResp, error)
	Delete(ctx context.Context, req *dtoproduct.ProductDeleteReq) error
	Update(ctx context.Context, req *dtoproduct.ProductUpdateReq) error
	Detail(ctx context.Context, req *dtoproduct.ProductDetailReq) (*dtoproduct.ProductDetailResp, error)
	PageList(ctx context.Context, req *dtoproduct.ProductPageListReq) (*dtoproduct.ProductPageListResp, error)
}

type productSvc struct{}

var _ ProductSvc = (*productSvc)(nil)

// NewProductSvc 构造 ProductSvc。
func NewProductSvc() ProductSvc {
	return &productSvc{}
}

// Create 创建商品管理
func (svc *productSvc) Create(ctx context.Context, req *dtoproduct.ProductCreateReq) (*dtoproduct.ProductCreateResp, error) {
	insertEntity := &model.ProductEntity{
		Name:       req.Name,
		CategoryID: req.CategoryID,
		Price:      req.Price,
		CreatedBy:  req.UserID,
		UpdatedBy:  req.UserID,
	}

	if err := dao.NewProductDao().Insert(ctx, insertEntity); err != nil {
		glog.Errorf(ctx, "[svcproduct.ProductCreate] dao Create fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.ProductCreateError)
	}
	return &dtoproduct.ProductCreateResp{
		ProductID: insertEntity.ID,
	}, nil
}

// CreateInTx 事务示例：在一个事务里创建两条商品记录，任一失败则整体回滚。
//
// 关键点（样板示范）：
//   - 事务入口统一走 dbclient.DemoDB(ctx).Transaction(...)，不要自行 db.Begin()；
//   - 事务内的 DAO 必须用 dao.NewProductDao().WithTx(tx) 绑定到同一个 tx，
//     否则 DAO 会走事务外的连接，写入不参与本事务；
//   - 回调返回非 nil error 即触发回滚，把底层错误原样向上传递（不要吞掉）。
func (svc *productSvc) CreateInTx(ctx context.Context, req *dtoproduct.ProductCreateInTxReq) (*dtoproduct.ProductCreateInTxResp, error) {
	resp := &dtoproduct.ProductCreateInTxResp{ProductIDs: make([]uint, 0, 2)}

	txErr := dbclient.DemoDB(ctx).Transaction(func(tx *gorm.DB) error {
		// 事务内 DAO：必须 WithTx 绑定。
		productDao := dao.NewProductDao().WithTx(tx)

		first := &model.ProductEntity{Name: req.FirstName}
		if err := productDao.Insert(ctx, first); err != nil {
			return err
		}
		resp.ProductIDs = append(resp.ProductIDs, first.ID)

		// 演示回滚：第一条已写入，这里人为制造错误，验证整体原子性。
		if req.FailAfterFirst {
			return errors.New("demand failure after first insert")
		}

		second := &model.ProductEntity{Name: req.SecondName}
		if err := productDao.Insert(ctx, second); err != nil {
			return err
		}
		resp.ProductIDs = append(resp.ProductIDs, second.ID)
		return nil
	})
	if txErr != nil {
		glog.Errorf(ctx, "[svcproduct.ProductCreateInTx] Transaction fail, err:%v", txErr)
		return nil, code.GetError(code.ProductCreateError)
	}
	return resp, nil
}

// Delete 删除商品管理
func (svc *productSvc) Delete(ctx context.Context, req *dtoproduct.ProductDeleteReq) error {
	productEntity, err := dao.NewProductDao().GetByID(ctx, req.ProductID)
	if err != nil {
		glog.Errorf(ctx, "[svcproduct.ProductDelete] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.ProductDeleteError)
	}
	if productEntity == nil || productEntity.ID == 0 {
		return code.GetError(code.ProductNotExistError)
	}

	if err := dao.NewProductDao().Delete(ctx, req.ProductID, req.UserID); err != nil {
		glog.Errorf(ctx, "[svcproduct.Delete] dao Delete fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.ProductDeleteError)
	}
	return nil
}

// Update 更新商品管理
func (svc *productSvc) Update(ctx context.Context, req *dtoproduct.ProductUpdateReq) error {
	productEntity, err := dao.NewProductDao().GetByID(ctx, req.ProductID)
	if err != nil {
		glog.Errorf(ctx, "[svcproduct.ProductUpdate] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.ProductUpdateError)
	}
	if productEntity == nil || productEntity.ID == 0 {
		return code.GetError(code.ProductNotExistError)
	}

	// 只写请求显式携带的字段（指针非 nil），避免漏传字段把列清空。
	updateMap := map[string]any{
		"updated_by": req.UserID,
	}
	if req.Name != nil {
		updateMap["name"] = *req.Name
	}
	if req.CategoryID != nil {
		updateMap["category_id"] = *req.CategoryID
	}
	if req.Price != nil {
		updateMap["price"] = *req.Price
	}
	if err := dao.NewProductDao().UpdateMap(ctx, req.ProductID, updateMap); err != nil {
		glog.Errorf(ctx, "[svcproduct.ProductUpdate] dao UpdateMap fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.ProductUpdateError)
	}
	return nil
}

// Detail 根据id获取商品管理
func (svc *productSvc) Detail(ctx context.Context, req *dtoproduct.ProductDetailReq) (*dtoproduct.ProductDetailResp, error) {
	productEntity, err := dao.NewProductDao().GetByID(ctx, req.ProductID)
	if err != nil {
		glog.Errorf(ctx, "[svcproduct.ProductDetail] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.ProductGetDetailError)
	}
	if productEntity == nil || productEntity.ID == 0 {
		return nil, code.GetError(code.ProductNotExistError)
	}
	resp := &dtoproduct.ProductDetailResp{
		ProductID: productEntity.ID,
		ProductBaseInfo: objproduct.ProductBaseInfo{
			Name:       productEntity.Name,
			CategoryID: productEntity.CategoryID,
			Price:      productEntity.Price,
		},
		OperatorBaseInfo: gobject.OperatorBaseInfo{
			CreatedBy: productEntity.CreatedBy,
			UpdatedBy: productEntity.UpdatedBy,
			CreatedAt: productEntity.CreatedAt.Unix(),
			UpdatedAt: productEntity.UpdatedAt.Unix(),
		},
	}
	return resp, nil
}

// PageList 分页获取商品管理列表
func (svc *productSvc) PageList(ctx context.Context, req *dtoproduct.ProductPageListReq) (*dtoproduct.ProductPageListResp, error) {
	cond := &dao.ProductCond{
		BaseCond: &gormdao.BaseCond{
			Page:     req.Page,
			PageSize: req.PageSize,
			// 分页必须有稳定排序，否则 offset 翻页可能重复/漏行。
			OrderField: `"id" DESC`,
		},
		Name:       req.Name,
		CategoryID: req.CategoryID,
	}
	productEntityList, total, err := dao.NewProductDao().GetPageListByCond(ctx, cond)
	if err != nil {
		glog.Errorf(ctx, "[svcproduct.ProductPageList] dao GetPageListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.ProductGetPageListError)
	}
	list := make([]dtoproduct.ProductPageListItem, 0, len(productEntityList))
	for _, v := range productEntityList {
		list = append(list, dtoproduct.ProductPageListItem{
			ProductID: v.ID,
			ProductBaseInfo: objproduct.ProductBaseInfo{
				Name:       v.Name,
				CategoryID: v.CategoryID,
				Price:      v.Price,
			},
			OperatorBaseInfo: gobject.OperatorBaseInfo{
				CreatedBy: v.CreatedBy,
				UpdatedBy: v.UpdatedBy,
				UpdatedAt: v.UpdatedAt.Unix(),
			},
		})
	}
	return &dtoproduct.ProductPageListResp{
		List:  list,
		Total: total,
	}, nil
}
