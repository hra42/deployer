package cloudflare

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

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
