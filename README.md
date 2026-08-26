# Solar Farm Maintenance Service

面向光伏场站维护人员的现场维护任务服务，使用 Go 标准库提供阵列任务、状态推进、遥测接入、合规检查和静态维护台。

## Endpoints

- `GET /healthz`
- `GET /api/maintenance-tasks`
- `POST /api/maintenance-tasks/{id}/status`，JSON body 为 `{"status":"scheduled|in_progress|completed"}`
- `GET /api/ops/records` 与 `POST /api/ops/records`（运维工单列表 / 创建）
- `POST /api/ops/records/{id}/transition`（工单状态推进）
- `GET /api/ops/records/{id}/audit` 与 `GET /api/ops/records/{id}/compliance`（审计事件 / 合规报告）
- `POST /api/ops/telemetry`（遥测批次接入）
- `GET /api/ops/metrics`、`GET /api/ops/schedule`、`GET /api/ops/health`
- `GET /` 与 `GET /app.js`

服务默认监听 8080，也支持 `PORT` 环境变量；维护默认设置支持 `SOLAR_SITE`、`SOLAR_MAX_BATCH`。从 `backend/` 执行 `go run .`。

## Enterprise Layout

```text
.
├── backend/              # Go module, all Go source, static assets
├── database/README.md
├── output/verification.md
├── runtime_smoke.json
├── .env.example
└── .gitignore
```

## Verification

- `gofmt -w .`、`go build ./...`、`go test ./...`：均通过
- 真实 `GET /healthz` 返回 200；维护任务与运维工单 API 均通过定向测试

## Engineering Notes

光伏维护流程代码按领域模型、校验、状态转换、并发安全存储、审计事件、遥测流水线、合规检查、通知派发和 HTTP 生命周期分层。请求会保留请求标识并经过恢复与超时保护；状态写入使用版本校验，错误通过可识别的领域错误返回。
