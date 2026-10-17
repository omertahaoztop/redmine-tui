package netguard

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func request(t *testing.T, raw string) *http.Request {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return &http.Request{URL: u}
}

func TestRequireSecureTransport(t *testing.T) {
	allowed := []string{
		"https://redmine.example.com",
		"http://127.0.0.1:3456",
		"http://localhost:3456/api/v1",
		"http://[::1]:3456",
	}
	for _, raw := range allowed {
		if err := RequireSecureTransport(request(t, raw).URL, raw); err != nil {
			t.Errorf("RequireSecureTransport(%q) = %v, want nil", raw, err)
		}
	}

	rejected := []string{
		"http://redmine.example.com",
		"http://10.0.0.5:3456/api/v1",
		"http://[2001:db8::1]",
	}
	for _, raw := range rejected {
		if err := RequireSecureTransport(request(t, raw).URL, raw); err == nil {
			t.Errorf("RequireSecureTransport(%q) accepted a plaintext remote endpoint", raw)
		}
	}
}

// Go strips the Authorization header on a cross-host redirect but keeps custom
// credential headers, and it keeps everything on a same-host downgrade.
func TestCheckRedirectStaysOnOrigin(t *testing.T) {
	via := []*http.Request{request(t, "https://vikunja.example.com/api/v1/login")}

	cases := []struct {
		to string
		ok bool
	}{
		{"https://vikunja.example.com/api/v1/tasks", true},
		{"https://vikunja.example.com:443/api/v1/tasks", true},
		{"http://vikunja.example.com/api/v1/tasks", false},
		{"https://vikunja.example.com:8443/api/v1/tasks", false},
		{"https://evil.example/api/v1/tasks", false},
		{"https://app.vikunja.example.com/api/v1/tasks", false},
	}
	for _, c := range cases {
		err := CheckRedirect(request(t, c.to), via)
		if c.ok && err != nil {
			t.Errorf("CheckRedirect(%q) = %v, want nil", c.to, err)
		}
		if !c.ok && err == nil {
			t.Errorf("CheckRedirect(%q) was allowed", c.to)
		}
	}
}

func TestCheckRedirectAllowsLoopback(t *testing.T) {
	via := []*http.Request{request(t, "http://127.0.0.1:3456/api/v1")}
	if err := CheckRedirect(request(t, "http://127.0.0.1:3456/api/v1/tasks"), via); err != nil {
		t.Fatalf("loopback redirect refused: %v", err)
	}
}

func TestCheckRedirectStopsLongChains(t *testing.T) {
	via := make([]*http.Request, 10)
	for i := range via {
		via[i] = request(t, "https://vikunja.example.com/api/v1")
	}
	if err := CheckRedirect(request(t, "https://vikunja.example.com/api/v1/x"), via); err == nil {
		t.Fatal("a ten hop redirect chain was allowed")
	}
}

// The body must be closed and drained so the transport can reuse the connection
// for the next request.
func TestDecodeJSONClosesAndDrains(t *testing.T) {
	body := &trackedBody{Reader: strings.NewReader(`{"a":1} trailing`)}
	var v map[string]int
	if err := DecodeJSON(body, &v); err != nil {
		t.Fatalf("DecodeJSON failed: %v", err)
	}
	if v["a"] != 1 {
		t.Fatalf("decoded %v", v)
	}
	if !body.drained {
		t.Error("the remainder was not read out")
	}
	if !body.closed {
		t.Error("the body was not closed")
	}
}

// A response larger than the budget must not be decoded in full.
func TestDecodeJSONBoundsTheBody(t *testing.T) {
	body := &trackedBody{Reader: strings.NewReader(strings.Repeat("x", MaxResponseBytes+1))}
	var v interface{}
	if err := DecodeJSON(body, &v); err == nil {
		t.Fatal("an oversized body was decoded")
	}
}

type trackedBody struct {
	io.Reader
	drained bool
	closed  bool
}

func (b *trackedBody) Read(p []byte) (int, error) {
	n, err := b.Reader.Read(p)
	if err == io.EOF {
		b.drained = true
	}
	return n, err
}

func (b *trackedBody) Close() error {
	b.closed = true
	return nil
}
