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

// The constructor's loopback HTTP exception must not permit redirects to undo
// HTTPS, including HTTPS reached after starting with an explicit HTTP endpoint.
func TestCheckRedirectNeverDowngradesHTTPS(t *testing.T) {
	cases := []struct {
		name string
		urls []string
	}{
		{"localhost", []string{"https://localhost:3456/start", "http://localhost:3456/target"}},
		{"IPv4 loopback", []string{"https://127.0.0.1:3456/start", "http://127.0.0.1:3456/target"}},
		{"IPv6 loopback", []string{"https://[::1]:3456/start", "http://[::1]:3456/target"}},
		{"default HTTPS port", []string{"https://localhost/start", "http://localhost:443/target"}},
		{"after upgrade", []string{"http://localhost:443/start", "https://localhost:443/secure", "http://localhost:443/target"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			calls, plaintextRedirects := 0, 0
			client := &http.Client{
				CheckRedirect: CheckRedirect,
				Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
					if calls > 0 && req.URL.Scheme == "http" {
						plaintextRedirects++
					}
					calls++
					resp := &http.Response{
						StatusCode: http.StatusOK,
						Header:     make(http.Header),
						Body:       io.NopCloser(strings.NewReader("{}")),
						Request:    req,
					}
					if calls < len(tc.urls) {
						resp.StatusCode = http.StatusTemporaryRedirect
						resp.Header.Set("Location", tc.urls[calls])
					}
					return resp, nil
				}),
			}
			req, err := http.NewRequest(http.MethodPost, tc.urls[0], strings.NewReader(`{"password":"synthetic"}`))
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("X-Redmine-API-Key", "synthetic-key")
			req.Header.Set("Authorization", "Bearer synthetic-token")
			resp, err := client.Do(req)
			if resp != nil {
				if closeErr := resp.Body.Close(); closeErr != nil {
					t.Error(closeErr)
				}
			}
			if err == nil || !strings.Contains(err.Error(), "refusing redirect") {
				t.Errorf("expected downgrade refusal, got %v", err)
			}
			if calls != len(tc.urls)-1 || plaintextRedirects != 0 {
				t.Errorf("transport calls = %d, plaintext redirects = %d; want %d and zero", calls, plaintextRedirects, len(tc.urls)-1)
			}
		})
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
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

// A response larger than the budget must not be decoded in full. The payload is
// valid JSON so that the size limit is the only thing that can fail the decode.
func TestDecodeJSONBoundsTheBody(t *testing.T) {
	body := &trackedBody{Reader: strings.NewReader(`"` + strings.Repeat("x", MaxResponseBytes) + `"`)}
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
