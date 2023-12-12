package daemon

import (
	"github.com/sevlyar/go-daemon"
	"github.com/silenceper/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/text3cn/good/config"
	"github.com/text3cn/good/kit"
	"github.com/text3cn/good/types"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
)

var targetProcess *exec.Cmd
var targetProgrameName string
var cfg types.DaemonProcessConfig

func AddCommand(command *types.Command) {
	// 加载配置
	cfg = types.DaemonProcessConfig{RuntimePath: "./runtime"}
	// 从配置文件加载配置
	filename, _ := filepath.Abs("./app.yaml")
	if kit.FileExist(filename) {
		appCfg := viper.New()
		appCfg.AddConfigPath(config.Currpath)
		appCfg.SetConfigName("app")
		appCfg.SetConfigType("yaml")
		if err := appCfg.ReadInConfig(); err != nil {
			panic(err)
		}
		cfg.RuntimePath = appCfg.GetString("runtime.path")
	}

	// start
	command.RootCmd.AddCommand(&cobra.Command{
		Use:   "start",
		Short: "Start a program to run it in background",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				log.Error("Please specify the program name.")
				return
			}
			if len(args) > 0 {
				targetProgrameName = args[0]
				forkSelf()
				log.Info("Start " + targetProgrameName + " success")
			}
		},
	})

	// stop
	command.RootCmd.AddCommand(&cobra.Command{
		Use:   "stop",
		Short: "Stop the running programe",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				log.Error("Please specify the program name.")
				return
			}
			if len(args) > 0 {
				stopChild(args[0])
			}
		},
	})

	// status
	command.RootCmd.AddCommand(&cobra.Command{
		Use:   "status",
		Short: "Check goodle application status",
		Run: func(cmd *cobra.Command, args []string) {
			// TODO: 读取配置文件看有哪些服务需要启动，然后检查各服务状态
			// drawControl()
		},
	})
}

// 先 fork 出一个 good 进程，然后再 fork 出的 good 进程中前台启动目标程序
// fork 成功后父进程 return，子进程运行时脱离控制台，
// 子进程向控制台打印日志时，会被定向到 /dev/null 所以控制台是没有输出的，
// 因此需要将子进程的输出保存到文件中。
func forkSelf() {
	runtimePath := cfg.RuntimePath
	kit.MkDir(runtimePath, 0777)
	ctx := &daemon.Context{
		PidFileName: filepath.Join(runtimePath, targetProgrameName+".pid"),
		PidFilePerm: 0644,
		LogFileName: filepath.Join(runtimePath, targetProgrameName+".log"),
		LogFilePerm: 0640,
		Umask:       027,
		WorkDir:     "./",                                      // 子进程工作目录
		Args:        []string{"", "start", targetProgrameName}, // 传递给子进程的参
	}
	// 拷贝上下文创建子进程
	child, err := ctx.Reborn()
	defer ctx.Release()
	if err != nil {
		log.Error("Failed to create child process, " + err.Error())
		return
	}
	if child != nil {
		// 父进程工作完成，直接在这退出了
		return
	}
	// 启动目标进程
	targetProcess = exec.Command("./" + targetProgrameName) // 被启动的程序
	targetProcess.Stdout = os.Stdout
	targetProcess.Stderr = os.Stderr
	targetProcess.Start()

	// 优雅关闭，退出进程有以下四种信号：
	// SIGINT  : 前台运行模式下 Windows/Linux 都可以通过 Ctrl+C 键来产生 SIGINT 信号请求中断进程
	// SIGQUIT : 与 SIGINT 类似，前台模式下 Ctrl+\ 通知进程中断，唯一不同是默认会产生 core 文件
	// SIGTERM : 通过 kill pid 命令结束后台进程
	// SIGKILL : 通过 kill -9 pid 命令强制结束后台进程
	// 除了 SIGKILL 信号无法被 Golang 捕获，其余三个信号都是能被阻塞和捕获的。
	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT) // 订阅指定信号
	<-quit                                                                // 阻塞当前 Goroutine 等待订阅的中断信号
	log.Info("Stop " + targetProgrameName + " process")
	targetProcess.Process.Signal(syscall.SIGTERM) // 向目标进程发送中断信号

	// TODO: 优雅退出，调用Server.Shutdown graceful结束()
	// httpcore.Serve 可以返回回调函数在这里启动服务，然后 Shutdown
	//if err := server.Shutdown(context.Background()); err != nil {
	//	log.Fatal("Server Shutdown:", err)
	//}
	// 验证, 在控制器写个 time.Sleep(10 * time.Second) ，然后浏览器访问，然后杀进程看会不会不会等 10 秒睡完再结束
	// 还需要超时处理，如果控制器中睡 1 个小时或者死循环那就无法退出了
	//ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	//defer cancel()
	//if err := srv.Shutdown(ctx); err != nil {
	//	log.Fatal("server shutdown error:", err)
	//}
	//select {
	//case <-ctx.Done():
	//	log.Println("timeout of 5 seconds")
	//}
	//log.Println("server exiting")
}

// 停止子进程
func stopChild(childProcessName string) {
	var pid int
	var err error
	var process *os.Process
	runtimePath := cfg.RuntimePath
	pidfile := filepath.Join(runtimePath, childProcessName+".pid")
	if pid, err = daemon.ReadPidFile(pidfile); err != nil {
		log.Error("Pid not found")
		return
	}
	process, err = os.FindProcess(pid)    // 通过 pid 获取子进程
	err = process.Signal(syscall.SIGTERM) // 给 fork 出的子进程发送中断信号
	if err != nil {
		log.Error("Stop " + childProcessName + " fail" + err.Error())
	} else {
		log.Info("Stop fork process successed.")
	}
}
