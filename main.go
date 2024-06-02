package main

import (
	"flag"
	"log"
	"net/url"

	"github.com/ctrl-plane/webshell/pkg/server"
)

var addr = flag.String("addr", "localhost:3000", "ws service address")

func main() {
	flag.Parse()
	log.SetFlags(0)

	u := url.URL{Scheme: "ws", Host: *addr, Path: "/api/ws"}
	client, err := New(u)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	go client.ReadInput()
	go server.Serve()

	select {}
}
