// Package db 存放本服务拥有的数据库结构定义（建表 DDL）及其嵌入入口。
//
// DDL 放在 Go 模块内、而不是仓库根的脚本目录，有两个理由：
//
//  1. 归属：这些表由 apps/api 拥有，数据结构跟着拥有它的服务走；
//  2. 可读性：//go:embed 只能嵌入本包目录下的文件、模式里不能写 ".."，
//     放进模块内才能让 model 包的一致性用例直接读到 DDL 原文，
//     不依赖测试的工作目录（不靠 ../../.. 这种相对路径）。
package db

import (
	_ "embed" // 仅为启用下面的 //go:embed 指令，本包不引用该包的符号
)

// TableInitDDL 是建表脚本原文（PostgreSQL）。
//
// 它是业务表结构的唯一事实源：新环境建库、CI 建表、quickstart 建表都执行这一份；
// model 的 gorm tag 必须与它逐列一致，由 model 包的 TestModelsMatchDDL 守护。
//
//go:embed table_init.sql
var TableInitDDL string
