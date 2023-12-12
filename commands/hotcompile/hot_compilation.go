package hotcompile

import (
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/mitchellh/go-ps"
	"github.com/silenceper/log"
	"github.com/spf13/cobra"
	"github.com/text3cn/good/config"
	"github.com/text3cn/good/types"
	"io/ioutil"
	"os"
	"os/exec"
	path "path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"
)

var (
	cmd          *exec.Cmd
	eventTime    = make(map[string]int64)
	scheduleTime time.Time
)
var ignoredFilesRegExps = []string{
	`.#(\w+).go$`,
	`.(\w+).go.swp$`,
	`(\w+).go~$`,
	`(\w+).tmp$`,
}

func AddCommand(command *types.Command) {
	// 一级命令
	dev := &cobra.Command{
		Use:   "dev",
		Short: "Hot compilation",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) > 0 {
				config.BuildPkg = args[0]
			}
			HotCompilationRun()
		},
	}
	// 二级命令
	//dev.AddCommand(&cobra.Command{
	//	Use:     "xxx",
	//	Short:   "xxxx",
	//	Aliases: []string{"f"},
	//	Example: "./main start f",
	//	Run: func(cmd *cobra.Command, args []string) {
	//
	//	},
	//})
	command.RootCmd.AddCommand(dev)
}

func HotCompilationRun() {
	var paths []string
	ReadAppDirectories(config.Currpath, &paths)
	for _, path := range config.HotCompileCfg.WatchPaths {
		ReadAppDirectories(path, &paths)
	}
	files := []string{}
	if config.BuildPkg == "" {
		config.BuildPkg = config.HotCompileCfg.BuildPkg
	}
	if config.BuildPkg != "" {
		files = strings.Split(config.BuildPkg, ",")
	}
	NewWatcher(paths, files)
	Autobuild(files)
	<-config.Exit
	runtime.Goexit()
}

// NewWatcher new watcher
func NewWatcher(paths []string, files []string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Errorf(" Fail to create new Watcher[ %s ]\n", err)
		os.Exit(2)
	}

	go func() {
		for {
			select {
			case e := <-watcher.Events:
				isbuild := true

				// Skip ignored files
				if shouldIgnoreFile(e.Name) {
					continue
				}
				if !checkIfWatchExt(e.Name) {
					continue
				}

				mt := getFileModTime(e.Name)
				if t := eventTime[e.Name]; mt == t {
					// log.Infof("[SKIP] # %s #\n", e.String())
					isbuild = false
				}

				eventTime[e.Name] = mt

				if isbuild {
					go func() {
						// Wait 1s before autobuild util there is no file change.
						scheduleTime = time.Now().Add(1 * time.Second)
						for {
							time.Sleep(time.Until(scheduleTime))
							if time.Now().After(scheduleTime) {
								break
							}
						}

						Autobuild(files)
					}()
				}
			case err := <-watcher.Errors:
				log.Errorf("%v", err)
				log.Warnf(" %s\n", err.Error()) // No need to exit here
			}
		}
	}()

	log.Infof("Initializing watcher...\n")
	for _, path := range paths {
		log.Infof("Directory( %s )\n", path)
		err = watcher.Add(path)
		if err != nil {
			log.Errorf("Fail to watch directory[ %s ]\n", err)
			os.Exit(2)
		}
	}
}

// 获取文件最后修改时间
func getFileModTime(path string) int64 {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return time.Now().Unix()
	}
	modificationTime := fileInfo.ModTime()
	return modificationTime.Unix()
}

var building bool

// Autobuild auto build
//
//nolint:funlen
func Autobuild(files []string) {
	if building {
		log.Infof("still in building...\n")
		return
	}
	building = true
	defer func() {
		building = false
	}()

	log.Infof("Start building...\n")

	if err := os.Chdir(config.Currpath); err != nil {
		log.Errorf("Chdir Error: %+v\n", err)
		return
	}

	for _, prevCmd := range config.HotCompileCfg.PrevBuildCmds {
		log.Infof("Run external cmd '%s'", prevCmd)
		cmdArr := strings.Split(prevCmd, " ")
		prevCmdExec := exec.Command(cmdArr[0])
		prevCmdExec.Env = append(os.Environ(), config.HotCompileCfg.Envs...)
		prevCmdExec.Args = cmdArr
		prevCmdExec.Stdout = os.Stdout
		prevCmdExec.Stderr = os.Stderr
		err := prevCmdExec.Run()
		if err != nil {
			panic(err)
		}
	}

	cmdName := "go"

	var err error

	args := []string{"build"}
	args = append(args, "-o", config.HotCompileCfg.OutputAppPath)
	args = append(args, config.HotCompileCfg.BuildArgs...)
	if config.HotCompileCfg.BuildTags != "" {
		args = append(args, "-tags", config.HotCompileCfg.BuildTags)
	}
	args = append(args, files...)

	bcmd := exec.Command(cmdName, args...)
	bcmd.Env = append(os.Environ(), "GOGC=off")
	bcmd.Stdout = os.Stdout
	bcmd.Stderr = os.Stderr
	log.Infof("Build Args: %s %s", cmdName, strings.Join(args, " "))
	err = bcmd.Run()

	if err != nil {
		log.Errorf("============== Build failed ===================\n")
		log.Errorf("%+v\n", err)
		return
	}
	log.Infof("Build was successful\n")
	if !config.HotCompileCfg.DisableRun {
		if len(config.HotCompileCfg.RunCmd) != 0 {
			Restart(config.HotCompileCfg.RunCmd, true)
		} else {
			Restart(config.HotCompileCfg.OutputAppPath, false)
		}
	}
}

// Kill kill main process and all its children
func Kill() {
	defer func() {
		if e := recover(); e != nil {
			fmt.Println("Kill.recover -> ", e)
		}
	}()
	if cmd != nil && cmd.Process != nil {
		// err := cmd.Process.Kill()
		err := killAllProcesses(cmd.Process.Pid)
		if err != nil {
			fmt.Println("Kill -> ", err)
		}
	}
}

// kill main process and all its children
func killAllProcesses(pid int) (err error) {
	hasAllKilled := make(chan bool)
	go func() {
		pids, err := psTree(pid)
		if err != nil {
			log.Fatalf("getting all sub processes error: %v\n", err)
			return
		}
		log.Debugf("main pid: %d", pid)
		log.Debugf("pids: %+v", pids)

		for _, subPid := range pids {
			_ = killProcess(subPid)
		}

		waitForProcess(pid, hasAllKilled)
	}()

	// finally kill the main process
	<-hasAllKilled
	log.Debugf("killing MAIN process pid: %d", pid)
	err = cmd.Process.Kill()
	if err != nil {
		return
	}
	log.Debugf("kill MAIN process succeed")

	return
}

func killProcess(pid int) (err error) {
	log.Debugf("killing process pid: %d", pid)
	ps, err := os.FindProcess(pid)
	if err != nil {
		log.Errorf("find process %d error: %v\n", pid, err)
		return
	}
	err = ps.Kill()
	if err != nil {
		log.Errorf("killing process %d error: %v\n", pid, err)
		// retry
		time.AfterFunc(2*time.Second, func() {
			log.Debugf("retry killing process pid: %d", pid)
			_ = killProcess(pid)
		})
		return
	}
	return
}

// implement pstree based on the cross-platform ps utility in go, go-ps
func psTree(rootPid int) (res []int, err error) {
	pidOfInterest := map[int]struct{}{rootPid: {}}
	pss, err := ps.Processes()
	if err != nil {
		fmt.Println("ERROR: ", err)
		return
	}

	// we must sort the ps by ppid && pid first, otherwise we probably will miss some sub-processes
	// of the root process during for-range searching
	sort.Slice(pss, func(i, j int) bool {
		ppidLess := pss[i].PPid() < pss[j].PPid()
		pidLess := pss[i].PPid() == pss[j].PPid() && pss[i].Pid() < pss[j].Pid()

		return ppidLess || pidLess
	})

	for _, ps := range pss {
		ppid := ps.PPid()
		if _, exists := pidOfInterest[ppid]; exists {
			pidOfInterest[ps.Pid()] = struct{}{}
		}
	}

	for pid := range pidOfInterest {
		if pid != rootPid {
			res = append(res, pid)
		}
	}

	return
}

func waitForProcess(pid int, hasAllKilled chan bool) {
	pids, _ := psTree(pid)
	if len(pids) == 0 {
		hasAllKilled <- true
		return
	}

	log.Infof("still waiting for %d processes %+v to exit", len(pids), pids)
	time.AfterFunc(time.Second, func() {
		waitForProcess(pid, hasAllKilled)
	})
}

// Restart restart app
func Restart(appname string, isCmd bool) {
	// log.Debugf("kill running process")
	Kill()
	go Start(appname, isCmd)
}

// Start start app
func Start(appname string, isCmd bool) {
	log.Infof("Restarting %s ...\n", appname)
	if !strings.HasPrefix(appname, "./") && !isCmd {
		appname = "./" + appname
	}

	cmd = exec.Command(appname)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Args = append([]string{appname}, config.HotCompileCfg.CmdArgs...)
	cmd.Env = append(os.Environ(), config.HotCompileCfg.Envs...)
	log.Infof("Run %s", strings.Join(cmd.Args, " "))
	go func() {
		_ = cmd.Run()
	}()

	log.Infof("%s is running...\n", appname)
	config.Started <- true
}

// Should ignore filenames generated by
// Emacs, Vim or SublimeText
func shouldIgnoreFile(filename string) bool {
	for _, regex := range ignoredFilesRegExps {
		r, err := regexp.Compile(regex)
		if err != nil {
			panic("Could not compile the regex: " + regex)
		}
		if r.MatchString(filename) {
			return true
		}
		continue
	}
	return false
}

// checkIfWatchExt returns true if the name HasSuffix <watch_ext>.
func checkIfWatchExt(name string) bool {
	for _, s := range config.HotCompileCfg.WatchExts {
		if strings.HasSuffix(name, s) {
			return true
		}
	}
	return false
}

func ReadAppDirectories(directory string, paths *[]string) {
	fileInfos, err := ioutil.ReadDir(directory)
	if err != nil {
		return
	}

	useDirectory := false
	for _, fileInfo := range fileInfos {
		if strings.HasSuffix(fileInfo.Name(), "docs") {
			continue
		}
		if strings.HasSuffix(fileInfo.Name(), "swagger") {
			continue
		}

		if !config.HotCompileCfg.VendorWatch && strings.HasSuffix(fileInfo.Name(), "vendor") {
			continue
		}

		if isExcluded(path.Join(directory, fileInfo.Name())) {
			continue
		}

		if fileInfo.IsDir() && fileInfo.Name()[0] != '.' {
			ReadAppDirectories(directory+"/"+fileInfo.Name(), paths)
			continue
		}
		if useDirectory {
			continue
		}
		*paths = append(*paths, directory)
		useDirectory = true
	}
}

// If a file is excluded
func isExcluded(filePath string) bool {
	for _, p := range config.HotCompileCfg.ExcludedDirs {
		absP, err := path.Abs(p)
		if err != nil {
			log.Errorf("err =%v", err)
			log.Errorf("Can not get absolute path of [ %s ]\n", p)
			continue
		}
		absFilePath, err := path.Abs(filePath)
		if err != nil {
			log.Errorf("Can not get absolute path of [ %s ]\n", filePath)
			break
		}
		if strings.HasPrefix(absFilePath, absP) {
			log.Infof("Excluding from watching [ %s ]\n", filePath)
			return true
		}
	}
	return false
}
