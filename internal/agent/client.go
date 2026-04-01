package agent

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
)

// Client communicates with the passphrase caching agent.
type Client struct {
	socketPath string
}

// NewClient creates a new agent client using the default socket path.
func NewClient() *Client {
	return &Client{socketPath: SocketPath()}
}

// Ping checks if the agent is running.
func (c *Client) Ping() bool {
	resp, err := c.send("PING")
	return err == nil && resp == "PONG"
}

// Get retrieves a cached passphrase for the given secret path.
func (c *Client) Get(secretPath string) (string, bool) {
	resp, err := c.send(fmt.Sprintf("GET %s", url.PathEscape(secretPath)))
	if err != nil || resp == "MISS" || resp == "" {
		return "", false
	}
	passBytes, err := base64.StdEncoding.DecodeString(resp)
	if err != nil {
		return "", false
	}
	return string(passBytes), true
}

// Set stores a passphrase in the cache for the given secret path.
func (c *Client) Set(secretPath, passphrase string) {
	encoded := base64.StdEncoding.EncodeToString([]byte(passphrase))
	c.send(fmt.Sprintf("SET %s %s", url.PathEscape(secretPath), encoded))
}

// Forget removes a specific cached passphrase.
func (c *Client) Forget(secretPath string) {
	c.send(fmt.Sprintf("FORGET %s", url.PathEscape(secretPath)))
}

// Shutdown asks the agent to stop.
func (c *Client) Shutdown() error {
	resp, err := c.send("SHUTDOWN")
	if err != nil {
		return err
	}
	if resp != "BYE" {
		return fmt.Errorf("unexpected response: %s", resp)
	}
	return nil
}

// EnsureRunning auto-spawns the agent if it's not already running.
func (c *Client) EnsureRunning() error {
	if c.Ping() {
		return nil
	}

	// Create socket directory
	dir := filepath.Dir(c.socketPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("creating socket directory: %w", err)
	}

	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("finding executable: %w", err)
	}

	cmd := exec.Command(exePath, "agent", "start")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting agent: %w", err)
	}

	// Poll for readiness
	for i := 0; i < 20; i++ {
		time.Sleep(50 * time.Millisecond)
		if c.Ping() {
			return nil
		}
	}
	return fmt.Errorf("agent failed to start within 1 second")
}

func (c *Client) verifySocketOwnership() error {
	info, err := os.Stat(c.socketPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // socket doesn't exist yet; let dial fail naturally
		}
		return err
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return fmt.Errorf("cannot verify socket ownership")
	}
	if stat.Uid != uint32(os.Getuid()) {
		return fmt.Errorf("socket %s is not owned by current user", c.socketPath)
	}
	return nil
}

func (c *Client) send(request string) (string, error) {
	if err := c.verifySocketOwnership(); err != nil {
		return "", err
	}

	conn, err := net.DialTimeout("unix", c.socketPath, 2*time.Second)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(5 * time.Second))

	fmt.Fprintln(conn, request)

	scanner := bufio.NewScanner(conn)
	if scanner.Scan() {
		return scanner.Text(), nil
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", fmt.Errorf("no response")
}
