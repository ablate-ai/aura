# Aura 探针监控面板

<p align="center">
  <strong>基于 Prometheus 的探针监控面板</strong>
</p>

<p align="center">
  <a href="https://github.com/ablate-ai/aura/releases"><img src="https://img.shields.io/github/release/ablate-ai/aura" alt="Release"></a>
  <a href="https://github.com/ablate-ai/aura/blob/main/LICENSE"><img src="https://img.shields.io/github/license/ablate-ai/aura" alt="License"></a>
</p>

---

## 简介

Aura 是一个轻量级的探针监控面板，用于展示 Prometheus 监控的探针状态。支持：

- **Blackbox Exporter** - HTTP/HTTPS 探针监控
- **Node Exporter** - 服务器状态监控
- **其他 up 指标** - 任何基于 `up` 指标的服务

## 特性

- 一键安装，开箱即用
- 实时状态监控（30秒自动刷新）
- 告警面板，快速定位故障
- 历史趋势分析
- 响应式设计，支持移动端
- 纯静态页面，无额外依赖
- 多平台支持（Linux/macOS/Windows）

## 快速开始

### 一键安装

```bash
sh -c "$(curl -sfL https://raw.githubusercontent.com/ablate-ai/aura/main/install.sh)"
```

访问 http://localhost:8080 查看监控面板。

### 国内加速

如果下载慢，直接用镜像地址：

```bash
sh -c "$(curl -sfL https://ghfast.top/https://raw.githubusercontent.com/ablate-ai/aura/main/install.sh)"
```

### 自定义配置

```bash
# 指定 Prometheus 地址
PROM_BASEURL=http://your-prom:9090 sh -c "$(curl -sfL https://raw.githubusercontent.com/ablate-ai/aura/main/install.sh)"

# 指定监听端口
PORT=3000 sh -c "$(curl -sfL https://raw.githubusercontent.com/ablate-ai/aura/main/install.sh)"
```

## 配置

### 环境变量

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `PROM_BASEURL` | Prometheus 地址 | `http://prom.ooxo.cc/` |
| `PORT` | 监听端口 | `8080` |

### 配置文件

安装后会创建 `/etc/aura/env`：

```bash
# Prometheus 地址
PROM_BASEURL="http://prom.ooxo.cc/"

# 监听端口
PORT="8080"
```

## 使用

### Systemd 服务

```bash
systemctl start aura    # 启动服务
systemctl enable aura   # 开机自启
systemctl status aura   # 查看状态
journalctl -u aura -f   # 查看日志
```

### 直接运行

```bash
# 默认配置
aura

# 自定义配置
PROM_BASEURL=http://your-prom/ PORT=3000 aura

# 查看版本
aura -version
```

## 功能预览

### 状态总览

- 实时展示所有探针状态
- 绿色/红色指示灯
- UP/DOWN 数量统计

### 告警面板

- 顶部展示所有故障探针
- 快速定位问题

### 探针分类

- **HTTP 探针** - Blackbox 监控的网站/API
- **Node 探针** - 服务器 Node Exporter

### 历史趋势

- 1小时 / 6小时 / 24小时 / 7天
- 可用性百分比曲线
- 基于 Chart.js 绘制

## API

| 端点 | 说明 |
|------|------|
| `GET /` | Web 监控面板 |
| `GET /api/probes` | 获取所有探针状态 |
| `GET /api/trend?hours=N` | 获取历史趋势（N=1,6,24,168） |
| `GET /api/alerts` | 获取告警列表 |

### API 响应示例

```json
// GET /api/probes
{
  "status": "success",
  "data": [
    {
      "name": "个人博客",
      "type": "blackbox",
      "target": "个人博客",
      "status": "up",
      "value": 1,
      "timestamp": 1772117829,
      "instance": "https://zzfzzf.com",
      "job": "blackbox_http_2xx",
      "metricType": "http"
    }
  ]
}
```

## 手动安装

### 下载二进制

访问 [Releases](https://github.com/ablate-ai/aura/releases) 下载对应平台的二进制文件。

```bash
# Linux amd64
wget https://github.com/ablate-ai/aura/releases/latest/download/aura_Linux_x86_64.tar.gz
tar -xzf aura_Linux_x86_64.tar.gz
sudo mv aura /usr/local/bin/
sudo chmod +x /usr/local/bin/aura

# 运行
PROM_BASEURL=http://your-prom/ aura
```

### Docker 运行

```bash
docker run -d \
  --name aura \
  -p 8080:8080 \
  -e PROM_BASEURL=http://your-prom:9090 \
  ghcr.io/ablate-ai/aura:latest
```

## 开发

```bash
# 克隆仓库
git clone https://github.com/ablate-ai/aura.git
cd aura

# 运行
go run main.go

# 编译
go build
```

## 常见问题

### 1. 无法连接 Prometheus

检查 `PROM_BASEURL` 是否正确配置：

```bash
curl http://your-prom:9090/api/v1/query?query=up
```

### 2. 页面显示空白

检查浏览器控制台是否有错误，确保 API 可访问：

```bash
curl http://localhost:8080/api/probes
```

### 3. 探针数量为 0

检查 Prometheus 是否有 `up` 指标：

```bash
curl -s 'http://your-prom:9090/api/v1/query?query=up' | jq '.data.result | length'
```

## 许可证

[MIT](LICENSE)

---

<p align="center">
  Made with ❤️ by <a href="https://github.com/ablate-ai">ablate-ai</a>
</p>
