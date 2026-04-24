package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// RPCRequest is a JSON-RPC 2.0 request from the client.
type RPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      json.RawMessage `json:"id"`
}

// RPCResponse is a JSON-RPC 2.0 response we send back to the client.
type RPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
	ID      json.RawMessage `json:"id"`
}

// RPCError is the error object inside a JSON-RPC 2.0 response.
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Config holds all proxy settings.
type Config struct {
	UpstreamURL    string
	ListenAddr     string
	RequestTimeout time.Duration
	MaxBodyBytes   int64
}

// Proxy forwards JSON-RPC calls to the upstream blockchain node.
type Proxy struct {
	cfg         Config
	httpClient  *http.Client
	upstreamURL *url.URL
}

// New creates a Proxy from config.
func New(cfg Config) (*Proxy, error) {
	u, err := url.Parse(cfg.UpstreamURL)
	if err != nil {
		return nil, fmt.Errorf("invalid upstream URL: %w", err)
	}

	client := &http.Client{
		Timeout: cfg.RequestTimeout,
	}

	return &Proxy{
		cfg:         cfg,
		httpClient:  client,
		upstreamURL: u,
	}, nil
}

// ServeHTTP is the main entry point — every HTTP request comes here.
func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Health check endpoint
	if r.Method == http.MethodGet && r.URL.Path == "/healthz" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
		return
	}

	// Only POST is allowed for JSON-RPC
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, -32600, "only POST is supported")
		return
	}

	// Read the request body
	body, err := io.ReadAll(io.LimitReader(r.Body, p.cfg.MaxBodyBytes))
	if err != nil {
		writeError(w, http.StatusBadRequest, -32700, "failed to read body")
		return
	}
	defer r.Body.Close()

	// Is it a batch (starts with '[') or single request?
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) > 0 && trimmed[0] == '[' {
		p.handleBatch(w, r, body)
	} else {
		p.handleSingle(w, r, body)
	}
}

func (p *Proxy) handleSingle(w http.ResponseWriter, r *http.Request, body []byte) {
	var req RPCRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(w, http.StatusBadRequest, -32700, "invalid JSON")
		return
	}
	if err := validate(req); err != nil {
		writeError(w, http.StatusBadRequest, -32600, err.Error())
		return
	}

	respBody, err := p.forward(r, body)
	if err != nil {
		writeError(w, http.StatusBadGateway, -32603, "upstream error: "+err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(respBody)
}

func (p *Proxy) handleBatch(w http.ResponseWriter, r *http.Request, body []byte) {
	var reqs []RPCRequest
	if err := json.Unmarshal(body, &reqs); err != nil {
		writeError(w, http.StatusBadRequest, -32700, "invalid JSON batch")
		return
	}
	if len(reqs) == 0 {
		writeError(w, http.StatusBadRequest, -32600, "empty batch")
		return
	}

	respBody, err := p.forward(r, body)
	if err != nil {
		writeError(w, http.StatusBadGateway, -32603, "upstream error: "+err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(respBody)
}

// forward sends the body to polygon.drpc.org and returns the response.
func (p *Proxy) forward(r *http.Request, body []byte) ([]byte, error) {
	req, err := http.NewRequestWithContext(
		r.Context(),
		http.MethodPost,
		p.upstreamURL.String(),
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

// validate checks the basic JSON-RPC 2.0 rules.
func validate(req RPCRequest) error {
	if req.JSONRPC != "2.0" {
		return fmt.Errorf(`jsonrpc must be "2.0"`)
	}
	if req.Method == "" {
		return fmt.Errorf("method is required")
	}
	return nil
}

// writeError sends a JSON-RPC error response.
func writeError(w http.ResponseWriter, httpCode, rpcCode int, msg string) {
	resp := RPCResponse{
		JSONRPC: "2.0",
		Error:   &RPCError{Code: rpcCode, Message: msg},
		ID:      json.RawMessage("null"),
	}
	data, _ := json.Marshal(resp)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpCode)
	w.Write(data)
}