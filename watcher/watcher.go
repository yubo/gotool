// Copyright 2015-2020 yubo. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"k8s.io/klog/v2"
)

type config struct {
	IncludePaths  []string      `flag:"include,i" default:"." description:"list paths to include extra."`
	ExcludedPaths []string      `flag:"exclude" default:"vendor" description:"List of paths to exclude."`
	FileExts      []string      `flag:"file,f" default:".go" description:"List of file extension."`
	Delay         time.Duration `flag:"delay,d" default:"500ms" description:"delay time when recv fs notify(Millisecond)"`
	Cmd1          string        `flag:"c1" default:"make" description:"run this cmd(c1) when recv inotify event"`
	Cmd2          string        `flag:"c2" default:"make -s devrun" description:"invoke the cmd(c2) output when c1 is successfully executed"`
}

type watcher struct {
	*config
	*fsnotify.Watcher
	sync.Mutex

	cmd       *exec.Cmd
	eventTime map[string]int64
}

func NewWatcher(cf *config) (*watcher, error) {
	if len(cf.IncludePaths) > 1 {
		cf.IncludePaths = cf.IncludePaths[1:]
	}

	klog.Infof("include paths %v", cf.IncludePaths)
	klog.Infof("excludedPaths %v", cf.ExcludedPaths)
	klog.Infof("watch file exts %v", cf.FileExts)

	watcher := &watcher{
		config:    cf,
		eventTime: make(map[string]int64),
	}

	// expend currpath
	dir, _ := os.Getwd()
	watcher.readAppDirectories(dir)

	return watcher, nil

}

// NewWatcher starts an fsnotify Watcher on the specified paths
func (p *watcher) Do() (done <-chan error, err error) {
	if p.Watcher, err = fsnotify.NewWatcher(); err != nil {
		return nil, fmt.Errorf("Failed to create watcher: %s", err)
	}

	done = make(chan error, 0)
	buildEvent := make(chan *fsnotify.Event, 10)

	go func() {
		var ev *fsnotify.Event
		pending := false
		ticker := time.NewTicker(p.Delay)
		for {
			select {
			case e := <-buildEvent:
				pending = true
				ev = e
				ticker.Stop()
				ticker = time.NewTicker(p.Delay)
			case <-ticker.C:
				if pending {
					p.autoBuild(ev)
					pending = false
				}
			}
		}
	}()

	go func() {
		for {
			select {
			case e := <-p.Events:
				build := true

				if !p.shouldWatchFileWithExtension(e.Name) {
					continue
				}

				mt := GetFileModTime(e.Name)
				if t := p.eventTime[e.Name]; mt == t {
					klog.V(7).Info(e.String())
					build = false
				}

				p.eventTime[e.Name] = mt

				if build {
					klog.V(7).Infof("Event fired: %s", e)
					buildEvent <- &e
				}
			case err := <-p.Errors:
				klog.Warningf("Watcher error: %s", err.Error()) // No need to exit here
			}
		}
	}()

	klog.Info("Initializing watcher...")
	for _, path := range p.IncludePaths {
		klog.V(6).Infof("Watching: %s", path)
		if err := p.Add(path); err != nil {
			klog.Fatalf("Failed to watch directory: %s", err)
		}
	}
	p.autoBuild(&fsnotify.Event{Name: "first time", Op: fsnotify.Create})
	return
}

// autoBuild builds the specified set of files
func (p *watcher) autoBuild(e *fsnotify.Event) {
	p.Lock()
	defer p.Unlock()

	cmd := command(p.Cmd1)
	output, err := cmd.CombinedOutput()
	klog.Infof("---------- %s -------", e)
	if err != nil {
		klog.Error(string(output))
		klog.Error(err)
		return
	}

	klog.V(3).Info(string(output))
	klog.V(3).Info("Built Successfully!")
	p.restart()
}

// kill kills the running command process
func (p *watcher) kill() {
	defer func() {
		if e := recover(); e != nil {
			klog.Infof("Kill recover: %s", e)
		}
	}()
	if p.cmd != nil && p.cmd.Process != nil {
		klog.V(3).Infof("Process %d will be killing", p.cmd.Process.Pid)
		err := p.cmd.Process.Kill()
		if err != nil {
			klog.V(3).Infof("Error while killing cmd process: %s", err)
		}
	}
}

// restart kills the running command process and starts it again
func (p *watcher) restart() {
	p.kill()
	go p.start()
}

func (p *watcher) getCmd() string {
	cmd := command(p.Cmd2)
	output, err := cmd.Output()
	if err != nil {
		klog.Errorf("run %s err %s", p.Cmd2, err)
	}
	return strings.TrimSpace(strings.Split(string(output), "\n")[0])
}

// start starts the command process
func (p *watcher) start() {
	cmd := p.getCmd()
	if cmd == "" {
		klog.Infof("cmd is empty")
		return
	}

	p.cmd = command(cmd)
	p.cmd.Stdout = os.Stdout
	p.cmd.Stderr = os.Stderr

	if err := p.cmd.Start(); err != nil {
		klog.Infof("execute %s err %s", cmd, err)
		return
	}

	klog.V(3).Infof("Process %d execute %s", p.cmd.Process.Pid, cmd)

	go func() {
		if err := p.cmd.Wait(); err != nil {
			klog.Infof("Process %d exit %v", p.cmd.Process.Pid, err)
		} else {
			klog.Infof("process exit 0")
		}
	}()
}

// shouldWatchFileWithExtension returns true if the name of the file
// hash a suffix that should be watched.
func (p *watcher) shouldWatchFileWithExtension(name string) bool {
	for _, s := range p.FileExts {
		if strings.HasSuffix(name, s) {
			return true
		}
	}
	return false
}

// If a file is excluded
func (p *watcher) isExcluded(file string) bool {
	for _, p := range p.ExcludedPaths {
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
			p.IncludePaths = append(p.IncludePaths, directory)
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

func command(s string) *exec.Cmd {
	c := strings.Fields(s)
	return exec.Command(c[0], c[1:]...)
}
