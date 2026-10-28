package objuser

// UserBaseInfo 用户基础信息（创建、详情、列表共用）。
type UserBaseInfo struct {
	CompanyID    uint   `json:"companyID" form:"companyID"`       // 公司id
	DepartmentID uint   `json:"departmentID" form:"departmentID"` // 部门id
	Name         string `json:"name" form:"name"`                 // 姓名
}
