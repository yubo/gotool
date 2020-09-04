package main

import (
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
	"github.com/spf13/cobra"
	"github.com/yubo/golib/logs"
	"k8s.io/klog/v2"
)

type config struct {
	extraPaths    StrFlags
	excludedPaths StrFlags
	fileExts      StrFlags
	delay         int64
	cmd1          string
	cmd2          string
	delay2        time.Duration
	paths         []string
}

type watcher struct {
	*config
	*fsnotify.Watcher
	sync.Mutex

	cmd       *exec.Cmd
	eventTime map[string]int64
	//currpath  string
}

func NewWatcher(cf *config) (*watcher, error) {
	cf.delay2 = time.Millisecond * time.Duration(cf.delay)

	if len(cf.excludedPaths) == 0 {
		cf.excludedPaths.Set("vendor")
	}

	if len(cf.fileExts) == 0 {
		cf.fileExts.Set(".go")
	}

	cf.paths = []string{"."}
	cf.paths = append(cf.paths, cf.extraPaths...)

	klog.Infof("excludedPaths %v", cf.excludedPaths)
	klog.Infof("watch file exts %v", cf.fileExts)
	klog.Infof("include paths %v", cf.paths)

	watcher := &watcher{
		config:    cf,
		eventTime: make(map[string]int64),
		//currpath:  currpath,
	}

	// expend currpath
	dir, _ := os.Getwd()
	//os.Chdir(currpath)
	watcher.readAppDirectories(dir)

	return watcher, nil

}

// NewWatcher starts an fsnotify Watcher on the specified paths
func (p *watcher) Do() (done <-chan error, err error) {
	if p.Watcher, err = fsnotify.NewWatcher(); err != nil {
		return nil, fmt.Errorf("Failed to create watcher: %s", err)
	}

	done = make(chan error, 0)
	buildEvent := make(chan struct{}, 10)

	go func() {
		pending := false
		ticker := time.NewTicker(p.delay2)
		for {
			select {
			case <-buildEvent:
				pending = true
				ticker.Stop()
				ticker = time.NewTicker(p.delay2)
			case <-ticker.C:
				if pending {
					p.AutoBuild()
					pending = false
				}
			}
		}
	}()

	go func() {
		for {
			select {
			case e := <-p.Events:
				isBuild := true

				if !p.shouldWatchFileWithExtension(e.Name) {
					continue
				}

				mt := GetFileModTime(e.Name)
				if t := p.eventTime[e.Name]; mt == t {
					klog.V(7).Info(e.String())
					isBuild = false
				}

				p.eventTime[e.Name] = mt

				if isBuild {
					klog.V(7).Infof("Event fired: %s", e)
					buildEvent <- struct{}{}
				}
			case err := <-p.Errors:
				klog.Warningf("Watcher error: %s", err.Error()) // No need to exit here
			}
		}
	}()

	klog.Info("Initializing watcher...")
	for _, path := range p.paths {
		klog.V(6).Infof("Watching: %s", path)
		if err := p.Add(path); err != nil {
			klog.Fatalf("Failed to watch directory: %s", err)
		}
	}
	p.AutoBuild()
	return
}

// AutoBuild builds the specified set of files
func (p *watcher) AutoBuild() {
	p.Lock()
	defer p.Unlock()

	cmd := command(p.cmd1)
	output, err := cmd.CombinedOutput()
	klog.Info("##################################################################################\n")
	if err != nil {
		klog.Error(string(output))
		klog.Error(err)
		return
	}

	klog.V(3).Info(string(output))
	klog.V(3).Info("Built Successfully!")
	p.Restart()
}

// Kill kills the running command process
func (p *watcher) Kill() {
	defer func() {
		if e := recover(); e != nil {
			klog.Infof("Kill recover: %s", e)
		}
	}()
	if p.cmd != nil && p.cmd.Process != nil {
		klog.V(3).Infof("kill %d\n", p.cmd.Process.Pid)
		err := p.cmd.Process.Kill()
		if err != nil {
			klog.V(3).Infof("Error while killing cmd process: %s", err)
		}
	}
}

// Restart kills the running command process and starts it again
func (p *watcher) Restart() {
	klog.V(3).Infof("Kill running process %s %d\n", FILE(), LINE())
	p.Kill()
	go p.Start()
}

func (p *watcher) getCmd() string {
	cmd := command(p.cmd2)
	output, err := cmd.Output()
	if err != nil {
		klog.Errorf("run %s err %s", p.cmd2, err)
	}
	return strings.TrimSpace(strings.Split(string(output), "\n")[0])
}

// Start starts the command process
func (p *watcher) Start() {
	cmd := p.getCmd()
	p.cmd = command(cmd)
	p.cmd.Stdout = os.Stdout
	p.cmd.Stderr = os.Stderr

	go func() {
		err := p.cmd.Run()
		if err != nil {
			klog.Infof("process(%s) exit(%v)", cmd, err)
		} else {
			klog.Infof("process exit(0)")
		}

	}()

	time.Sleep(time.Second)
	klog.V(3).Infof("%s running...", cmd)
}

// shouldWatchFileWithExtension returns true if the name of the file
// hash a suffix that should be watched.
func (p *watcher) shouldWatchFileWithExtension(name string) bool {
	for _, s := range p.fileExts {
		if strings.HasSuffix(name, s) {
			return true
		}
	}
	return false
}

// If a file is excluded
func (p *watcher) isExcluded(file string) bool {
	for _, p := range p.excludedPaths {
		absP, err := filepath.Abs(p)
		if err != nil {
			klog.Errorf("Cannot get absolute path of '%s'", p)
			continue
		}
		absFilePath, err := filepath.Abs(file)
		if err != nil {
			klog.Errorf("Cannot get absolute path of '%s'", file)
			break
		}
		if strings.HasPrefix(absFilePath, absP) {
			klog.V(4).Infof("'%s' is not being watched", file)
			return true
		}
	}
	return false
}

func (p *watcher) readAppDirectories(directory string) {
	fileInfos, err := ioutil.ReadDir(directory)
	if err != nil {
		return
	}

	useDirectory := false
	for _, fileInfo := range fileInfos {
		if p.isExcluded(path.Join(directory, fileInfo.Name())) {
			continue
		}

		if fileInfo.IsDir() && fileInfo.Name()[0] != '.' {
			p.readAppDirectories(directory + "/" + fileInfo.Name())
			continue
		}

		if useDirectory {
			continue
		}

		if p.shouldWatchFileWithExtension(fileInfo.Name()) {
			p.paths = append(p.paths, directory)
			useDirectory = true
		}
	}
}

// GetFileModTime returns unix timestamp of `os.File.ModTime` for the given path.
func GetFileModTime(path string) int64 {
	path = strings.Replace(path, "\\", "/", -1)
	f, err := os.Open(path)
	if err != nil {
		//klog.Errorf("Failed to open file on '%s': %s", path, err)
		return time.Now().Unix()
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		klog.Errorf("Failed to get file stats: %s", err)
		return time.Now().Unix()
	}
	return fi.ModTime().Unix()
}

func main() {
	logs.InitLogs()

	var cf config
	var rootCmd = &cobra.Command{
		Use:   "watcher",
		Short: "watcher is a tool which watch files change and execute some command",
		RunE: func(cmd *cobra.Command, args []string) error {
			return watch(&cf)
		},
	}

	fs := rootCmd.PersistentFlags()

	fs.VarP(&cf.extraPaths, "list", "i", "list paths to include extra.")
	fs.VarP(&cf.excludedPaths, "exclude", "e", "List of paths to exclude.(default [vendor])")
	fs.VarP(&cf.fileExts, "file", "f", "List of file extension(default [.go])")
	fs.Int64VarP(&cf.delay, "delay", "d", 500, "delay time when recv fs notify(Millisecond)")
	fs.StringVar(&cf.cmd1, "c1", "make", "run this cmd(c1) when recv inotify event")
	fs.StringVar(&cf.cmd2, "c2", "make -s devrun", "invoke the cmd(c2) output when c1 is successfully executed")

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func watch(cf *config) error {

	watcher, err := NewWatcher(cf)
	if err != nil {
		return err
	}

	if done, err := watcher.Do(); err != nil {
		return err
	} else {
		return <-done
	}
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

type StrFlags []string

func (s *StrFlags) String() string {
	return fmt.Sprintf("%s", *s)
}

func (s *StrFlags) Set(value string) error {
	*s = append(*s, value)
	return nil
}

func (s *StrFlags) Type() string {
	return "strflags"
}

func command(s string) *exec.Cmd {
	c := strings.Fields(s)
	return exec.Command(c[0], c[1:]...)
}
