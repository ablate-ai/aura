package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/c.chen/aura/config"
)

// Client Prometheus 客户端
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient 创建新的 Prometheus 客户端
func NewClient(cfg *config.Config) *Client {
	timeout := 30 * time.Second
	if cfg.Timeout > 0 {
		timeout = cfg.Timeout
	}

	httpClient := &http.Client{Timeout: timeout}
	if cfg.HTTPClient != nil {
		if hc, ok := cfg.HTTPClient.(*http.Client); ok {
			httpClient = hc
		}
	}

	return &Client{
		baseURL:    cfg.BaseURL,
		httpClient: httpClient,
	}
}

// QueryResult 查询结果
type QueryResult struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric map[string]string `json:"metric"`
			Value  []interface{}     `json:"value"`
		} `json:"result"`
	} `json:"data"`
	ErrorType string `json:"errorType,omitempty"`
	Error     string `json:"error,omitempty"`
}

// RangeQueryResult 范围查询结果
type RangeQueryResult struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric map[string]string `json:"metric"`
			Values [][]interface{}   `json:"values"`
		} `json:"result"`
	} `json:"data"`
	ErrorType string `json:"errorType,omitempty"`
	Error     string `json:"error,omitempty"`
}

// QueryInstant 即时查询（单点查询）
func (c *Client) QueryInstant(ctx context.Context, query string, timestamp time.Time) (*QueryResult, error) {
	u, err := url.Parse(c.baseURL + "api/v1/query")
	if err != nil {
		return nil, fmt.Errorf("解析 URL 失败: %w", err)
	}

	q := u.Query()
	q.Set("query", query)
	if !timestamp.IsZero() {
		q.Set("time", strconv.FormatInt(timestamp.Unix(), 10))
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	var result QueryResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	if result.Status != "success" {
		return &result, fmt.Errorf("查询错误: %s", result.Error)
	}

	return &result, nil
}

// QueryRange 范围查询
func (c *Client) QueryRange(ctx context.Context, query string, start, end time.Time, step time.Duration) (*RangeQueryResult, error) {
	u, err := url.Parse(c.baseURL + "api/v1/query_range")
	if err != nil {
		return nil, fmt.Errorf("解析 URL 失败: %w", err)
	}

	q := u.Query()
	q.Set("query", query)
	q.Set("start", strconv.FormatInt(start.Unix(), 10))
	q.Set("end", strconv.FormatInt(end.Unix(), 10))
	q.Set("step", strconv.FormatFloat(step.Seconds(), 'f', -1, 64))
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	var result RangeQueryResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	if result.Status != "success" {
		return &result, fmt.Errorf("查询错误: %s", result.Error)
	}

	return &result, nil
}
