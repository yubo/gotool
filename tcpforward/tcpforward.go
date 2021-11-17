// Copyright 2021 yubo. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package main

import (
	"fmt"
	"io"
	"net"
	"os"
	"sync"

	"github.com/spf13/cobra"
)

// usage: tcpforward -l 0.0.0.0:8022 -r 127.0.0.1:22

type config struct {
	left  string
	right string
}

func (c *config) Validate() (err error) {
	return nil
}

func main() {
	cf := &config{
		left:  "0.0.0.0:8022",
		right: "127.0.0.1:22",
	}

	cmd := &cobra.Command{
		Use: "portforward",
		RunE: func(cmd *cobra.Command, args []string) error {
			return forward(cf)
		},
	}

	fs := cmd.PersistentFlags()
	fs.StringVarP(&cf.left, "left", "l", cf.left, "left addr")
	fs.StringVarP(&cf.right, "right", "r", cf.right, "right addr")

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}

}

func forward(cf *config) error {
	ln, err := net.Listen("tcp", cf.left)
	if err != nil {
		return fmt.Errorf("listen %s err %s", cf.left, err)
	}
	defer ln.Close()

	fmt.Printf("listening on %s\n", cf.left)

	for {
		conn, err := ln.Accept()
		if err != nil {
			return fmt.Errorf("accept %s err %s", cf.left, err)
		}
		go handleConnection(conn, cf.right)
	}
}

func handleConnection(left net.Conn, rAddr string) {
	right, err := net.Dial("tcp", rAddr)
	if err != nil {
		fmt.Printf("dial %s err %s", rAddr, err)
		return
	}

	connInfo := fmt.Sprintf("%s -> %s", left.RemoteAddr(), rAddr)

	fmt.Printf("%-12s %s\n", "connected", connInfo)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		io.Copy(right, left)
		wg.Add(-1)
	}()

	wg.Add(1)
	go func() {
		io.Copy(left, right)
		wg.Add(-1)
	}()

	wg.Wait()
	fmt.Printf("%-12s %s\n", "closed", connInfo)
}
