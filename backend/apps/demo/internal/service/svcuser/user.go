package svcuser

import (
	"context"
	"fmt"

	"github.com/morehao/ark-go-starter/demo/dao"
	"github.com/morehao/ark-go-starter/demo/internal/dto/dtouser"
	"github.com/morehao/ark-go-starter/demo/model"
	"github.com/morehao/ark-go-starter/demo/object/objuser"
	"github.com/morehao/ark-go-starter/pkg/code"
	"github.com/morehao/ark-go-starter/pkg/dbclient"
	"github.com/morehao/golib/biz/gobject"
	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/gutil"
	"gorm.io/gorm"
)

type UserSvc interface {
	Create(ctx context.Context, req *dtouser.UserCreateReq) (*dtouser.UserCreateResp, error)
	CreateInTx(ctx context.Context, req *dtouser.UserCreateInTxReq) (*dtouser.UserCreateInTxResp, error)
	Delete(ctx context.Context, req *dtouser.UserDeleteReq) error
	Update(ctx context.Context, req *dtouser.UserUpdateReq) error
	Detail(ctx context.Context, req *dtouser.UserDetailReq) (*dtouser.UserDetailResp, error)
	PageList(ctx context.Context, req *dtouser.UserPageListReq) (*dtouser.UserPageListResp, error)
}

type userSvc struct {
}

var _ UserSvc = (*userSvc)(nil)

func NewUserSvc() UserSvc {
	return &userSvc{}
}

// Create 创建用户管理
func (svc *userSvc) Create(ctx context.Context, req *dtouser.UserCreateReq) (*dtouser.UserCreateResp, error) {
	insertEntity := &model.UserEntity{
		CompanyID:    req.CompanyID,
		DepartmentID: req.DepartmentID,
		Name:         req.Name,
		CreatedBy:    req.Operator.UserID,
		UpdatedBy:    req.Operator.UserID,
	}

	if err := dao.NewUserDao().Insert(ctx, insertEntity); err != nil {
		glog.Errorf(ctx, "[svcuser.UserCreate] dao Create fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.UserCreateError)
	}
	return &dtouser.UserCreateResp{
		UserID: insertEntity.ID,
	}, nil
}

// CreateInTx 事务示例：在一个事务里创建两条用户记录，任一失败则整体回滚。
//
// 关键点（样板示范）：
//   - 事务入口统一走 dbclient.DemoDB(ctx).Transaction(...)，不要自行 db.Begin()；
//   - 事务内的 DAO 必须用 dao.NewUserDao().WithTx(tx) 绑定到同一个 tx，
//     否则 DAO 会走事务外的连接，写入不参与本事务；
//   - 回调返回非 nil error 即触发回滚，把底层错误原样向上传递（不要吞掉）。
func (svc *userSvc) CreateInTx(ctx context.Context, req *dtouser.UserCreateInTxReq) (*dtouser.UserCreateInTxResp, error) {
	resp := &dtouser.UserCreateInTxResp{UserIDs: make([]uint, 0, 2)}

	txErr := dbclient.DemoDB(ctx).Transaction(func(tx *gorm.DB) error {
		// 事务内 DAO：必须 WithTx 绑定。
		userDao := dao.NewUserDao().WithTx(tx)

		first := &model.UserEntity{Name: req.FirstName}
		if err := userDao.Insert(ctx, first); err != nil {
			return err
		}
		resp.UserIDs = append(resp.UserIDs, first.ID)

		// 演示回滚：第一条已写入，这里人为制造错误，验证整体原子性。
		if req.FailAfterFirst {
			return fmt.Errorf("demand failure after first insert")
		}

		second := &model.UserEntity{Name: req.SecondName}
		if err := userDao.Insert(ctx, second); err != nil {
			return err
		}
		resp.UserIDs = append(resp.UserIDs, second.ID)
		return nil
	})
	if txErr != nil {
		glog.Errorf(ctx, "[svcuser.UserCreateInTx] Transaction fail, err:%v", txErr)
		return nil, code.GetError(code.UserCreateError)
	}
	return resp, nil
}

// Delete 删除用户管理
func (svc *userSvc) Delete(ctx context.Context, req *dtouser.UserDeleteReq) error {
	userEntity, err := dao.NewUserDao().GetByID(ctx, req.UserID)
	if err != nil {
		glog.Errorf(ctx, "[svcuser.UserDelete] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.UserDeleteError)
	}
	if userEntity == nil || userEntity.ID == 0 {
		return code.GetError(code.UserNotExistError)
	}

	if err := dao.NewUserDao().Delete(ctx, req.UserID, req.Operator.UserID); err != nil {
		glog.Errorf(ctx, "[svcuser.Delete] dao Delete fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.UserDeleteError)
	}
	return nil
}

// Update 更新用户管理
func (svc *userSvc) Update(ctx context.Context, req *dtouser.UserUpdateReq) error {
	userEntity, err := dao.NewUserDao().GetByID(ctx, req.UserID)
	if err != nil {
		glog.Errorf(ctx, "[svcuser.UserUpdate] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.UserUpdateError)
	}
	if userEntity == nil || userEntity.ID == 0 {
		return code.GetError(code.UserNotExistError)
	}

	// 只写请求显式携带的字段（指针非 nil），避免漏传字段把列清空。
	updateMap := map[string]any{
		"updated_by": req.Operator.UserID,
	}
	if req.CompanyID != nil {
		updateMap["company_id"] = *req.CompanyID
	}
	if req.DepartmentID != nil {
		updateMap["department_id"] = *req.DepartmentID
	}
	if req.Name != nil {
		updateMap["name"] = *req.Name
	}
	if err := dao.NewUserDao().UpdateMap(ctx, req.UserID, updateMap); err != nil {
		glog.Errorf(ctx, "[svcuser.UserUpdate] dao UpdateMap fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.UserUpdateError)
	}
	return nil
}

// Detail 根据id获取用户管理
func (svc *userSvc) Detail(ctx context.Context, req *dtouser.UserDetailReq) (*dtouser.UserDetailResp, error) {
	userEntity, err := dao.NewUserDao().GetByID(ctx, req.UserID)
	if err != nil {
		glog.Errorf(ctx, "[svcuser.UserDetail] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.UserGetDetailError)
	}
	if userEntity == nil || userEntity.ID == 0 {
		return nil, code.GetError(code.UserNotExistError)
	}
	resp := &dtouser.UserDetailResp{
		UserID: userEntity.ID,
		UserBaseInfo: objuser.UserBaseInfo{
			CompanyID:    userEntity.CompanyID,
			DepartmentID: userEntity.DepartmentID,
			Name:         userEntity.Name,
		},
		OperatorBaseInfo: gobject.OperatorBaseInfo{
			CreatedBy: userEntity.CreatedBy,
			UpdatedBy: userEntity.UpdatedBy,
			CreatedAt: userEntity.CreatedAt.Unix(),
			UpdatedAt: userEntity.UpdatedAt.Unix(),
		},
	}
	return resp, nil
}

// PageList 分页获取用户管理列表
func (svc *userSvc) PageList(ctx context.Context, req *dtouser.UserPageListReq) (*dtouser.UserPageListResp, error) {
	cond := &dao.UserCond{
		BaseCond: &gormdao.BaseCond{
			Page:     req.Page,
			PageSize: req.PageSize,
			// 分页必须有稳定排序，否则 offset 翻页可能重复/漏行。
			OrderField: `"id" DESC`,
		},
		CompanyID:    req.CompanyID,
		DepartmentID: req.DepartmentID,
		Name:         req.Name,
	}
	userEntityList, total, err := dao.NewUserDao().GetPageListByCond(ctx, cond)
	if err != nil {
		glog.Errorf(ctx, "[svcuser.UserPageList] dao GetPageListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.UserGetPageListError)
	}
	list := make([]dtouser.UserPageListItem, 0, len(userEntityList))
	for _, v := range userEntityList {
		list = append(list, dtouser.UserPageListItem{
			UserID: v.ID,
			UserBaseInfo: objuser.UserBaseInfo{
				CompanyID:    v.CompanyID,
				DepartmentID: v.DepartmentID,
				Name:         v.Name,
			},
			OperatorBaseInfo: gobject.OperatorBaseInfo{
				CreatedBy: v.CreatedBy,
				UpdatedBy: v.UpdatedBy,
				UpdatedAt: v.UpdatedAt.Unix(),
			},
		})
	}
	return &dtouser.UserPageListResp{
		List:  list,
		Total: total,
	}, nil
}
