# ScrapeHub

ScrapeHub 是通用指标采集代理：通过静态配置与服务发现维护采集目标，
调度器按周期错峰派发抓取任务，抓取器拉取指标文本，解析器流式解析，
转发器按时间序批量写入下游存储，并持续维护目标健康状态与重试背压。

## 构建

```bash
./build_benzhi_docker.sh scrapehub linux/amd64
./build_benzhi_docker.sh scrapehub linux/arm64
```

## 运行

```bash
go run ./cmd/scrapehub -addr :8080
```

启动后打开 http://localhost:8080/monitor 查看采集监控页面。

## 接口

- `GET /healthz` 健康检查
- `GET /api/v1/targets` 目标列表
- `POST /api/v1/targets` 注册目标
- `DELETE /api/v1/targets/{id}` 移除目标
- `POST /api/v1/cycle` 触发一轮采集
- `GET /api/v1/metrics` 运行指标
- `GET /api/v1/health/{id}` 目标健康状态
- `GET /monitor` 采集监控页面

## 容器内验证

```bash
go build ./...
go test ./...
go vet ./...
go run ./cmd/scrapehub
```
