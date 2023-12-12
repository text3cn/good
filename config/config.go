package config

import (
	"github.com/silenceper/log"
	"github.com/spf13/viper"
	"github.com/text3cn/good/kit"
	"github.com/text3cn/good/types"
	"os"
	"path"
	"path/filepath"
	"runtime"
)

var (
	HotCompileCfg   *types.HotCompileConfig
	CrossCompileCfg *types.CrossCompileCfg
	Currpath        string
	Exit            chan bool
	BuildPkg        string
	Started         chan bool
)

func LoadConfig() {
	HotCompileCfg = &types.HotCompileConfig{}
	CrossCompileCfg = &types.CrossCompileCfg{}
	Currpath, _ = os.Getwd()

	// 从配置文件加载配置
	filename, _ := filepath.Abs("./good.yaml")
	if kit.FileExist(filename) {
		goodCfg := viper.New()
		goodCfg.AddConfigPath(Currpath)
		goodCfg.SetConfigName("good")
		goodCfg.SetConfigType("yaml")
		if err := goodCfg.ReadInConfig(); err != nil {
			panic(err)
		}

		// 热编译
		if goodCfg.IsSet("hotCompilation.outputDir") {
			HotCompileCfg.HotCompileOutputDir = goodCfg.GetString("hotCompilation.outputDir")
		}
		if goodCfg.IsSet("hotCompilation.appName") {
			HotCompileCfg.AppName = goodCfg.GetString("hotCompilation.appName")
		}
		if goodCfg.IsSet("hotCompilation.watchExts") {
			HotCompileCfg.WatchExts = goodCfg.GetStringSlice("hotCompilation.watchExts")
		}
		if goodCfg.IsSet("hotCompilation.watchDirs") {
			HotCompileCfg.WatchPaths = goodCfg.GetStringSlice("hotCompilation.watchDirs")
		}
		if goodCfg.IsSet("hotCompilation.excludedDirs") {
			HotCompileCfg.ExcludedDirs = goodCfg.GetStringSlice("hotCompilation.excludedDirs")
		}
		if goodCfg.IsSet("hotCompilation.prevBuildCmds") {
			HotCompileCfg.ExcludedDirs = goodCfg.GetStringSlice("hotCompilation.prevBuildCmds")
		}

		// 交叉编译
		if goodCfg.IsSet("crossCompilation.appName") {
			CrossCompileCfg.AppName = goodCfg.GetString("crossCompilation.appName")
		}
		if goodCfg.IsSet("crossCompilation.outputDir") {
			CrossCompileCfg.OutputDir = goodCfg.GetString("crossCompilation.outputDir")
		}
	}
	setHotCompileDefaultConfig()
	setCrossCompileDefaultConfig()
}

// 热编译默认配置
func setHotCompileDefaultConfig() {
	if HotCompileCfg.AppName == "" {
		HotCompileCfg.AppName = path.Base(Currpath)
	}

	outputExt := ""
	if runtime.GOOS == "windows" {
		outputExt = ".exe"
	}
	if HotCompileCfg.HotCompileOutputDir == "" {
		HotCompileCfg.OutputAppPath = "./" + HotCompileCfg.AppName + outputExt
	} else {
		HotCompileCfg.OutputAppPath = HotCompileCfg.HotCompileOutputDir + string(filepath.Separator) + HotCompileCfg.AppName + outputExt
	}

	HotCompileCfg.WatchExts = append(HotCompileCfg.WatchExts, ".go")

	// set log level, default is debug
	if HotCompileCfg.LogLevel != "" {
		setLogLevel(HotCompileCfg.LogLevel)
	}
}

// 交叉编译默认配置
func setCrossCompileDefaultConfig() {
	if CrossCompileCfg.AppName == "" {
		CrossCompileCfg.AppName = path.Base(Currpath)
	}
	if CrossCompileCfg.OutputDir == "" {
		CrossCompileCfg.OutputDir = "./"
	}
}

func setLogLevel(level string) {
	switch level {
	case "debug":
		log.SetLogLevel(log.LevelDebug)
	case "info":
		log.SetLogLevel(log.LevelInfo)
	case "warn":
		log.SetLogLevel(log.LevelWarning)
	case "error":
		log.SetLogLevel(log.LevelError)
	case "fatal":
		log.SetLogLevel(log.LevelFatal)
	default:
		log.SetLogLevel(log.LevelDebug)
	}
}
