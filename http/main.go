// go get github.com/yubo/gotool/http

package main

import (
	"flag"
	"log"
	"net/http"
)

func main() {
	port := flag.String("p", "8000", "server mode")
	dir := flag.String("d", ".", "server mode")

	fs := http.FileServer(http.Dir(*dir))
	http.Handle("/", fs)

	log.Printf("Listening %s ...", *port)
	http.ListenAndServe(":"+*port, nil)
}
