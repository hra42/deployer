package cloudflare

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

// AccessApp is the shape we read back from the Cloudflare API. The Policies
// field is deliberately a raw passthrough — on responses Cloudflare returns
// full policy objects, but on requests it expects an array of policy ID
// strings. Use accessAppRequest for writes.
type AccessApp struct {
	ID              string          `json:"id,omitempty"`
	Name            string          `json:"name"`
	Domain          string          `json:"domain"`
	Type            string          `json:"type"`
	SessionDuration string          `json:"session_duration,omitempty"`
	Policies        json.RawMessage `json:"policies,omitempty"`
}

// accessAppRequest is the body shape for Create/Update. Cloudflare expects
// policies as a flat array of policy ID strings here.
type accessAppRequest struct {
	Name                   string   `json:"name"`
	Domain                 string   `json:"domain"`
	Type                   string   `json:"type"`
	Policies               []string `json:"policies,omitempty"`
	AllowedIdPs            []string `json:"allowed_idps,omitempty"`
	AutoRedirectToIdentity bool     `json:"auto_redirect_to_identity,omitempty"`
}

// IdentityProvider is a stripped-down view of a Cloudflare Access IdP.
type IdentityProvider struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

// ListIdentityProviders returns all IdPs configured on the account, including
// the implicit "onetimepin" pseudo-IdP that Cloudflare auto-creates.
func (c *Client) ListIdentityProviders(ctx context.Context, accountID string) ([]IdentityProvider, error) {
	path := fmt.Sprintf("/accounts/%s/access/identity_providers", url.PathEscape(accountID))
	result, err := c.do(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	var idps []IdentityProvider
	if err := json.Unmarshal(result, &idps); err != nil {
		return nil, fmt.Errorf("cloudflare: decode identity providers: %w", err)
	}
	return idps, nil
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

// AccessAppOptions carries optional configuration for Create/Update calls.
// Zero values are omitted from the request so Cloudflare keeps defaults.
type AccessAppOptions struct {
	// AllowedIdPs, if non-empty, restricts the app to those IdP IDs. Combined
	// with AutoRedirectToIdentity=true and a single entry, this enables
	// Cloudflare's "instant auth" flow (skip the IdP picker).
	AllowedIdPs []string
	// AutoRedirectToIdentity skips Cloudflare's IdP selection screen. Only
	// honored by Cloudflare when exactly one IdP is allowed.
	AutoRedirectToIdentity bool
}

// CreateAccessApp creates a self_hosted Access app scoped to domain, attached
// to the given reusable policy IDs, and returns the created app.
func (c *Client) CreateAccessApp(ctx context.Context, accountID, name, domain string, policyIDs []string, opts AccessAppOptions) (*AccessApp, error) {
	body := accessAppRequest{
		Name:                   name,
		Domain:                 domain,
		Type:                   "self_hosted",
		Policies:               policyIDs,
		AllowedIdPs:            opts.AllowedIdPs,
		AutoRedirectToIdentity: opts.AutoRedirectToIdentity,
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
func (c *Client) UpdateAccessApp(ctx context.Context, accountID, appID, name, domain string, policyIDs []string, opts AccessAppOptions) error {
	body := accessAppRequest{
		Name:                   name,
		Domain:                 domain,
		Type:                   "self_hosted",
		Policies:               policyIDs,
		AllowedIdPs:            opts.AllowedIdPs,
		AutoRedirectToIdentity: opts.AutoRedirectToIdentity,
	}
	path := fmt.Sprintf("/accounts/%s/access/apps/%s", url.PathEscape(accountID), url.PathEscape(appID))
	_, err := c.do(ctx, "PUT", path, body)
	return err
}
