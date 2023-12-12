package crosscompile

import (
	"github.com/silenceper/log"
	"github.com/spf13/cobra"
	"github.com/text3cn/good/config"
	"github.com/text3cn/good/kit"
	"github.com/text3cn/good/types"
	"os"
	"os/exec"
	"path/filepath"
)

var buildFilePath string

func AddCommand(command *types.Command) {
	dev := &cobra.Command{
		Use:   "build",
		Short: "Hot compilation",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) > 0 {
				platform := args[0]
				if platform != "linux" && platform != "mac" && platform != "windows" {
					log.Error("Platform not support")
					return
				}
				// 创建目录
				if config.CrossCompileCfg.OutputDir != "./" {
					kit.MkDir(config.CrossCompileCfg.OutputDir, 0777)
				}
				buildFilePath = config.CrossCompileCfg.OutputDir + string(filepath.Separator) + config.CrossCompileCfg.AppName
				// 构建
				pkg := ""
				if len(args) > 1 {
					pkg = args[1]
				}

				switch platform {
				case "linux":
					build("linux", pkg)
				case "mac":
					build("darwin", pkg)
				case "windows":
					build("windows", pkg)
				}
			}
		},
	}
	command.RootCmd.AddCommand(dev)
}

func build(platform, pkg string) {
	// 设置命令和参数
	if platform == "windows" {
		buildFilePath += ".exe"
	}
	args := []string{"build", "-ldflags=-s -w", "-o", buildFilePath}
	if pkg != "" {
		args = append(args, pkg)
	}
	cmd := exec.Command("go", args...)

	// 设置环境变量
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0", "GOOS="+platform, "GOARCH=amd64")

	// 设置输出到标准输出和标准错误
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// 执行命令
	err := cmd.Run()
	if err != nil {
		log.Errorf("Build Fail: %s\n", err)
		return
	}
	log.Info("Build Successed.")
}
