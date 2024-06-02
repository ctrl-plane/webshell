package shell

import (
	"log"
	"os"
	"os/exec"

	"github.com/creack/pty"
)

type Shell struct {
	pty *os.File
}

func createPtyShell() (*os.File, error) {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/bash"
	}
	cmd := exec.Command(shell)
	return pty.Start(cmd)
}

func New() (*Shell, error) {
	pty, err := createPtyShell()
	if err != nil {
		return nil, err
	}
	return &Shell{pty: pty}, nil
}

func (s *Shell) Read() chan []byte {
	ch := make(chan []byte)

	go func() {
		defer close(ch)
		buf := make([]byte, 1024)
		for {
			n, err := s.pty.Read(buf)
			if err != nil {
				log.Println("Error reading from PTY:", err)
				return
			}
			ch <- append([]byte{}, buf[:n]...) // Send a copy of the buffer slice
		}
	}()

	return ch
}

func (s *Shell) Write(data []byte) (int, error) {
	return s.pty.Write(data)
}

func (s *Shell) Close() error {
	return s.pty.Close()
}
