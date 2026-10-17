// Package netguard holds the rules the API clients apply when talking to a
// remote service: where credentials may travel, and how much of a response is
// trusted.
package netguard

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
)

// MaxResponseBytes caps how much of a response body is read. net/http gunzips
// transparently, so a small substituted response would otherwise be free to
// expand into an arbitrarily large allocation.
const MaxResponseBytes = 8 << 20

// IsLoopback reports whether host addresses the local machine. Only the literal
// "localhost" and loopback IPs qualify: any other name may resolve anywhere.
func IsLoopback(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// RequireSecureTransport returns an error when a credential-bearing request to u
// would travel in the clear. raw is echoed in the message so the user sees the
// value they configured. Loopback is exempt: that traffic never leaves the host.
func RequireSecureTransport(u *url.URL, raw string) error {
	if u.Scheme != "http" || IsLoopback(u.Hostname()) {
		return nil
	}
	return fmt.Errorf("refusing to send credentials to plaintext endpoint %q: use https", raw)
}

// CheckRedirect keeps credentials on the authority they were configured for: a
// redirect to another host or port would hand them to a third party (Go only
// strips the Authorization header, and only across hosts) and one that
// downgrades to http would expose them on the wire. HTTPS downgrades are rejected
// even on loopback. via[0] is the original request.
func CheckRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= 10 {
		return fmt.Errorf("stopped after 10 redirects")
	}
	if via[len(via)-1].URL.Scheme == "https" && req.URL.Scheme != "https" {
		return fmt.Errorf("refusing redirect from https to %s: credentials require encrypted transport", req.URL.Scheme)
	}
	origin := via[0].URL
	if !strings.EqualFold(req.URL.Hostname(), origin.Hostname()) || port(req.URL) != port(origin) {
		return fmt.Errorf("refusing redirect from %s to %s: credentials stay on the configured host",
			origin.Host, req.URL.Host)
	}
	return RequireSecureTransport(req.URL, req.URL.String())
}

// port returns the effective port, so "https://host" and "https://host:443"
// count as the same authority.
func port(u *url.URL) string {
	if p := u.Port(); p != "" {
		return p
	}
	if u.Scheme == "https" {
		return "443"
	}
	return "80"
}

// DecodeJSON decodes v from a response body, bounding how much of it is read and
// closing it. The remainder is read out afterwards so the transport can reuse
// the connection instead of opening a new one for the next request.
func DecodeJSON(body io.ReadCloser, v interface{}) error {
	defer body.Close()

	err := json.NewDecoder(io.LimitReader(body, MaxResponseBytes)).Decode(v)
	// Bounded on purpose: a response that already blew the decode budget is not
	// worth draining in full.
	io.Copy(io.Discard, io.LimitReader(body, MaxResponseBytes))
	return err
}
