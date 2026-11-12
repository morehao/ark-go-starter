// Package dtohealth 定义健康检查接口的响应 DTO。
package dtohealth

// DbCheckResp 数据源连通性检查响应
type DbCheckResp struct {
	Postgres string `json:"postgres"` // PostgreSQL 连通性检查结果，连通为 ok，否则为错误信息
	Redis    string `json:"redis"`    // Redis 连通性检查结果，连通为 ok，否则为错误信息
}
