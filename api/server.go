package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/c.chen/aura/client"
	"github.com/c.chen/aura/config"
)

// Server HTTP API 服务器
type Server struct {
	promClient *client.Client
	httpServer *http.Server
}

// NewServer 创建 API 服务器
func NewServer(cfg *config.Config, addr string) *Server {
	promClient := client.NewClient(cfg)

	return &Server{
		promClient: promClient,
		httpServer: &http.Server{
			Addr:         addr,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
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
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	Target    string    `json:"target"`
	Status    string    `json:"status"`
	StartTime time.Time `json:"startTime"`
	Duration  string    `json:"duration"`
}

// Start 启动服务器
func (s *Server) Start() error {
	// 注册路由
	http.HandleFunc("/api/probes", s.cors(s.handleProbes))
	http.HandleFunc("/api/trend", s.cors(s.handleTrend))
	http.HandleFunc("/api/alerts", s.cors(s.handleAlerts))
	http.HandleFunc("/", s.handleIndex)

	log.Printf("服务器启动在 http://%s", s.httpServer.Addr)
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

// handleProbes 获取所有探针状态
func (s *Server) handleProbes(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// 查询 up 指标
	result, err := s.promClient.QueryInstant(ctx, "up", time.Now())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
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

		// 判断探针类型
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

		// 判断状态
		status := "up"
		if value != 1 {
			status = "down"
		}

		probeTarget := metric["instance"]

		// 对于 blackbox，使用 name 作为主要标识
		if probeType == "blackbox" {
			if n, ok := metric["name"]; ok {
				probeTarget = n
			}
		}

		probe := ProbeStatus{
			Name:       probeName,
			Type:       probeType,
			Target:     probeTarget,
			Status:     status,
			Value:      value,
			Timestamp:  timestamp,
			Instance:   metric["instance"],
			Job:        metric["job"],
			MetricType: metricType,
		}

		probes = append(probes, probe)
	}

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

// handleAlerts 获取告警信息（down 的探针）
func (s *Server) handleAlerts(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// 查询当前 down 的探针
	result, err := s.promClient.QueryInstant(ctx, "up == 0", time.Now())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var alerts []AlertInfo
	now := time.Now()

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

		// 设置目标
		target := metric["instance"]
		if probeType == "blackbox" {
			if n, ok := metric["name"]; ok && n != target {
				target = fmt.Sprintf("%s (%s)", n, target)
			}
		}

		// 查询第一次 down 的时间（简化处理，使用 1 小时前作为估算）
		alert := AlertInfo{
			Name:      name,
			Type:      probeType,
			Status:    "down",
			StartTime: now.Add(-1 * time.Hour), // 简化处理
			Duration:  "1h+",
			Target:    target,
		}

		alerts = append(alerts, alert)
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"data":    alerts,
		"summary": map[string]interface{}{
			"total":  len(alerts),
			"status": func() string {
				if len(alerts) == 0 {
					return "all_up"
				}
				return "has_down"
			}(),
		},
	})
}

// handleIndex 静态文件服务
func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "web/index.html")
}
