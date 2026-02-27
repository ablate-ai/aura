package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/c.chen/aura/client"
	"github.com/c.chen/aura/config"
)

// Server HTTP API 服务器
type Server struct {
	promClient *client.Client
	httpServer *http.Server
	indexHTML  []byte
}

// NewServer 创建 API 服务器
func NewServer(cfg *config.Config, addr string, indexHTML []byte) *Server {
	promClient := client.NewClient(cfg)

	return &Server{
		promClient: promClient,
		indexHTML:  indexHTML,
		httpServer: &http.Server{
			Addr:         addr,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 0, // SSE 长连接不设超时
		},
	}
}

// ProbeStatus 探针状态
type ProbeStatus struct {
	Name       string  `json:"name"`
	Type       string  `json:"type"`       // blackbox 或 node
	Target     string  `json:"target"`     // 目标地址
	Status     string  `json:"status"`     // up 或 down
	Value      float64 `json:"value"`      // 当前值
	Timestamp  int64   `json:"timestamp"`  // 时间戳
	Instance   string  `json:"instance"`   // 实例
	Job        string  `json:"job"`        // 任务名
	MetricType string  `json:"metricType"` // 指标类型
}

// TrendData 趋势数据点
type TrendData struct {
	Timestamp int64   `json:"timestamp"`
	Value     float64 `json:"value"`
}

// AlertInfo 告警信息
type AlertInfo struct {
	Name   string `json:"name"`
	Type   string `json:"type"`
	Target string `json:"target"`
	Status string `json:"status"`
}

// NodeMetrics 节点指标
type NodeMetrics struct {
	Instance string  `json:"instance"`
	Name     string  `json:"name"`     // Prometheus name label（可选，用于公开展示）
	CPU      float64 `json:"cpu"`
	MemUsed  float64 `json:"memUsed"`
	MemTotal float64 `json:"memTotal"`
	Disk     float64 `json:"disk"`
	NetIn    float64 `json:"netIn"`
	NetOut   float64 `json:"netOut"`
	Load1    float64 `json:"load1"`
	Uptime   float64 `json:"uptime"`   // 系统运行时间（秒）
	Status   string  `json:"status"`
}

// Start 启动服务器
func (s *Server) Start() error {
	// 注册路由到独立 mux，避免污染全局 DefaultServeMux
	mux := http.NewServeMux()
	mux.HandleFunc("/api/probes", s.cors(s.handleProbes))
	mux.HandleFunc("/api/trend", s.cors(s.handleTrend))
	mux.HandleFunc("/api/alerts", s.cors(s.handleAlerts))
	mux.HandleFunc("/api/nodes", s.cors(s.handleNodes))
	mux.HandleFunc("/api/stream", s.handleStream)
	mux.HandleFunc("/", s.handleIndex)
	s.httpServer.Handler = mux

	return s.httpServer.ListenAndServe()
}

// Shutdown 关闭服务器
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

// CORS 中间件
func (s *Server) cors(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Content-Type", "application/json")

		if r.Method == http.MethodOptions {
			return
		}
		h(w, r)
	}
}

// fetchProbes 查询所有探针状态
func (s *Server) fetchProbes(ctx context.Context) []ProbeStatus {
	result, err := s.promClient.QueryInstant(ctx, "up", time.Now())
	if err != nil {
		return nil
	}

	var probes []ProbeStatus
	for _, r := range result.Data.Result {
		metric := r.Metric
		var value float64
		if len(r.Value) >= 2 {
			if v, ok := r.Value[1].(string); ok {
				value, _ = strconv.ParseFloat(v, 64)
			}
		}
		var timestamp int64
		if len(r.Value) >= 1 {
			if f, ok := r.Value[0].(float64); ok {
				timestamp = int64(f)
			}
		}

		probeType := "node"
		metricType := "node_exporter"
		probeName := metric["instance"]
		if job, ok := metric["job"]; ok {
			if job == "blackbox_http_2xx" || job == "blackbox_https_2xx" {
				probeType = "blackbox"
				metricType = "http"
				if n, ok := metric["name"]; ok {
					probeName = n
				}
			}
		}

		status := "up"
		if value != 1 {
			status = "down"
		}

		probeTarget := metric["instance"]
		if probeType == "blackbox" {
			if n, ok := metric["name"]; ok {
				probeTarget = n
			}
		}

		probes = append(probes, ProbeStatus{
			Name:       probeName,
			Type:       probeType,
			Target:     probeTarget,
			Status:     status,
			Value:      value,
			Timestamp:  timestamp,
			Instance:   metric["instance"],
			Job:        metric["job"],
			MetricType: metricType,
		})
	}
	return probes
}

// handleProbes 获取所有探针状态
func (s *Server) handleProbes(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	probes := s.fetchProbes(ctx)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"data":   probes,
	})
}

// handleTrend 获取趋势数据
func (s *Server) handleTrend(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	// 获取参数
	target := r.URL.Query().Get("target")
	if target == "" {
		target = "up"
	}

	hoursStr := r.URL.Query().Get("hours")
	hours := 24
	if h, err := strconv.Atoi(hoursStr); err == nil && h > 0 {
		hours = h
	}

	end := time.Now()
	start := end.Add(-time.Duration(hours) * time.Hour)
	step := 5 * time.Minute

	result, err := s.promClient.QueryRange(ctx, target, start, end, step)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	type seriesData struct {
		Metric map[string]string `json:"metric"`
		Data   []TrendData       `json:"data"`
	}

	var series []seriesData

	for _, r := range result.Data.Result {
		var data []TrendData
		for _, v := range r.Values {
			if len(v) >= 2 {
				var timestamp int64
				var value float64

				if f, ok := v[0].(float64); ok {
					timestamp = int64(f)
				}
				if s, ok := v[1].(string); ok {
					value, _ = strconv.ParseFloat(s, 64)
				}

				data = append(data, TrendData{
					Timestamp: timestamp,
					Value:     value,
				})
			}
		}

		series = append(series, seriesData{
			Metric: r.Metric,
			Data:   data,
		})
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"data":   series,
	})
}

// fetchAlerts 查询当前 down 的探针
func (s *Server) fetchAlerts(ctx context.Context) []AlertInfo {
	result, err := s.promClient.QueryInstant(ctx, "up == 0", time.Now())
	if err != nil {
		return nil
	}

	var alerts []AlertInfo

	for _, r := range result.Data.Result {
		metric := r.Metric
		probeType := "node"
		name := metric["instance"]

		if job, ok := metric["job"]; ok {
			if job == "blackbox_http_2xx" || job == "blackbox_https_2xx" {
				probeType = "blackbox"
				if n, ok := metric["name"]; ok {
					name = n
				}
			}
		}

		target := metric["instance"]
		if probeType == "blackbox" {
			if n, ok := metric["name"]; ok && n != target {
				target = fmt.Sprintf("%s (%s)", n, target)
			}
		}

		alerts = append(alerts, AlertInfo{
			Name:   name,
			Type:   probeType,
			Status: "down",
			Target: target,
		})
	}
	return alerts
}

// handleAlerts 获取告警信息（down 的探针）
func (s *Server) handleAlerts(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	alerts := s.fetchAlerts(ctx)
	status := "all_up"
	if len(alerts) > 0 {
		status = "has_down"
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"data":   alerts,
		"summary": map[string]interface{}{
			"total":  len(alerts),
			"status": status,
		},
	})
}

// fetchNodes 查询节点详细指标
func (s *Server) fetchNodes(ctx context.Context) []NodeMetrics {
	type queryTask struct {
		name  string
		query string
	}

	// 查询 name label 映射（用于公开展示，隐藏 IP）
	nameMap := s.queryNameMap(ctx, "up{job=~\"linux|kubernetes-nodes\"}")

	tasks := []queryTask{
		{"up", "up{job=~\"linux|kubernetes-nodes\"}"},
		{"cpu", `100 - (avg by (instance) (irate(node_cpu_seconds_total{mode="idle"}[5m])) * 100)`},
		{"memUsed", `(1 - node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes) * 100`},
		{"memTotal", `node_memory_MemTotal_bytes`},
		{"disk", `(1 - node_filesystem_avail_bytes{mountpoint="/"} / node_filesystem_size_bytes{mountpoint="/"}) * 100`},
		{"netIn", `irate(node_network_receive_bytes_total{device!~"lo|docker.*|veth.*|br.*"}[5m])`},
		{"netOut", `irate(node_network_transmit_bytes_total{device!~"lo|docker.*|veth.*|br.*"}[5m])`},
		{"load1", `node_load1`},
		{"uptime", `time() - node_boot_time_seconds`},
	}

	type result struct {
		name string
		data map[string]float64
	}

	ch := make(chan result, len(tasks))
	var wg sync.WaitGroup

	for _, t := range tasks {
		wg.Add(1)
		go func(task queryTask) {
			defer wg.Done()
			ch <- result{name: task.name, data: s.queryMetricMap(ctx, task.query)}
		}(t)
	}

	wg.Wait()
	close(ch)

	metrics := make(map[string]map[string]float64)
	for res := range ch {
		metrics[res.name] = res.data
	}

	upMap := metrics["up"]

	var nodes []NodeMetrics
	for instance, upVal := range upMap {
		status := "up"
		if upVal < 1 {
			status = "down"
		}
		nodes = append(nodes, NodeMetrics{
			Instance: instance,
			Name:     nameMap[instance],
			CPU:      metrics["cpu"][instance],
			MemUsed:  metrics["memUsed"][instance],
			MemTotal: metrics["memTotal"][instance],
			Disk:     metrics["disk"][instance],
			NetIn:    metrics["netIn"][instance],
			NetOut:   metrics["netOut"][instance],
			Load1:    metrics["load1"][instance],
			Uptime:   metrics["uptime"][instance],
			Status:   status,
		})
	}
	sort.Slice(nodes, func(i, j int) bool {
		nameI := nodes[i].Name
		if nameI == "" {
			nameI = nodes[i].Instance
		}
		nameJ := nodes[j].Name
		if nameJ == "" {
			nameJ = nodes[j].Instance
		}
		return nameI < nameJ
	})
	return nodes
}

// queryNameMap 查询指标并返回 instance -> name label 映射
func (s *Server) queryNameMap(ctx context.Context, query string) map[string]string {
	result, err := s.promClient.QueryInstant(ctx, query, time.Now())
	m := make(map[string]string)
	if err != nil {
		return m
	}
	for _, r := range result.Data.Result {
		instance := r.Metric["instance"]
		if instance == "" {
			continue
		}
		if name, ok := r.Metric["name"]; ok && name != "" {
			m[instance] = name
		}
	}
	return m
}

// queryMetricMap 查询指标并返回 instance -> value 映射
func (s *Server) queryMetricMap(ctx context.Context, query string) map[string]float64 {
	result, err := s.promClient.QueryInstant(ctx, query, time.Now())
	m := make(map[string]float64)
	if err != nil {
		return m
	}
	for _, r := range result.Data.Result {
		instance := r.Metric["instance"]
		if instance == "" {
			continue
		}
		if len(r.Value) >= 2 {
			if v, ok := r.Value[1].(string); ok {
				val, _ := strconv.ParseFloat(v, 64)
				m[instance] += val
			}
		}
	}
	return m
}

// handleNodes 获取节点详细指标
func (s *Server) handleNodes(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	nodes := s.fetchNodes(ctx)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"data":   nodes,
	})
}

// handleIndex 静态文件服务
func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(s.indexHTML)
}

// StreamPayload SSE 推送的数据结构
type StreamPayload struct {
	Probes []ProbeStatus `json:"probes"`
	Alerts []AlertInfo   `json:"alerts"`
	Nodes  []NodeMetrics `json:"nodes"`
}

// handleStream SSE 长连接，每 5 秒推送一次全量数据
func (s *Server) handleStream(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "不支持 SSE", http.StatusInternalServerError)
		return
	}

	send := func() {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		payload := StreamPayload{}

		// 并发拉取三类数据
		var wg sync.WaitGroup
		var mu sync.Mutex

		wg.Add(3)
		go func() {
			defer wg.Done()
			probes := s.fetchProbes(ctx)
			mu.Lock()
			payload.Probes = probes
			mu.Unlock()
		}()
		go func() {
			defer wg.Done()
			alerts := s.fetchAlerts(ctx)
			mu.Lock()
			payload.Alerts = alerts
			mu.Unlock()
		}()
		go func() {
			defer wg.Done()
			nodes := s.fetchNodes(ctx)
			mu.Lock()
			payload.Nodes = nodes
			mu.Unlock()
		}()
		wg.Wait()

		data, err := json.Marshal(payload)
		if err != nil {
			return
		}
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}

	// 立即推送一次
	send()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			send()
		}
	}
}
