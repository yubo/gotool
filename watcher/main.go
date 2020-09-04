// Copyright 2015-2020 yubo. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/yubo/golib/logs"
)

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
