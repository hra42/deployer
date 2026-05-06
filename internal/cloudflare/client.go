// Package cloudflare is a small client for the Cloudflare v4 REST API,
// implemented with net/http only — no SDK.
package cloudflare

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const defaultBaseURL = "https://api.cloudflare.com/client/v4"

type Client struct {
	token      string
	httpClient *http.Client
	baseURL    string
}

func New(token string) *Client {
	return newWithBase(token, defaultBaseURL)
}

func newWithBase(token, baseURL string) *Client {
	return &Client{
		token:      token,
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

type apiError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type envelope struct {
	Success bool            `json:"success"`
	Errors  []apiError      `json:"errors"`
	Result  json.RawMessage `json:"result"`
}

// do performs an authenticated request against the Cloudflare API and returns
// the raw "result" bytes from the standard envelope. body, if non-nil, is
// JSON-marshaled and sent as the request body.
func (c *Client) do(ctx context.Context, method, path string, body any) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("cloudflare: marshal request: %w", err)
		}
		reqBody = bytes.NewReader(buf)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("cloudflare: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cloudflare: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("cloudflare: read response: %w", err)
	}

	var env envelope
	if err := json.Unmarshal(respBytes, &env); err != nil {
		return nil, fmt.Errorf("cloudflare: decode response (status %d): %w", resp.StatusCode, err)
	}

	if !env.Success || resp.StatusCode >= 400 {
		msg := "request failed"
		if len(env.Errors) > 0 {
			msg = env.Errors[0].Message
		}
		return nil, fmt.Errorf("cloudflare: %s (status %d)", msg, resp.StatusCode)
	}

	return env.Result, nil
}
