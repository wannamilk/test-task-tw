package proxy

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// newTestProxy creates a Proxy pointing at a fake upstream server.
func newTestProxy(t *testing.T, upstreamURL string) *Proxy {
	t.Helper()
	p, err := New(Config{
		UpstreamURL:    upstreamURL,
		RequestTimeout: 5 * time.Second,
		MaxBodyBytes:   1 << 20,
	})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	return p
}

// fakeUpstream creates a test server that always returns the given body.
func fakeUpstream(t *testing.T, body string, code int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		io.WriteString(w, body)
	}))
}

func TestHealthz(t *testing.T) {
	up := fakeUpstream(t, `{}`, 200)
	defer up.Close()

	srv := httptest.NewServer(newTestProxy(t, up.URL))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("want 200, got %d", resp.StatusCode)
	}
}

func TestSingleRequest(t *testing.T) {
	up := fakeUpstream(t, `{"jsonrpc":"2.0","result":"0x1386de4","id":1}`, 200)
	defer up.Close()

	srv := httptest.NewServer(newTestProxy(t, up.URL))
	defer srv.Close()

	body := `{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}`
	resp, err := http.Post(srv.URL, "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("want 200, got %d", resp.StatusCode)
	}

	var rpc RPCResponse
	json.NewDecoder(resp.Body).Decode(&rpc)
	if rpc.Error != nil {
		t.Fatalf("unexpected error: %+v", rpc.Error)
	}
}

func TestBatchRequest(t *testing.T) {
	up := fakeUpstream(t, `[{"jsonrpc":"2.0","result":"0x1","id":1},{"jsonrpc":"2.0","result":"137","id":2}]`, 200)
	defer up.Close()

	srv := httptest.NewServer(newTestProxy(t, up.URL))
	defer srv.Close()

	body := `[{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1},{"jsonrpc":"2.0","method":"net_version","params":[],"id":2}]`
	resp, err := http.Post(srv.URL, "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("want 200, got %d", resp.StatusCode)
	}
}

func TestInvalidJSON(t *testing.T) {
	up := fakeUpstream(t, `{}`, 200)
	defer up.Close()

	srv := httptest.NewServer(newTestProxy(t, up.URL))
	defer srv.Close()

	resp, _ := http.Post(srv.URL, "application/json", bytes.NewBufferString(`{bad json`))
	defer resp.Body.Close()

	if resp.StatusCode != 400 {
		t.Errorf("want 400, got %d", resp.StatusCode)
	}

	var rpc RPCResponse
	json.NewDecoder(resp.Body).Decode(&rpc)
	if rpc.Error == nil || rpc.Error.Code != -32700 {
		t.Errorf("want code -32700, got %+v", rpc.Error)
	}
}

func TestWrongVersion(t *testing.T) {
	up := fakeUpstream(t, `{}`, 200)
	defer up.Close()

	srv := httptest.NewServer(newTestProxy(t, up.URL))
	defer srv.Close()

	body := `{"jsonrpc":"1.0","method":"eth_blockNumber","params":[],"id":1}`
	resp, _ := http.Post(srv.URL, "application/json", bytes.NewBufferString(body))
	defer resp.Body.Close()

	if resp.StatusCode != 400 {
		t.Errorf("want 400, got %d", resp.StatusCode)
	}
}

func TestMissingMethod(t *testing.T) {
	up := fakeUpstream(t, `{}`, 200)
	defer up.Close()

	srv := httptest.NewServer(newTestProxy(t, up.URL))
	defer srv.Close()

	body := `{"jsonrpc":"2.0","params":[],"id":1}`
	resp, _ := http.Post(srv.URL, "application/json", bytes.NewBufferString(body))
	defer resp.Body.Close()

	if resp.StatusCode != 400 {
		t.Errorf("want 400, got %d", resp.StatusCode)
	}
}

func TestNonPostRejected(t *testing.T) {
	up := fakeUpstream(t, `{}`, 200)
	defer up.Close()

	srv := httptest.NewServer(newTestProxy(t, up.URL))
	defer srv.Close()

	resp, _ := http.Get(srv.URL + "/")
	defer resp.Body.Close()

	if resp.StatusCode != 405 {
		t.Errorf("want 405, got %d", resp.StatusCode)
	}
}

func TestUpstreamUnreachable(t *testing.T) {
	p, _ := New(Config{
		UpstreamURL:    "http://127.0.0.1:19999", // nothing here
		RequestTimeout: 1 * time.Second,
		MaxBodyBytes:   1 << 20,
	})
	srv := httptest.NewServer(p)
	defer srv.Close()

	body := `{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}`
	resp, _ := http.Post(srv.URL, "application/json", bytes.NewBufferString(body))
	defer resp.Body.Close()

	if resp.StatusCode != 502 {
		t.Errorf("want 502, got %d", resp.StatusCode)
	}
}

func TestEmptyBatch(t *testing.T) {
	up := fakeUpstream(t, `{}`, 200)
	defer up.Close()

	srv := httptest.NewServer(newTestProxy(t, up.URL))
	defer srv.Close()

	resp, _ := http.Post(srv.URL, "application/json", bytes.NewBufferString(`[]`))
	defer resp.Body.Close()

	if resp.StatusCode != 400 {
		t.Errorf("want 400, got %d", resp.StatusCode)
	}
}

func TestValidate(t *testing.T) {
	cases := []struct {
		req     RPCRequest
		wantErr bool
	}{
		{RPCRequest{JSONRPC: "2.0", Method: "eth_blockNumber"}, false},
		{RPCRequest{JSONRPC: "1.0", Method: "eth_blockNumber"}, true},
		{RPCRequest{JSONRPC: "2.0", Method: ""}, true},
	}
	for _, c := range cases {
		err := validate(c.req)
		if (err != nil) != c.wantErr {
			t.Errorf("validate(%+v) err=%v wantErr=%v", c.req, err, c.wantErr)
		}
	}
}