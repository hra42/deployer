package cloudflare

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// Zone is a minimal view of a Cloudflare zone.
type Zone struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// FindZoneByDomain locates the Cloudflare zone that owns the given fully-
// qualified domain. It walks from the most-specific candidate (the full
// domain) up to the apex, asking Cloudflare for an exact match at each level,
// and returns the first hit. Subzones (delegated subdomains held as their own
// Cloudflare zone) are handled correctly because they match before their
// parent.
//
// Requires the API token to have "Zone:Zone:Read" on the account so
// /zones?name=... can be queried.
func (c *Client) FindZoneByDomain(ctx context.Context, domain string) (*Zone, error) {
	candidate := strings.TrimSuffix(strings.ToLower(domain), ".")
	if candidate == "" {
		return nil, fmt.Errorf("cloudflare: empty domain")
	}
	for {
		q := url.Values{}
		q.Set("name", candidate)
		result, err := c.do(ctx, "GET", "/zones?"+q.Encode(), nil)
		if err != nil {
			return nil, err
		}
		var zones []Zone
		if err := json.Unmarshal(result, &zones); err != nil {
			return nil, fmt.Errorf("cloudflare: decode zones: %w", err)
		}
		for i := range zones {
			if zones[i].Name == candidate {
				return &zones[i], nil
			}
		}
		// Strip the leftmost label and try the parent. Stop once there's no
		// dot left (we'd be querying the TLD, which Cloudflare doesn't own).
		idx := strings.Index(candidate, ".")
		if idx < 0 {
			break
		}
		candidate = candidate[idx+1:]
		if !strings.Contains(candidate, ".") {
			break
		}
	}
	return nil, fmt.Errorf("cloudflare: no zone found for %q (token may lack Zone:Read or the domain isn't on this account)", domain)
}

type DNSRecord struct {
	ID      string `json:"id,omitempty"`
	Type    string `json:"type"`
	Name    string `json:"name"`
	Content string `json:"content"`
	Proxied bool   `json:"proxied"`
	TTL     int    `json:"ttl"`
}

// FindCNAME returns the first CNAME record matching name in the given zone, or
// nil if no record exists.
func (c *Client) FindCNAME(ctx context.Context, zoneID, name string) (*DNSRecord, error) {
	q := url.Values{}
	q.Set("type", "CNAME")
	q.Set("name", name)
	path := fmt.Sprintf("/zones/%s/dns_records?%s", url.PathEscape(zoneID), q.Encode())

	result, err := c.do(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var records []DNSRecord
	if err := json.Unmarshal(result, &records); err != nil {
		return nil, fmt.Errorf("cloudflare: decode dns records: %w", err)
	}
	if len(records) == 0 {
		return nil, nil
	}
	return &records[0], nil
}

// CreateCNAME creates a new proxied CNAME pointing name → target.
func (c *Client) CreateCNAME(ctx context.Context, zoneID, name, target string) error {
	body := DNSRecord{
		Type:    "CNAME",
		Name:    name,
		Content: target,
		Proxied: true,
		TTL:     1,
	}
	path := fmt.Sprintf("/zones/%s/dns_records", url.PathEscape(zoneID))
	_, err := c.do(ctx, "POST", path, body)
	return err
}

// UpdateCNAME replaces the record at recordID with a proxied CNAME pointing
// name → target.
func (c *Client) UpdateCNAME(ctx context.Context, zoneID, recordID, name, target string) error {
	body := DNSRecord{
		Type:    "CNAME",
		Name:    name,
		Content: target,
		Proxied: true,
		TTL:     1,
	}
	path := fmt.Sprintf("/zones/%s/dns_records/%s", url.PathEscape(zoneID), url.PathEscape(recordID))
	_, err := c.do(ctx, "PUT", path, body)
	return err
}
