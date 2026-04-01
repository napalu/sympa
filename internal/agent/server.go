package agent

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

type entry struct {
	passphrase []byte
	expiresAt  time.Time
}

// Server is the passphrase caching daemon.
type Server struct {
	mu       sync.Mutex
	cache    map[string]*entry
	ttl      time.Duration
	listener net.Listener
	done     chan struct{}
}

// NewServer creates a new agent server with the given cache TTL.
func NewServer(ttl time.Duration) *Server {
	return &Server{
		cache: make(map[string]*entry),
		ttl:   ttl,
		done:  make(chan struct{}),
	}
}

// Run starts the agent, listening on the given Unix socket path. Blocks until Shutdown.
func (s *Server) Run(socketPath string) error {
	harden()

	dir := filepath.Dir(socketPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("creating socket directory: %w", err)
	}

	// Check if an agent is already running
	if info, err := os.Stat(socketPath); err == nil {
		if info.Mode()&os.ModeSocket != 0 {
			if stat, ok := info.Sys().(*syscall.Stat_t); ok && stat.Uid == uint32(os.Getuid()) {
				if conn, err := net.DialTimeout("unix", socketPath, time.Second); err == nil {
					conn.Close()
					return fmt.Errorf("agent already running (socket %s)", socketPath)
				}
			}
		}
		// Not a valid socket, wrong owner, or not responding — remove it
	}
	os.Remove(socketPath)

	ln, err := net.Listen("unix", socketPath)
	if err != nil {
		return fmt.Errorf("listening on %s: %w", socketPath, err)
	}
	s.listener = ln

	if err := os.Chmod(socketPath, 0600); err != nil {
		ln.Close()
		os.Remove(socketPath)
		return fmt.Errorf("securing socket: %w", err)
	}

	// Start reaper goroutine
	go s.reaper()

	// Ignore SIGINT — the agent is a daemon and must survive terminal Ctrl+C.
	signal.Ignore(syscall.SIGINT)

	// Shut down cleanly on SIGTERM (or explicit SHUTDOWN command).
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM)
	go func() {
		select {
		case <-sigCh:
			s.Shutdown()
		case <-s.done:
		}
		signal.Stop(sigCh)
	}()

	fmt.Fprintf(os.Stderr, "Agent started (socket %s, TTL %s)\n", socketPath, s.ttl)

	// Accept loop
	for {
		conn, err := ln.Accept()
		if err != nil {
			select {
			case <-s.done:
				return nil
			default:
				continue
			}
		}
		go s.handleConn(conn)
	}
}

// Shutdown stops the agent, zeroes all cached passphrases, and removes the socket.
func (s *Server) Shutdown() {
	select {
	case <-s.done:
		return // already shut down
	default:
		close(s.done)
	}

	if s.listener != nil {
		s.listener.Close()
	}

	s.mu.Lock()
	s.clearLocked()
	s.mu.Unlock()

	// Remove socket file
	if s.listener != nil {
		os.Remove(s.listener.Addr().String())
	}
}

func (s *Server) handleConn(conn net.Conn) {
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(5 * time.Second))

	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 0, 64*1024), 64*1024)

	if !scanner.Scan() {
		return
	}
	line := scanner.Text()

	parts := strings.SplitN(line, " ", 3)
	verb := strings.ToUpper(parts[0])

	var response string
	switch verb {
	case "PING":
		response = "PONG"

	case "GET":
		if len(parts) < 2 {
			response = "ERR missing path"
			break
		}
		path, err := url.PathUnescape(parts[1])
		if err != nil {
			response = "ERR invalid path"
			break
		}
		s.mu.Lock()
		if e, ok := s.cache[path]; ok && time.Now().Before(e.expiresAt) {
			response = base64.StdEncoding.EncodeToString(e.passphrase)
			e.expiresAt = time.Now().Add(s.ttl) // refresh TTL on access
		} else {
			if ok {
				// expired — clean up
				zeroBytes(e.passphrase)
				delete(s.cache, path)
			}
			response = "MISS"
		}
		s.mu.Unlock()

	case "SET":
		if len(parts) < 3 {
			response = "ERR missing path or passphrase"
			break
		}
		path, err := url.PathUnescape(parts[1])
		if err != nil {
			response = "ERR invalid path"
			break
		}
		passBytes, err := base64.StdEncoding.DecodeString(parts[2])
		if err != nil {
			response = "ERR invalid base64"
			break
		}
		s.mu.Lock()
		if old, ok := s.cache[path]; ok {
			zeroBytes(old.passphrase)
		}
		stored := make([]byte, len(passBytes))
		copy(stored, passBytes)
		zeroBytes(passBytes)
		s.cache[path] = &entry{
			passphrase: stored,
			expiresAt:  time.Now().Add(s.ttl),
		}
		s.mu.Unlock()
		response = "OK"

	case "FORGET":
		if len(parts) < 2 {
			response = "ERR missing path"
			break
		}
		path, err := url.PathUnescape(parts[1])
		if err != nil {
			response = "ERR invalid path"
			break
		}
		s.mu.Lock()
		if e, ok := s.cache[path]; ok {
			zeroBytes(e.passphrase)
			delete(s.cache, path)
		}
		s.mu.Unlock()
		response = "OK"

	case "CLEAR":
		s.mu.Lock()
		s.clearLocked()
		s.mu.Unlock()
		response = "OK"

	case "SHUTDOWN":
		response = "BYE"
		fmt.Fprintln(conn, response)
		go s.Shutdown()
		return

	default:
		response = "ERR unknown command"
	}

	fmt.Fprintln(conn, response)
}

func (s *Server) clearLocked() {
	for k, e := range s.cache {
		zeroBytes(e.passphrase)
		delete(s.cache, k)
	}
}

func (s *Server) reaper() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.done:
			return
		case <-ticker.C:
			s.mu.Lock()
			now := time.Now()
			for k, e := range s.cache {
				if now.After(e.expiresAt) {
					zeroBytes(e.passphrase)
					delete(s.cache, k)
				}
			}
			s.mu.Unlock()
		}
	}
}

func zeroBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
}
