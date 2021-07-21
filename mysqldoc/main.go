// Copyright 2020 yubo. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/yubo/golib/staging/logs"
)

// usage: mysqldoc --dsn1="root:1234@tcp(localhost:3306)/src_db?charset=utf8" --dsn2="root:1234@tcp(localhost:3306)/src_db?charset=utf8"

type Config struct {
	dsn string
}

func main() {
	logs.InitLogs()

	cf := &Config{}
	var rootCmd = &cobra.Command{
		Use:   "mysqldoc",
		Short: "mysqldoc is a tool that generate MySQL database documents",
		RunE: func(cmd *cobra.Command, args []string) error {
			return mysqldoc(cf)
		},
	}

	fs := rootCmd.PersistentFlags()
	fs.StringVar(&cf.dsn, "dsn", "", "dsn e.g. root:1234@tcp(localhost:3306)/src_db?charset=utf8")

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func mysqldoc(cf *Config) error {
	p := &Doc{Config: cf}
	if err := p.conn(); err != nil {
		return err
	}
	defer p.close()

	if err := p.dbDoc(); err != nil {
		return err
	}

	return nil
}
