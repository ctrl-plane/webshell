package instance

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/ctrl-plane/webshell/pkg/shell"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

func New(
	u string, 
	dialer websocket.Dialer,
	headers http.Header,
) (*WebsocketClient, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}

	identifer := uuid.New().String()
	headers.Add("X-Hostname", hostname)
	headers.Add("X-Runtime", runtime.GOOS)
	headers.Add("X-Arch", runtime.GOARCH)
	headers.Add("X-Identifier", identifer)

	wc := &WebsocketClient{
		shells:    make(map[string]*shell.Shell),
		identifer: identifer,

		url:       u,
		headers:   headers,
		dialer:   dialer,
	}
	wc.reconnect()
	return wc, nil
}

type EventType string

const (
	MessageTypeShellData   EventType = "shell/data"
	MessageTypeShellCreate EventType = "shell/create"
)

type Event struct {
	Type EventType `json:"type"`
}

type EventWebsocketShellData struct {
	Type       EventType `json:"type"`
	InstanceID string    `json:"instanceId"`
	ClientID   string    `json:"clientId"`
	Data       string    `json:"data"`
}

type EventWebsocketShellCreate struct {
	Type       EventType `json:"type"`
	InstanceID string    `json:"instanceId"`
	ClientID   string    `json:"clientId"`
}

type WebsocketClient struct {
	url      string
	headers  http.Header
	dialer	 websocket.Dialer

	identifer string
	conn      *websocket.Conn
	shells    map[string]*shell.Shell
	mu        sync.Mutex
}

func (i *WebsocketClient) reconnect() {
	for {

		conn, _, err := websocket.DefaultDialer.Dial(i.url, i.headers)
		if err != nil {
			log.Printf("Failed to connect to %s: %v. Retrying in 1 second...", i.url, err)
			time.Sleep(1 * time.Millisecond)
			continue
		}
		i.conn = conn
		conn.SetCloseHandler(func(_ int, __ string) error {
			print("")
			i.Close()
			go i.reconnect()
			return nil
		})

		log.Printf("Connected to %s, using ID %s\n", i.url, i.identifer)
		break
	}
}

func (i *WebsocketClient) NewShell(clientId string) (string, *shell.Shell, error) {

	sh, err := shell.New()
	if err != nil {
		return "", nil, err
	}

	go func() {
		dataChan := sh.Read()
		defer close(dataChan)
		for data := range dataChan {
			msg := EventWebsocketShellData{
				Type:       MessageTypeShellData,
				InstanceID: i.identifer,
				ClientID:   clientId,
				Data:       string(data),
			}

			if err := i.conn.WriteJSON(msg); err != nil {
				log.Println("Failed to write to WebSocket:", err)
				break
			}
		}
	}()

	i.addShell(clientId, sh)
	return clientId, sh, nil
}

func (i *WebsocketClient) processShellCreateEvent(event *EventWebsocketShellCreate) {
	if event.InstanceID != i.identifer {
		log.Printf("Instance ID mismatch: %s != %s\n", event.InstanceID, i.identifer)
		return
	}

	_, exists := i.GetShell(event.ClientID)
	if exists {
		return
	}
	_, _, err := i.NewShell(event.ClientID)
	if err != nil {
		log.Println("Failed to create shell:", err)
		return
	}

	log.Printf("New shell created for %s\n", event.ClientID)
}

func (i *WebsocketClient) processShellDataEvent(event *EventWebsocketShellData) {
	if event.InstanceID != i.identifer {
		log.Printf("Instance ID mismatch: %s != %s\n", event.InstanceID, i.identifer)
		return
	}

	sh, ok := i.GetShell(event.ClientID)
	if !ok {
		log.Printf("Shell found for client %s\n", event.ClientID)
		return
	}

	if _, err := sh.Write([]byte(event.Data)); err != nil {
		log.Println("Failed to write to pty:", err)
	}
}

func (i *WebsocketClient) ReadInput() {
	for {
		if i.conn == nil {
			time.Sleep(250 * time.Millisecond)
			continue
		}

		var rawMessage json.RawMessage
		if err := i.conn.ReadJSON(&rawMessage); err != nil {
			log.Println("Client failed to read from WebSocket:", err)
			i.conn.Close()
			i.conn = nil
			i.reconnect()
			continue
		}

		var event Event
		if err := json.Unmarshal(rawMessage, &event); err != nil {
			log.Println("Failed to unmarshal message type:", err)
			break
		}

		switch event.Type {
		case MessageTypeShellCreate:
			var msg EventWebsocketShellCreate
			if err := json.Unmarshal(rawMessage, &msg); err != nil {
				log.Println("Failed to unmarshal message type:", err)
				break
			}
			go i.processShellCreateEvent(&msg)
		case MessageTypeShellData:
			var msg EventWebsocketShellData
			if err := json.Unmarshal(rawMessage, &msg); err != nil {
				log.Println("Failed to unmarshal message type:", err)
				break
			}
			go i.processShellDataEvent(&msg)
		}
	}
}

func (i *WebsocketClient) addShell(id string, sh *shell.Shell) {
	i.mu.Lock()
	i.shells[id] = sh
	i.mu.Unlock()
}

func (i *WebsocketClient) RemoveShell(id string) {
	i.mu.Lock()
	s, _ := i.GetShell(id)
	s.Close()
	delete(i.shells, id)
	i.mu.Unlock()
}

func (i *WebsocketClient) GetShell(id string) (*shell.Shell, bool) {
	i.mu.Lock()
	ws, ok := i.shells[id]
	i.mu.Unlock()
	return ws, ok
}

func (i *WebsocketClient) Close() {
	i.mu.Lock()
	for _, ws := range i.shells {
		ws.Close()
	}
	i.conn.Close()
	i.mu.Unlock()
}
