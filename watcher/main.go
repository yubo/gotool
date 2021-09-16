// Copyright 2015-2020 yubo. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/yubo/golib/cli/globalflag"
	"github.com/yubo/golib/configer"
	"github.com/yubo/golib/logs"
)

func main() {
	logs.InitLogs()
	defer logs.FlushLogs()

	if err := newRootCmd().Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	var cf config
	cmd := &cobra.Command{
		Use:   "watcher",
		Short: "watcher is a tool which watch files change and execute some command",
		RunE: func(cmd *cobra.Command, args []string) error {
			return watch(&cf)
		},
	}

	configer.AddFlags(cmd.Flags(), &cf)
	globalflag.AddGlobalFlags(cmd.Flags(), "watcher")
	return cmd
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
