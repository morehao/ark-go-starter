package testsetup

// 测试环境支持的应用名(appName，用于 service.name / 日志标识)常量。
const (
	AppNameDemo = "demo"
)

// 应用标识(appName) 与 目录名(AppDir) 解耦（D11）。
//
// 目录名只用于定位仓库内文件（如 config 目录），与「应用逻辑标识」不是一回事：
// demo 应用的 Go 目录已迁为 apps/api，但其 service.name / 日志 service / swagger
// instanceName 仍是 demo。新增应用若目录名≠逻辑名，在 appDirByName 补一条即可，
// 也可用环境变量 APP_DIR 临时覆盖某次测试的目录名。
const (
	AppDirDemo = "api"
)

// appDirByName 是「应用标识 → 目录名」映射。
var appDirByName = map[string]string{
	AppNameDemo: AppDirDemo,
}
