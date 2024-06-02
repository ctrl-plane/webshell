package main

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/ctrl-plane/webshell/pkg/shell"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

func New(u url.URL) (*WebsocketClient, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}

	headers := http.Header{
        "X-Hostname": {hostname},
		"X-Runtime": {runtime.GOOS},
		"X-Arch": {runtime.GOARCH},
		"X-Instance-ID": {InstanceID()},
    }
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), headers)
	if err != nil {
		return nil, err
	}
	wc := &WebsocketClient{
		conn:   conn,
		shells: make(map[string]*shell.Shell),
	}

	conn.SetCloseHandler(func(_ int, __ string) error {
		wc.Close()
		return nil
	})

	log.Printf("Connected to %s, using ID %s\n", u.String(), InstanceID())

	return wc, nil
}

type EventType string

const (
	MessageTypeShellData   EventType = "shell-data"
	MessageTypeShellCreate EventType = "shell-create"
)

type Event struct {
	Type EventType `json:"type"`
}

type EventWebsocketShellData struct {
	Type       EventType `json:"type"`
	InstanceID string    `json:"instanceId"`
	ShellID    string    `json:"shellId"`
	Data       string    `json:"data"`
}

type EventWebsocketShellCreate struct {
	Type       EventType `json:"type"`
	From       string    `json:"from"`
	InstanceID string    `json:"instanceId"`
}

type WebsocketClient struct {
	conn   *websocket.Conn
	shells map[string]*shell.Shell
	mu     sync.Mutex
}

func (i *WebsocketClient) NewShell() (string, *shell.Shell, error) {
	id := uuid.New().String()

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
				InstanceID: InstanceID(),
				ShellID:    id,
				Data:       string(data),
			}

			if err := i.conn.WriteJSON(msg); err != nil {
				log.Println("Failed to write to WebSocket:", err)
				break
			}
		}
	}()

	i.addShell(id, sh)
	return id, sh, nil
}

func (i *WebsocketClient) processShellCreateEvent(event *EventWebsocketShellCreate) {
	if event.InstanceID != InstanceID() {
		log.Printf("Instance ID mismatch: %s != %s\n", event.InstanceID, InstanceID())
		return
	}

	id, _, err := i.NewShell()
	if err != nil {
		log.Println("Failed to create shell:", err)
		return
	}

	log.Printf("Shell %s created\n", id)
}

func (i *WebsocketClient) processShellDataEvent(event *EventWebsocketShellData) {
	if event.InstanceID != InstanceID() {
		log.Printf("Instance ID mismatch: %s != %s\n", event.InstanceID, InstanceID())
		return
	}

	sh, ok := i.GetShell(event.ShellID)
	if !ok {
		log.Printf("Shell %s not found\n", event.ShellID)
		return
	}

	if _, err := sh.Write([]byte(event.Data)); err != nil {
		log.Println("Failed to write to pty:", err)
	}
}

func (i *WebsocketClient) ReadInput() {
	for {
		var rawMessage json.RawMessage
		if err := i.conn.ReadJSON(&rawMessage); err != nil {
			log.Println("Failed to read from WebSocket:", err)
			break
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

func InstanceID() string {
	clientIDFilePath, err := getClientIDFilePath()
	if err != nil {
		log.Fatal("Failed to get client ID file path:", err)
	}

	clientID, err := readClientID(clientIDFilePath)
	if err != nil {
		log.Println("Failed to read client ID:", err)
		clientID = uuid.New().String()
		if err := writeClientID(clientIDFilePath, clientID); err != nil {
			log.Fatal("Failed to write client ID:", err)
		}
	}

	return clientID
}

const (
	ConfigDir  = ".config/instance"
	ConfigFile = "client_id.txt"
)

func getClientIDFilePath() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}
	configPath := filepath.Join(usr.HomeDir, ConfigDir)
	if err := os.MkdirAll(configPath, 0755); err != nil {
		return "", err
	}
	return filepath.Join(configPath, ConfigFile), nil
}

func readClientID(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func writeClientID(filePath, id string) error {
	return os.WriteFile(filePath, []byte(id), 0644)
}
