package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/ctrl-plane/webshell/pkg/instance"
	"github.com/gorilla/websocket"
)

type headersFlag map[string]string

func (h headersFlag) String() string {
	return fmt.Sprintf("%v", map[string]string(h))
}

func (h headersFlag) Set(value string) error {
	parts := strings.SplitN(value, ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid header format, expected key:value, got %s", value)
	}
	h[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
	return nil
}

func main() {
	url := flag.String("url", "wss://localhost:8080/echo", "WebSocket server URL")
	insecure := flag.Bool("insecure", false, "Skip TLS certificate verification")
	headers := headersFlag{}
	flag.Var(headers, "header", "Custom header in key:value format (can be specified multiple times)")

	flag.Parse()

	tlsConfig := &tls.Config{
		InsecureSkipVerify: *insecure,
	}

	dialer := websocket.Dialer{
		TLSClientConfig: tlsConfig,
	}

	header := http.Header{}
	for key, value := range headers {
		header.Set(key, value)
	}

	client, err := instance.New(*url, dialer, header)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	go client.ReadInput()

	select {}
}
