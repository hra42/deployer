package cloudflare

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

type AccessApp struct {
	ID              string   `json:"id,omitempty"`
	Name            string   `json:"name"`
	Domain          string   `json:"domain"`
	Type            string   `json:"type"`
	SessionDuration string   `json:"session_duration,omitempty"`
	Policies        []string `json:"policies,omitempty"`
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

// CreateAccessApp creates a self_hosted Access app scoped to domain, attached
// to the given reusable policy IDs, and returns the created app.
func (c *Client) CreateAccessApp(ctx context.Context, accountID, name, domain string, policyIDs []string) (*AccessApp, error) {
	body := AccessApp{
		Name:     name,
		Domain:   domain,
		Type:     "self_hosted",
		Policies: policyIDs,
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

// UpdateAccessApp replaces the app at appID with a self_hosted app for domain,
// attached to the given reusable policy IDs. The Cloudflare API treats the
// policies array as authoritative — any policy not listed is detached.
func (c *Client) UpdateAccessApp(ctx context.Context, accountID, appID, name, domain string, policyIDs []string) error {
	body := AccessApp{
		Name:     name,
		Domain:   domain,
		Type:     "self_hosted",
		Policies: policyIDs,
	}
	path := fmt.Sprintf("/accounts/%s/access/apps/%s", url.PathEscape(accountID), url.PathEscape(appID))
	_, err := c.do(ctx, "PUT", path, body)
	return err
}
