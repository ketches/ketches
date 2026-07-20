package egress

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/ketches/ketches/internal/app"
	"golang.org/x/net/idna"
)

const (
	DefaultDialTimeout       = 5 * time.Second
	DefaultRequestTimeout    = 30 * time.Second
	DefaultMaxResponseBytes  = 8 << 20
	DefaultMaxResponseHeader = 1 << 20
)

var (
	ErrHostNotAllowed   = errors.New("outbound host is not allowed")
	ErrResponseTooLarge = errors.New("outbound response exceeds size limit")
	ErrUnsafeAddress    = errors.New("outbound address is not public")
)

var blockedAddressPrefixes = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"),
	netip.MustParsePrefix("10.0.0.0/8"),
	netip.MustParsePrefix("100.64.0.0/10"),
	netip.MustParsePrefix("100.100.100.200/32"),
	netip.MustParsePrefix("127.0.0.0/8"),
	netip.MustParsePrefix("168.63.129.16/32"),
	netip.MustParsePrefix("169.254.0.0/16"),
	netip.MustParsePrefix("172.16.0.0/12"),
	netip.MustParsePrefix("192.0.0.0/24"),
	netip.MustParsePrefix("192.0.2.0/24"),
	netip.MustParsePrefix("192.168.0.0/16"),
	netip.MustParsePrefix("198.18.0.0/15"),
	netip.MustParsePrefix("198.51.100.0/24"),
	netip.MustParsePrefix("203.0.113.0/24"),
	netip.MustParsePrefix("224.0.0.0/4"),
	netip.MustParsePrefix("240.0.0.0/4"),
	netip.MustParsePrefix("::/128"),
	netip.MustParsePrefix("::1/128"),
	netip.MustParsePrefix("64:ff9b::/96"),
	netip.MustParsePrefix("64:ff9b:1::/48"),
	netip.MustParsePrefix("100::/64"),
	netip.MustParsePrefix("2001::/23"),
	netip.MustParsePrefix("2002::/16"),
	netip.MustParsePrefix("fc00::/7"),
	netip.MustParsePrefix("fe80::/10"),
	netip.MustParsePrefix("ff00::/8"),
}

var blockedHostnames = map[string]struct{}{
	"instance-data":              {},
	"instance-data.ec2.internal": {},
	"metadata":                   {},
	"metadata.azure.internal":    {},
	"metadata.google.internal":   {},
}

type Resolver interface {
	LookupNetIP(ctx context.Context, network, host string) ([]netip.Addr, error)
}

type Policy struct {
	allowedHosts []string
	resolver     Resolver
	dialer       *net.Dialer
}

type ResolvedEndpoint struct {
	URL       *url.URL
	Host      string
	Addresses []netip.Addr
}

func CurrentPolicy() *Policy {
	return NewPolicy(app.Config.EgressAllowedHosts)
}

func NewPolicy(allowedHosts string) *Policy {
	return &Policy{
		allowedHosts: parseAllowedHosts(allowedHosts),
		resolver:     net.DefaultResolver,
		dialer: &net.Dialer{
			Timeout:   DefaultDialTimeout,
			KeepAlive: 30 * time.Second,
		},
	}
}

func (p *Policy) ValidateURLSyntax(rawURL string, allowedSchemes ...string) (*url.URL, error) {
	trimmed := strings.TrimSpace(rawURL)
	if trimmed == "" {
		return nil, errors.New("outbound URL is required")
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return nil, app.WrapErrorf(err, "parse outbound URL: %w", err)
	}
	if parsed.Scheme == "" || parsed.Host == "" || parsed.Hostname() == "" {
		return nil, errors.New("outbound URL must include a scheme and host")
	}

	parsed.Scheme = strings.ToLower(parsed.Scheme)
	if !containsFold(allowedSchemes, parsed.Scheme) {
		return nil, app.NewErrorf("outbound URL scheme %q is not allowed", parsed.Scheme)
	}

	host, err := normalizeHostname(parsed.Hostname())
	if err != nil {
		return nil, err
	}
	if strings.HasSuffix(parsed.Hostname(), ".") {
		return nil, app.NewErrorf("outbound host must not use a trailing dot")
	}
	if err := p.validateHostname(host); err != nil {
		return nil, err
	}
	if port := parsed.Port(); port != "" {
		value, err := strconv.Atoi(port)
		if err != nil || value < 1 || value > 65535 {
			return nil, errors.New("outbound URL has an invalid port")
		}
	}

	return parsed, nil
}

func (p *Policy) ValidateURL(ctx context.Context, rawURL string, allowedSchemes ...string) (*ResolvedEndpoint, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	parsed, err := p.ValidateURLSyntax(rawURL, allowedSchemes...)
	if err != nil {
		return nil, err
	}

	host, err := normalizeHostname(parsed.Hostname())
	if err != nil {
		return nil, err
	}
	addresses, err := p.ResolvePublicHost(ctx, host)
	if err != nil {
		return nil, err
	}

	return &ResolvedEndpoint{URL: parsed, Host: host, Addresses: addresses}, nil
}

func (p *Policy) ResolvePublicHost(ctx context.Context, host string) ([]netip.Addr, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	normalizedHost, err := normalizeHostname(host)
	if err != nil {
		return nil, err
	}
	if err := p.validateHostname(normalizedHost); err != nil {
		return nil, err
	}

	if address, err := netip.ParseAddr(normalizedHost); err == nil {
		address = address.Unmap()
		if err := validatePublicAddress(address); err != nil {
			return nil, err
		}
		return []netip.Addr{address}, nil
	}

	addresses, err := p.resolver.LookupNetIP(ctx, "ip", normalizedHost)
	if err != nil {
		return nil, app.WrapErrorf(err, "resolve outbound host %q: %w", normalizedHost, err)
	}
	if len(addresses) == 0 {
		return nil, app.NewErrorf("resolve outbound host %q: no addresses returned", normalizedHost)
	}

	unique := make([]netip.Addr, 0, len(addresses))
	seen := make(map[netip.Addr]struct{}, len(addresses))
	for _, address := range addresses {
		address = address.Unmap()
		if err := validatePublicAddress(address); err != nil {
			return nil, app.WrapErrorf(err, "outbound host %q resolved to unsafe address %s: %w", normalizedHost, address, err)
		}
		if _, ok := seen[address]; ok {
			continue
		}
		seen[address] = struct{}{}
		unique = append(unique, address)
	}

	return unique, nil
}

func (p *Policy) NewHTTPClient(timeout time.Duration, maxResponseBytes int64) *http.Client {
	return &http.Client{
		Transport: p.NewHTTPTransport(maxResponseBytes),
		Timeout:   timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return errors.New("outbound redirect limit exceeded")
			}
			_, err := p.ValidateURL(req.Context(), req.URL.String(), "https")
			return err
		},
	}
}

func (p *Policy) NewHTTPTransport(maxResponseBytes int64) http.RoundTripper {
	return p.newHTTPTransport(maxResponseBytes, nil)
}

func (p *Policy) NewHTTPTransportWithTLSConfig(maxResponseBytes int64, tlsConfig *tls.Config) http.RoundTripper {
	return p.newHTTPTransport(maxResponseBytes, tlsConfig)
}

func (p *Policy) newHTTPTransport(maxResponseBytes int64, tlsConfig *tls.Config) http.RoundTripper {
	if maxResponseBytes < 1 {
		maxResponseBytes = DefaultMaxResponseBytes
	}

	transport := &http.Transport{ForceAttemptHTTP2: true}
	if defaultTransport, ok := http.DefaultTransport.(*http.Transport); ok {
		transport = defaultTransport.Clone()
	}
	transport.Proxy = nil
	transport.DialContext = p.DialContext
	transport.TLSHandshakeTimeout = 10 * time.Second
	transport.ResponseHeaderTimeout = 20 * time.Second
	transport.MaxResponseHeaderBytes = DefaultMaxResponseHeader
	if tlsConfig != nil {
		transport.TLSClientConfig = tlsConfig.Clone()
	}

	return &validatingRoundTripper{
		policy:           p,
		transport:        transport,
		maxResponseBytes: maxResponseBytes,
	}
}

func (p *Policy) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, app.WrapErrorf(err, "parse outbound address %q: %w", address, err)
	}
	addresses, err := p.ResolvePublicHost(ctx, host)
	if err != nil {
		return nil, err
	}

	var dialErr error
	for _, resolved := range addresses {
		conn, err := p.dialer.DialContext(ctx, network, net.JoinHostPort(resolved.String(), port))
		if err == nil {
			return conn, nil
		}
		dialErr = err
	}
	return nil, app.WrapErrorf(dialErr, "dial outbound host %q: %w", host, dialErr)
}

func (p *Policy) validateHostname(host string) error {
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return app.WrapErrorf(ErrUnsafeAddress, "%w: %s", ErrUnsafeAddress, host)
	}
	if _, blocked := blockedHostnames[host]; blocked {
		return app.WrapErrorf(ErrUnsafeAddress, "%w: %s", ErrUnsafeAddress, host)
	}
	if len(p.allowedHosts) > 0 && !hostMatchesAllowlist(host, p.allowedHosts) {
		return app.WrapErrorf(ErrHostNotAllowed, "%w: %s", ErrHostNotAllowed, host)
	}
	if address, err := netip.ParseAddr(host); err == nil {
		return validatePublicAddress(address.Unmap())
	}
	return nil
}

func validatePublicAddress(address netip.Addr) error {
	if !address.IsValid() || !address.IsGlobalUnicast() || address.IsPrivate() || address.IsLoopback() || address.IsLinkLocalUnicast() || address.IsLinkLocalMulticast() || address.IsMulticast() || address.IsUnspecified() {
		return app.WrapErrorf(ErrUnsafeAddress, "%w: %s", ErrUnsafeAddress, address)
	}
	for _, prefix := range blockedAddressPrefixes {
		if prefix.Contains(address) {
			return app.WrapErrorf(ErrUnsafeAddress, "%w: %s", ErrUnsafeAddress, address)
		}
	}
	return nil
}

func normalizeHostname(host string) (string, error) {
	trimmed := strings.TrimSuffix(strings.TrimSpace(host), ".")
	if trimmed == "" {
		return "", errors.New("outbound host is required")
	}
	if address, err := netip.ParseAddr(trimmed); err == nil {
		return address.Unmap().String(), nil
	}
	value, err := idna.Lookup.ToASCII(trimmed)
	if err != nil {
		return "", app.WrapErrorf(err, "normalize outbound host: %w", err)
	}
	return strings.ToLower(value), nil
}

func parseAllowedHosts(raw string) []string {
	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		value := strings.ToLower(strings.TrimSuffix(strings.TrimSpace(part), "."))
		if value == "" {
			continue
		}
		if strings.HasPrefix(value, "*.") {
			suffix, err := normalizeHostname(strings.TrimPrefix(value, "*."))
			if err == nil {
				result = append(result, "*."+suffix)
			} else {
				result = append(result, "\x00invalid")
			}
			continue
		}
		if normalized, err := normalizeHostname(value); err == nil {
			result = append(result, normalized)
		} else {
			result = append(result, "\x00invalid")
		}
	}
	return result
}

func hostMatchesAllowlist(host string, allowedHosts []string) bool {
	for _, allowed := range allowedHosts {
		if host == allowed {
			return true
		}
		if strings.HasPrefix(allowed, "*.") {
			suffix := strings.TrimPrefix(allowed, "*")
			if strings.HasSuffix(host, suffix) && host != strings.TrimPrefix(suffix, ".") {
				return true
			}
		}
	}
	return false
}

func containsFold(values []string, target string) bool {
	for _, value := range values {
		if strings.EqualFold(value, target) {
			return true
		}
	}
	return false
}

type validatingRoundTripper struct {
	policy           *Policy
	transport        http.RoundTripper
	maxResponseBytes int64
}

func (t *validatingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if _, err := t.policy.ValidateURL(req.Context(), req.URL.String(), "https"); err != nil {
		return nil, err
	}

	response, err := t.transport.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	if response.ContentLength > t.maxResponseBytes {
		_ = response.Body.Close()
		return nil, ErrResponseTooLarge
	}
	response.Body = &maxBytesReadCloser{
		body:      response.Body,
		remaining: t.maxResponseBytes,
	}
	return response, nil
}

type maxBytesReadCloser struct {
	body      io.ReadCloser
	remaining int64
	exceeded  bool
}

func (r *maxBytesReadCloser) Read(buffer []byte) (int, error) {
	if r.exceeded {
		return 0, ErrResponseTooLarge
	}
	if int64(len(buffer)) > r.remaining+1 {
		buffer = buffer[:r.remaining+1]
	}

	read, err := r.body.Read(buffer)
	if int64(read) > r.remaining {
		allowed := int(r.remaining)
		r.remaining = 0
		r.exceeded = true
		if allowed > 0 {
			return allowed, nil
		}
		return 0, ErrResponseTooLarge
	}
	r.remaining -= int64(read)
	return read, err
}

func (r *maxBytesReadCloser) Close() error {
	return r.body.Close()
}
