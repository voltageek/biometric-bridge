// Package tray provides the system tray GUI for the Biometric Bridge.
package tray

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

const (
	singleInstancePort = 17070
	appName            = "kinetic-vault"
)

// InstanceLocker provides cross-platform single-instance enforcement using a TCP socket.
type InstanceLocker struct {
	listener net.Listener
	mu       sync.Mutex
	serverCh chan string
}

// NewInstanceLocker creates a new instance locker.
func NewInstanceLocker() *InstanceLocker {
	return &InstanceLocker{
		serverCh: make(chan string, 1),
	}
}

// TryLock attempts to acquire the lock by binding to a localhost TCP port.
// Returns true if lock acquired (no other instance running), false otherwise.
func (l *InstanceLocker) TryLock() bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	addr := fmt.Sprintf("127.0.0.1:%d", singleInstancePort)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		// Port is in use, another instance is running
		return false
	}

	l.listener = listener

	// Start server to handle commands from other instances
	go l.runServer()

	return true
}

// Unlock releases the lock by closing the listener.
func (l *InstanceLocker) Unlock() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.listener != nil {
		return l.listener.Close()
	}
	return nil
}

// BringToFront signals the existing instance to show its window.
// This should be called when TryLock returns false.
func (l *InstanceLocker) BringToFront() error {
	addr := fmt.Sprintf("127.0.0.1:%d", singleInstancePort)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("connect to existing instance: %w", err)
	}
	defer conn.Close()

	// Send SHOW command
	_, err = fmt.Fprintln(conn, "SHOW")
	if err != nil {
		return fmt.Errorf("send command: %w", err)
	}

	return nil
}

// WaitForCommand blocks until a command is received from another instance.
// Returns the command string (e.g., "SHOW") or an empty string if the server stops.
func (l *InstanceLocker) WaitForCommand() string {
	select {
	case cmd := <-l.serverCh:
		return cmd
	case <-time.After(100 * time.Millisecond):
		return ""
	}
}

// runServer handles incoming connections from other instances.
func (l *InstanceLocker) runServer() {
	for {
		conn, err := l.listener.Accept()
		if err != nil {
			// Listener closed
			return
		}

		go l.handleConnection(conn)
	}
}

// handleConnection processes a single client connection.
func (l *InstanceLocker) handleConnection(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)
	line, err := reader.ReadString('\n')
	if err != nil {
		return
	}

	// Trim newline and process command
	cmd := line[:len(line)-1]
	if cmd == "SHOW" {
		select {
		case l.serverCh <- cmd:
		default:
		}
	}
}

// IsAnotherInstanceRunning checks if another instance is running without acquiring the lock.
func IsAnotherInstanceRunning() bool {
	addr := fmt.Sprintf("127.0.0.1:%d", singleInstancePort)
	conn, err := net.DialTimeout("tcp", addr, 1*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// GetPIDFilePath returns the path to the PID file for this application.
func GetPIDFilePath() string {
	if runtime.GOOS == "windows" {
		return filepath.Join(os.TempDir(), appName+".pid")
	}
	return filepath.Join("/tmp", appName+".pid")
}
