// Copyright 2015 yubo. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//
// go install github.com/yubo/gotool/httpd
package main

import (
	"flag"
	"fmt"
	"net/http"
	"net/http/httputil"
)

func main() {
	port := flag.String("p", "8000", "server mode")
	dir := flag.String("d", "", "server mode")

	flag.Parse()

	if d := *dir; d != "" {
		fs := http.FileServer(http.Dir(d))
		http.Handle("/", fs)
	} else {
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			b, _ := httputil.DumpRequest(r, true)
			fmt.Printf("%s\n", string(b))

		})
	}

	fmt.Printf("Listening %s ...\n", *port)
	http.ListenAndServe(":"+*port, nil)
}
