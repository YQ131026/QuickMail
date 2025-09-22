# QuickMail Service

QuickMail is a Go + Gin HTTP service that manages SMTP providers and delivers email with automatic failover between providers. Credentials are stored encrypted on disk and access to the API is protected with an optional API key header.

## Prerequisites

- Go 1.22+
- 配置 `config/config.json`，为服务提供所有运行所需的设置。

## Running

1. 编辑 `config/config.json`，确保 `secret` 为 16/24/32 字节，用于加密 SMTP 凭证。
2. 运行服务：

```bash
go run ./cmd/quickmail
```

若需要使用其他配置文件路径，可设置环境变量 `QUICKMAIL_CONFIG_FILE` 指向新的 JSON 文件。

Provider configuration is stored in `data/providers.json` with passwords encrypted at rest. The file is created automatically on first run.

基础配置位于 `config/config.json`，示例：

```json
{
  "provider_store_path": "data/providers.json",
  "api_key": "",
  "secret": "0123456789abcdef0123456789abcdef",
  "port": "8080"
}
```

## API Overview

- `GET /health` — Service heartbeat.
- `GET /health/providers/:name` — Checks SMTP connectivity for a provider.
- `GET /providers` — Lists configured providers (without passwords).
- `POST /providers` — Upserts an SMTP provider.
- `DELETE /providers/:name` — Removes a provider.
- `POST /send` — Sends an email; falls back to secondary providers when the primary fails.

### Provider payload

```json
{
  "name": "gmail",
  "host": "smtp.gmail.com",
  "port": 587,
  "username": "user@gmail.com",
  "password": "app-password",
  "from": "user@gmail.com",
  "use_tls": true
}
```

### Send email payload

```json
{
  "subject": "Hello",
  "body": "<strong>Welcome!</strong>",
  "is_html": true,
  "to": ["recipient@example.com"],
  "attachments": [
    {
      "filename": "example.txt",
      "content": "YmFzZTY0LWVuY29kZWQgZmlsZQ==",
      "content_type": "text/plain"
    }
  ],
  "provider_priority": ["gmail", "backup"],
  "from": "user@gmail.com"
}
```

If `provider_priority` is omitted the service tries providers in the order returned by `GET /providers`.

## Logging

The service logs to stdout with the prefix `quickmail` and records successes and failures per provider.

## Development

- Format: `gofmt -w ./...`
- Build: `go build ./...`
- Test: `go test ./...`
