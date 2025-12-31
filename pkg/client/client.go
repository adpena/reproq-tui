package client

import (
	"context"
	"crypto/tls"
	"net/http"
	"strings"
	"sync"
	"time"
)

type Options struct {
	Timeout            time.Duration
	Headers            map[string]string
	InsecureSkipVerify bool
}

type Client struct {
	httpClient *http.Client
	headers    http.Header
	mu         sync.RWMutex
}

func New(opts Options) *Client {
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: opts.InsecureSkipVerify,
		},
	}
	headers := http.Header{}
	for key, val := range opts.Headers {
		headers.Set(key, val)
	}
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	return &Client{
		httpClient: &http.Client{
			Timeout:   timeout,
			Transport: transport,
		},
		headers: headers,
	}
}

func (c *Client) Do(req *http.Request) (*http.Response, error) {
	c.mu.RLock()
	for key, values := range c.headers {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}
	c.mu.RUnlock()
	return c.httpClient.Do(req)
}

func (c *Client) Get(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(req)
}

func (c *Client) SetHeader(key, value string) {
	if key == "" {
		return
	}
	c.mu.Lock()
	c.headers.Set(key, value)
	c.mu.Unlock()
}

func (c *Client) ClearHeader(key string) {
	if key == "" {
		return
	}
	c.mu.Lock()
	c.headers.Del(key)
	c.mu.Unlock()
}

func (c *Client) HasHeader(key string) bool {
	if key == "" {
		return false
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	for existing := range c.headers {
		if strings.EqualFold(existing, key) {
			return true
		}
	}
	return false
}
