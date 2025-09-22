# QuickMail Service

QuickMail is a Go + Gin HTTP service that manages SMTP providers and delivers email with automatic failover between providers. Credentials are stored encrypted on disk and access to the API is protected with an optional API key header.

## Prerequisites

- Go 1.22+
- Environment variable `MAIL_CONFIG_SECRET` set to a 16, 24, or 32 byte secret used to encrypt provider credentials.
- Optional environment variable `QUICKMAIL_API_KEY` to require clients to send `X-API-Key`.
- Optional `PORT` environment variable to override the default `8080` port.

## Running

```bash
export MAIL_CONFIG_SECRET="your-32-byte-secret-here"
export QUICKMAIL_API_KEY="example-key"
go run ./cmd/quickmail
```

Provider configuration is stored in `data/providers.json` with passwords encrypted at rest. The file is created automatically on first run.

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
