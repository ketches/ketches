package egress

import (
	"context"
	"errors"
	"io"
	"net/netip"
	"strings"
	"testing"
)

type staticResolver struct {
	addresses []netip.Addr
	err       error
}

func (r staticResolver) LookupNetIP(context.Context, string, string) ([]netip.Addr, error) {
	return r.addresses, r.err
}

func TestPolicyRejectsUnsafeURLSyntax(t *testing.T) {
	policy := NewPolicy("")
	for _, rawURL := range []string{
		"http://example.com",
		"https://127.0.0.1",
		"https://10.0.0.1",
		"https://169.254.169.254",
		"https://localhost",
		"https://metadata.google.internal",
	} {
		if _, err := policy.ValidateURLSyntax(rawURL, "https"); err == nil {
			t.Errorf("expected %q to be rejected", rawURL)
		}
	}
}

func TestPolicyRejectsUnsafeDNSResult(t *testing.T) {
	policy := NewPolicy("")
	policy.resolver = staticResolver{addresses: []netip.Addr{netip.MustParseAddr("192.168.1.10")}}

	_, err := policy.ValidateURL(context.Background(), "https://service.example.com", "https")
	if !errors.Is(err, ErrUnsafeAddress) {
		t.Fatalf("expected unsafe address error, got %v", err)
	}
}

func TestPolicyAppliesHTTPSHostAllowlist(t *testing.T) {
	policy := NewPolicy("api.example.com,*.trusted.example")
	if _, err := policy.ValidateURLSyntax("https://api.example.com", "https"); err != nil {
		t.Fatalf("expected exact allowlisted host: %v", err)
	}
	if _, err := policy.ValidateURLSyntax("https://service.trusted.example", "https"); err != nil {
		t.Fatalf("expected wildcard allowlisted host: %v", err)
	}
	if _, err := policy.ValidateURLSyntax("https://example.com", "https"); !errors.Is(err, ErrHostNotAllowed) {
		t.Fatalf("expected host allowlist rejection, got %v", err)
	}
}

func TestMaxBytesReadCloserRejectsOversizedBody(t *testing.T) {
	reader := &maxBytesReadCloser{body: io.NopCloser(strings.NewReader("12345")), remaining: 4}
	_, err := io.ReadAll(reader)
	if !errors.Is(err, ErrResponseTooLarge) {
		t.Fatalf("expected response size error, got %v", err)
	}
}
