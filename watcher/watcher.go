package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/golang/glog"
)

var (
	currpath      string
	cmd           *exec.Cmd
	excludedPaths StrFlags
	fileExts      StrFlags
	extraPaths    StrFlags
	delay         int64
	state         sync.Mutex
	eventTime     = make(map[string]int64)
	scheduleTime  time.Time
)

func init() {
	currpath, _ = os.Getwd()
	flag.Var(&extraPaths, "i", "list paths to include extra.")
	flag.Var(&excludedPaths, "e", "List of paths to exclude.(default [vendor])")
	flag.Var(&fileExts, "f", "List of file extension(default [.go])")
	flag.Int64Var(&delay, "d", 500, "delay time when recv fs notify(Millisecond)")
}

type StrFlags []string

func (s *StrFlags) String() string {
	return fmt.Sprintf("%s", *s)
}

func (s *StrFlags) Set(value string) error {
	*s = append(*s, value)
	return nil
}

// __FILE__ returns the file name in which the function was invoked
func FILE() string {
	_, file, _, _ := runtime.Caller(1)
	return file
}

// __LINE__ returns the line number at which the function was invoked
func LINE() int {
	_, _, line, _ := runtime.Caller(1)
	return line
}

// NewWatcher starts an fsnotify Watcher on the specified paths
func NewWatcher(delay time.Duration, paths []string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		glog.Fatalf("Failed to create watcher: %s", err)
	}

	buildEvent := make(chan struct{}, 10)

	go func() {
		pending := false
		ticker := time.NewTicker(delay)
		for {
			select {
			case <-buildEvent:
				pending = true
				ticker.Stop()
				ticker = time.NewTicker(delay)
			case <-ticker.C:
				if pending {
					AutoBuild()
					pending = false
				}
			}
		}
	}()

	go func() {
		for {
			select {
			case e := <-watcher.Events:
				isBuild := true

				if !shouldWatchFileWithExtension(e.Name) {
					continue
				}

				mt := GetFileModTime(e.Name)
				if t := eventTime[e.Name]; mt == t {
					glog.V(7).Info(e.String())
					isBuild = false
				}

				eventTime[e.Name] = mt

				if isBuild {
					glog.V(7).Infof("Event fired: %s", e)
					buildEvent <- struct{}{}
				}
			case err := <-watcher.Errors:
				glog.Warningf("Watcher error: %s", err.Error()) // No need to exit here
			}
		}
	}()

	glog.Info("Initializing watcher...")
	for _, path := range paths {
		glog.V(6).Infof("Watching: %s", path)
		err = watcher.Add(path)
		if err != nil {
			glog.Fatalf("Failed to watch directory: %s", err)
		}
	}
}

// AutoBuild builds the specified set of files
func AutoBuild() {
	state.Lock()
	defer state.Unlock()

	os.Chdir(currpath)

	_cmd := exec.Command("make")
	output, err := _cmd.CombinedOutput()
	glog.Info("##################################################################################\n")
	if err != nil {
		glog.Error(string(output))
		glog.Error(err)
		return
	}

	glog.V(3).Info(string(output))
	glog.V(3).Info("Built Successfully!")
	Restart()
}

// Kill kills the running command process
func Kill() {
	defer func() {
		if e := recover(); e != nil {
			glog.Infof("Kill recover: %s", e)
		}
	}()
	if cmd != nil && cmd.Process != nil {
		glog.V(3).Infof("kill %d\n", cmd.Process.Pid)
		err := cmd.Process.Kill()
		if err != nil {
			glog.V(3).Infof("Error while killing cmd process: %s", err)
		}
	}
}

// Restart kills the running command process and starts it again
func Restart() {
	glog.V(3).Infof("Kill running process %s %d\n", FILE(), LINE())
	Kill()
	go Start()
}

func getCmd() string {
	_cmd := exec.Command("make", "-s", "devrun")
	output, _ := _cmd.Output()
	return strings.TrimSpace(strings.Split(string(output), "\n")[0])
}

// Start starts the command process
func Start() {
	c := strings.Fields(getCmd())
	cmd = exec.Command(c[0], c[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	go func() {
		err := cmd.Run()
		if err != nil {
			glog.Infof("process(%s) exit(%v)", strings.Join(c, " "), err)
		} else {
			glog.Infof("process exit(0)")
		}

	}()

	time.Sleep(time.Second)
	glog.V(3).Infof("%v running...", c)
}

// shouldWatchFileWithExtension returns true if the name of the file
// hash a suffix that should be watched.
func shouldWatchFileWithExtension(name string) bool {
	for _, s := range fileExts {
		if strings.HasSuffix(name, s) {
			return true
		}
	}
	return false
}

// If a file is excluded
func isExcluded(file string) bool {
	for _, p := range excludedPaths {
		absP, err := filepath.Abs(p)
		if err != nil {
			glog.Errorf("Cannot get absolute path of '%s'", p)
			continue
		}
		absFilePath, err := filepath.Abs(file)
		if err != nil {
			glog.Errorf("Cannot get absolute path of '%s'", file)
			break
		}
		if strings.HasPrefix(absFilePath, absP) {
			glog.V(4).Infof("'%s' is not being watched", file)
			return true
		}
	}
	return false
}

func readAppDirectories(directory string, paths *[]string) {
	fileInfos, err := ioutil.ReadDir(directory)
	if err != nil {
		return
	}

	useDirectory := false
	for _, fileInfo := range fileInfos {
		if isExcluded(path.Join(directory, fileInfo.Name())) {
			continue
		}

		if fileInfo.IsDir() && fileInfo.Name()[0] != '.' {
			readAppDirectories(directory+"/"+fileInfo.Name(), paths)
			continue
		}

		if useDirectory {
			continue
		}

		if shouldWatchFileWithExtension(fileInfo.Name()) {
			*paths = append(*paths, directory)
			useDirectory = true
		}
	}
}

// GetFileModTime returns unix timestamp of `os.File.ModTime` for the given path.
func GetFileModTime(path string) int64 {
	path = strings.Replace(path, "\\", "/", -1)
	f, err := os.Open(path)
	if err != nil {
		//glog.Errorf("Failed to open file on '%s': %s", path, err)
		return time.Now().Unix()
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		glog.Errorf("Failed to get file stats: %s", err)
		return time.Now().Unix()
	}
	return fi.ModTime().Unix()
}

func main() {
	paths := []string{"."}

	flag.Parse()

	if len(excludedPaths) == 0 {
		excludedPaths.Set("vendor")
	}

	if len(fileExts) == 0 {
		fileExts.Set(".go")
	}

	paths = append(paths, extraPaths...)

	glog.Infof("excludedPaths %v", excludedPaths)
	glog.Infof("watch file exts %v", fileExts)
	glog.Infof("include paths %v", paths)

	readAppDirectories(currpath, &paths)
	NewWatcher(time.Millisecond*time.Duration(delay), paths)
	AutoBuild()
	select {}
}
