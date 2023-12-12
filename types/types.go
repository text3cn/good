package types

import "github.com/spf13/cobra"

type Command struct {
	RootCmd *cobra.Command
}

type DaemonProcessConfig struct {
	RuntimePath string
}

type HotCompileConfig struct {
	// 编译输出的文件名，默认使用入口文件所在目录名称作为编译输出程序的名称
	AppName string

	// 编译输出可执行文件所在目录，默认当前目录
	HotCompileOutputDir string

	// 目录 + 可执行程序名
	OutputAppPath string

	// 监听什么类型的文件变更，默认只监听 .go 类型的文件
	WatchExts []string

	// 监听哪些些目录的文件变更，默认只监听当前目录（包括递归子目录）
	WatchPaths []string

	// Extra parameters when running the application
	// - arg1=val1
	CmdArgs []string

	// 在 go build 时需要添加的其他参数，例如 -race
	BuildArgs []string

	// 运行目标程序时需要添加环境变量，默认加载当前环境变量，例如 env1=val1
	Envs []string

	// 是否监听 vendor 文件夹中的文件更改，默认 false
	VendorWatch bool

	// 不需要监听文件更改的目录
	ExcludedDirs []string

	// For packages or files that need to be compiled, use the -p parameter first
	// 主包路径，也可以是单个文件，多个文件以逗号分隔
	BuildPkg string

	// go build 时添加 -tags 参数
	BuildTags string

	// 指定程序是否自动运行
	DisableRun bool

	// 在 go build 之前执行的命令，例如 swag init
	PrevBuildCmds []string

	// 在 go build 完成后执行的命令
	RunCmd string

	// 日志级别: debug, info, warn, error, fatal。默认 debug
	LogLevel string
}

type CrossCompileCfg struct {
	AppName   string
	OutputDir string
}
