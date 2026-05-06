package cloudflare

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

type AccessApp struct {
	ID              string `json:"id,omitempty"`
	Name            string `json:"name"`
	Domain          string `json:"domain"`
	Type            string `json:"type"`
	SessionDuration string `json:"session_duration,omitempty"`
}

// FindAccessApp returns the first Access app whose domain matches in the given
// account, or nil if none exists.
func (c *Client) FindAccessApp(ctx context.Context, accountID, domain string) (*AccessApp, error) {
	q := url.Values{}
	q.Set("domain", domain)
	path := fmt.Sprintf("/accounts/%s/access/apps?%s", url.PathEscape(accountID), q.Encode())

	result, err := c.do(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var apps []AccessApp
	if err := json.Unmarshal(result, &apps); err != nil {
		return nil, fmt.Errorf("cloudflare: decode access apps: %w", err)
	}
	for i := range apps {
		if apps[i].Domain == domain {
			return &apps[i], nil
		}
	}
	if len(apps) == 0 {
		return nil, nil
	}
	return &apps[0], nil
}

// CreateAccessApp creates a self_hosted Access app scoped to domain and returns
// the created app (including its assigned ID).
func (c *Client) CreateAccessApp(ctx context.Context, accountID, name, domain string) (*AccessApp, error) {
	body := AccessApp{
		Name:   name,
		Domain: domain,
		Type:   "self_hosted",
	}
	path := fmt.Sprintf("/accounts/%s/access/apps", url.PathEscape(accountID))
	result, err := c.do(ctx, "POST", path, body)
	if err != nil {
		return nil, err
	}
	var app AccessApp
	if err := json.Unmarshal(result, &app); err != nil {
		return nil, fmt.Errorf("cloudflare: decode access app: %w", err)
	}
	return &app, nil
}

// UpdateAccessApp replaces the app at appID with a self_hosted app for domain.
func (c *Client) UpdateAccessApp(ctx context.Context, accountID, appID, name, domain string) error {
	body := AccessApp{
		Name:   name,
		Domain: domain,
		Type:   "self_hosted",
	}
	path := fmt.Sprintf("/accounts/%s/access/apps/%s", url.PathEscape(accountID), url.PathEscape(appID))
	_, err := c.do(ctx, "PUT", path, body)
	return err
}

// AttachAccessPolicy attaches a pre-existing policy to the given app. If the
// policy is already attached, the operation is treated as success.
func (c *Client) AttachAccessPolicy(ctx context.Context, accountID, appID, policyID string) error {
	body := map[string]string{"id": policyID}
	path := fmt.Sprintf("/accounts/%s/access/apps/%s/policies", url.PathEscape(accountID), url.PathEscape(appID))
	_, err := c.do(ctx, "POST", path, body)
	if err != nil && strings.Contains(strings.ToLower(err.Error()), "already") {
		return nil
	}
	return err
}
