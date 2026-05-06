package sshx

import (
	"context"
	"fmt"
	"net"
	"os"
	"regexp"
	"strings"

	"golang.org/x/crypto/ssh"
)

type Client struct {
	c *ssh.Client
}

// Dial parses host as user@ip[:port] and opens an SSH connection using the key at keyPath.
// TODO(phase 5): replace InsecureIgnoreHostKey with knownhosts verification.
func Dial(host, keyPath string) (*Client, error) {
	user, addr, err := parseHost(host)
	if err != nil {
		return nil, err
	}
	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("read ssh key %s: %w", keyPath, err)
	}
	signer, err := ssh.ParsePrivateKey(keyData)
	if err != nil {
		return nil, fmt.Errorf("parse ssh key %s: %w", keyPath, err)
	}
	cfg := &ssh.ClientConfig{
		User:            user,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // TODO(phase 5): knownhosts
	}
	c, err := ssh.Dial("tcp", addr, cfg)
	if err != nil {
		return nil, fmt.Errorf("ssh dial %s: %w", addr, err)
	}
	return &Client{c: c}, nil
}

func (c *Client) Close() error {
	return c.c.Close()
}

// Run executes cmd remotely, streaming stdout/stderr to the local process.
func (c *Client) Run(ctx context.Context, cmd string) error {
	sess, err := c.c.NewSession()
	if err != nil {
		return fmt.Errorf("new session: %w", err)
	}
	defer sess.Close()
	sess.Stdout = os.Stdout
	sess.Stderr = os.Stderr
	return runWithCtx(ctx, sess, cmd)
}

// Output runs cmd and returns combined stdout+stderr.
func (c *Client) Output(ctx context.Context, cmd string) (string, error) {
	sess, err := c.c.NewSession()
	if err != nil {
		return "", fmt.Errorf("new session: %w", err)
	}
	defer sess.Close()
	var buf strings.Builder
	sess.Stdout = &buf
	sess.Stderr = &buf
	err = runWithCtx(ctx, sess, cmd)
	return buf.String(), err
}

func runWithCtx(ctx context.Context, sess *ssh.Session, cmd string) error {
	if err := sess.Start(cmd); err != nil {
		return fmt.Errorf("start %q: %w", redactCmd(cmd), err)
	}
	done := make(chan error, 1)
	go func() { done <- sess.Wait() }()
	select {
	case <-ctx.Done():
		_ = sess.Signal(ssh.SIGKILL)
		return ctx.Err()
	case err := <-done:
		if err != nil {
			return fmt.Errorf("remote %q: %w", redactCmd(cmd), err)
		}
		return nil
	}
}

func parseHost(h string) (user, addr string, err error) {
	at := strings.Index(h, "@")
	if at < 0 {
		return "", "", fmt.Errorf("ssh host must be user@host[:port], got %q", h)
	}
	user = h[:at]
	rest := h[at+1:]
	if user == "" || rest == "" {
		return "", "", fmt.Errorf("ssh host must be user@host[:port], got %q", h)
	}
	if _, _, e := net.SplitHostPort(rest); e != nil {
		rest = net.JoinHostPort(rest, "22")
	}
	return user, rest, nil
}

var tokenURLRe = regexp.MustCompile(`https://x-access-token:[^@]+@`)

func redactCmd(s string) string {
	return tokenURLRe.ReplaceAllString(s, "https://x-access-token:***@")
}
