# Aura 探针监控面板

基于 Prometheus 的探针监控面板，支持 Blackbox HTTP/HTTPS 探针和 Node Exporter 监控。

## 快速安装

```bash
curl -sfL https://raw.githubusercontent.com/ablate-ai/aura/main/install.sh | sh -
```

### 自定义配置

```bash
# 自定义 Prometheus 地址
curl -sfL https://raw.githubusercontent.com/ablate-ai/aura/main/install.sh | INSTALL_BASEURL=http://your-prom:9090 sh -

# 自定义监听端口
curl -sfL https://raw.githubusercontent.com/ablate-ai/aura/main/install.sh | INSTALL_PORT=3000 sh -
```

## 配置

配置文件位于 `/etc/aura/env`：

```bash
# Prometheus 地址
PROM_BASEURL="http://prom.ooxo.cc/"

# 监听端口
PORT="8080"
```

## 使用

### Systemd 服务

```bash
# 启动
systemctl start aura

# 开机自启
systemctl enable aura

# 查看状态
systemctl status aura

# 查看日志
journalctl -u aura -f
```

### 直接运行

```bash
# 使用默认配置
aura

# 自定义配置
PROM_BASEURL=http://your-prom/ PORT=3000 aura

# 查看版本
aura -version
```

## 功能

- **状态总览**：所有探针实时状态，绿色/红色指示
- **告警面板**：展示所有 down 状态的探针
- **探针分类**：Blackbox HTTP 探针、Node Exporter
- **历史趋势**：1小时/6小时/24小时/7天趋势图
- **自动刷新**：每 30 秒更新数据
- **响应式设计**：支持移动端

## API

| 端点 | 说明 |
|------|------|
| `GET /` | Web 监控面板 |
| `GET /api/probes` | 探针状态 |
| `GET /api/trend?hours=N` | 历史趋势 |
| `GET /api/alerts` | 告警列表 |

## 从二进制运行

下载最新 release：https://github.com/ablate-ai/aura/releases

```bash
# Linux amd64
wget https://github.com/ablate-ai/aura/releases/latest/download/aura_Linux_x86_64.tar.gz
tar -xzf aura_Linux_x86_64.tar.gz
sudo mv aura /usr/local/bin/

# 运行
PROM_BASEURL=http://your-prom/ aura
```

## License

MIT
