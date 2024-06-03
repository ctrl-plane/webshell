package main

import (
	"flag"
	"log"
	"net/url"

	"github.com/ctrl-plane/webshell/pkg/instance"
	"github.com/ctrl-plane/webshell/pkg/server"
)

var addr = flag.String("addr", "localhost:4000", "ws service address")

func main() {
	flag.Parse()
	u := url.URL{Scheme: "ws", Host: *addr, Path: "/route"}
	client, err := instance.New(u)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	go client.ReadInput()
	go server.Serve()

	select {}
}
