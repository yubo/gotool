// Copyright 2020 yubo. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yubo/golib/staging/logs"
)

// usage: ORIG_DIR=/a CUR_DIR=/b vimdiff src/main.go
// -> vim -d /a/src/main.go /b/src/main.go

type config struct {
	origDir string
	curDir  string
}

func isExist(file string) bool {
	_, err := os.Stat(file)
	if err == nil {
		return true
	}

	if os.IsNotExist(err) {
		return false
	}
	return false
}

func (c *config) Validate() (err error) {
	c.origDir, err = filepath.Abs(c.origDir)
	if err != nil {
		return err
	}
	if !isExist(c.origDir) {
		return fmt.Errorf("orig dir %s dose not exist", c.origDir)
	}
	c.curDir, err = filepath.Abs(c.curDir)
	if err != nil {
		return err
	}
	if !isExist(c.curDir) {
		return fmt.Errorf("cur dir %s dose not exist", c.curDir)
	}
	return nil
}

func main() {
	logs.InitLogs()

	cf := &config{
		origDir: os.Getenv("ORIG_DIR"),
		curDir:  os.Getenv("CUR_DIR"),
	}

	var rootCmd = &cobra.Command{
		Use:   "vimdiff",
		Short: "vimdiff is a tool which compares two dir files",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("unable to get dst file %#v", args)
			}
			return vimdiff(cf, args[0])
		},
	}

	fs := rootCmd.PersistentFlags()
	fs.StringVar(&cf.origDir, "orig", cf.origDir, "orig dir")
	fs.StringVar(&cf.curDir, "cur", cf.curDir, "cur dir")

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func vimdiff(cf *config, file string) error {
	file, err := filepath.Abs(file)
	if err != nil {
		return err
	}
	if !isExist(file) {
		return fmt.Errorf("file %s dose not exist", file)
	}

	if err := cf.Validate(); err != nil {
		return err
	}

	if !strings.HasPrefix(file, cf.curDir) {
		return fmt.Errorf("file %s not under cur dir %s", file, cf.curDir)
	}

	file2 := filepath.Join(cf.origDir, strings.TrimLeft(file, cf.curDir))

	cmd := exec.Command("vim", "-d", file, file2)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
