package main

import (
	"github.com/spf13/cobra" // https://github.com/spf13/cobra
	"github.com/text3cn/good/commands/crosscompile"
	"github.com/text3cn/good/commands/daemon"
	"github.com/text3cn/good/commands/hotcompile"
	"github.com/text3cn/good/config"
	"github.com/text3cn/good/types"
)

func main() {
	config.LoadConfig()
	RunConsole()
}

func RunConsole() {
	var cobraRoot = &cobra.Command{
		// 定义根命令的关键字
		Use: "good",
		// 简短介绍
		Short: "Goodle Framework Development Tool",
		// 根命令的执行函数
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.InitDefaultHelpFlag()
			return cmd.Help()
		},
		// 不需要出现 cobra 默认的 completion 子命令
		CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
	}
	var command = &types.Command{
		RootCmd: cobraRoot,
	}
	// 绑定指令
	hotcompile.AddCommand(command)
	daemon.AddCommand(command)
	crosscompile.AddCommand(command)

	// 命令行运行，执行 RootCommand
	command.RootCmd.Execute()
}
