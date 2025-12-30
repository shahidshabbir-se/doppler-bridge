# Doppler Bridge

A lightweight multi-tenant bridge service that automatically syncs secrets from [Doppler](https://doppler.com) to [Dokploy](https://dokploy.com). One instance handles multiple applications with path-based routing.

Built with **Go** for simplicity, performance, and easy deployment.

## Features

- ✅ **Multi-Tenant** - Single instance handles unlimited applications
- ✅ **Path-Based Routing** - Each app uses `/webhook/{app-name}`
- ✅ **Per-Service Tokens** - Each application uses its own Doppler token
- ✅ **Auto Scaling** - Single container, unlimited services
- ✅ **Secure** - Bearer token auth + optional HMAC verification
- ✅ **Cloudflare Zero Trust** - Support for protected Dokploy instances
- ✅ **Automatic Updates** - Updates env vars and triggers redeploy

## Architecture

```
ONE Doppler Bridge Instance
        │
        │ Routes based on webhook path
        ▼
   ┌────────────────────────────────────┐
   │  /webhook/meilisearch  ──────────► Meilisearch (App1)
   │  /webhook/api          ──────────► API Service (App2)  
   │  /webhook/worker       ──────────► Worker (App3)
   │  /webhook/db           ──────────► Database (App4)
   └────────────────────────────────────┘
```

## Quick Start

### Docker (Recommended)

```bash
docker pull ghcr.io/craetivohq/doppler-bridge:latest
```

### Build from Source

```bash
git clone https://github.com/craetivohq/doppler-bridge
cd doppler-bridge
go build -o doppler-bridge .
```

## Configuration

### Environment Variables

| Variable | Required | Description | Example |
|----------|----------|-------------|---------|
| `DOKPLOY_HOST` | ✅ | Dokploy instance URL | `http://100.108.216.16:3000` |
| `DOKPLOY_API_TOKEN` | ✅ | Dokploy API token | `dpl_xxxxx` |
| `WEBHOOK_SECRET` | ✅ | Bearer token for webhook auth | `your-secret-32-chars` |
| `SERVICES` | ✅ | Service mappings | See format below |
| `DOPPLER_TOKEN` | ❌ | Global Doppler token (fallback) | `dp.st.xxxxx` |
| `DOPPLER_SECRET` | ❌ | Doppler HMAC signing secret | `whsec_xxxxx` |
| `CF_ACCESS_CLIENT_ID` | ❌ | Cloudflare Access Client ID | `xxx.access` |
| `CF_ACCESS_CLIENT_SECRET` | ❌ | Cloudflare Access Client Secret | `xxx` |
| `PORT` | ❌ | Port to listen on | `3000` (default) |

### SERVICES Format

```
SERVICES=path:serviceId:serviceType:dopplerToken,path2:serviceId2:serviceType2:token2
```

**Format breakdown:**
- `path` - Webhook path (e.g., `meilisearch`, `api`, `worker`)
- `serviceId` - Dokploy application/compose ID
- `serviceType` - `application` or `compose`
- `dopplerToken` - Doppler service token for this app

**Example:**

```bash
SERVICES=meilisearch:A0cJdNYcDBokMFM5F3tRl:compose:dp.st.meilisearch-token,\
         api:abc123def456:application:dp.st.api-token,\
         worker:xyz789abc123:compose:dp.st.worker-token,\
         db:def456xyz789:application:dp.st.db-token
```

### Example Docker Compose

```yaml
version: '3.8'

services:
  doppler-bridge:
    image: ghcr.io/craetivohq/doppler-bridge:latest
    ports:
      - "3000:3000"
    environment:
      # Dokploy Configuration
      DOKPLOY_HOST: http://100.108.216.16:3000
      DOKPLOY_API_TOKEN: your-dokploy-api-token
      
      # Webhook Authentication
      WEBHOOK_SECRET: your-webhook-secret-32-chars
      
      # Cloudflare Zero Trust (if Dokploy is protected)
      CF_ACCESS_CLIENT_ID: 50d0d1da295233cb6b98062d36e101ff.access
      CF_ACCESS_CLIENT_SECRET: 55653bcf3abd8468b5f607bce358aea03b6ca87c81805259012beb10a10a5b20
      
      # Services (path:serviceId:serviceType:dopplerToken)
      SERVICES: meilisearch:A0cJdNYcDBokMFM5F3tRl:compose:dp.st.meilisearch-token,\
                api:abc123:application:dp.st.api-token,\
                worker:xyz789:compose:dp.st.worker-token
    
    restart: unless-stopped
    
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:3000/health"]
      interval: 30s
      timeout: 10s
      retries: 3
```

## Doppler Webhook Setup

Each application uses the same domain but different paths:

### Webhook URLs

```
App1 (Meilisearch) → https://bridge.yourdomain.com/webhook/meilisearch
App2 (API)         → https://bridge.yourdomain.com/webhook/api
App3 (Worker)      → https://bridge.yourdomain.com/webhook/worker
App4 (DB)          → https://bridge.yourdomain.com/webhook/db
```

### Doppler Dashboard Configuration

1. Go to **Doppler Dashboard** → **Integrations** → **Webhooks**
2. Add webhook for each application:

**App1 (Meilisearch):**
- **URL:** `https://bridge.yourdomain.com/webhook/meilisearch`
- **Custom Headers:** `Authorization: Bearer your-webhook-secret-32-chars`

**App2 (API):**
- **URL:** `https://bridge.yourdomain.com/webhook/api`
- **Custom Headers:** `Authorization: Bearer your-webhook-secret-32-chars`

...and so on for each application.

## Getting Required Tokens

### 1. Dokploy API Token

1. Log into Dokploy dashboard
2. Go to **Settings** → **API Tokens**
3. Click **Generate Token**
4. Copy the token

### 2. Dokploy Service ID

**For Application:**
- URL: `https://dokploy.example.com/project/{project-id}/services/application/{application-id}`
- Copy `{application-id}`

**For Compose:**
- URL: `https://dokploy.example.com/project/{project-id}/services/compose/{compose-id}`
- Copy `{compose-id}`
- Set `serviceType` to `compose`

### 3. Doppler Service Tokens

Create a service token for each application/environment:

```bash
# For App1 - Meilisearch (stg)
doppler configs tokens create --name "doppler-bridge-meilisearch" --config stg --plain

# For App2 - API (stg)  
doppler configs tokens create --name "doppler-bridge-api" --config stg --plain

# For App3 - Worker (stg)
doppler configs tokens create --name "doppler-bridge-worker" --config stg --plain
```

Or via Doppler Dashboard:
1. Select project/config
2. Go to **Access** → **Service Tokens**
3. Click **Generate** (read access)
4. Copy the token

### 4. Generate Webhook Secret

```bash
openssl rand -hex 32
```

## Adding New Applications

To add a new application, simply:

1. **Add the service to SERVICES environment variable:**
```bash
SERVICES=existing-services...,\
         newapp:newServiceId:application:dp.st.newapp-token
```

2. **Add Doppler webhook:**
   - **URL:** `https://bridge.yourdomain.com/webhook/newapp`
   - **Header:** `Authorization: Bearer your-webhook-secret`

3. **Redeploy** the Doppler Bridge container

No code changes needed!

## Security

### Three-Layer Security

1. **Bearer Token (Required)** - Validates webhook requests
2. **Doppler HMAC (Optional)** - Cryptographic signature verification
3. **Cloudflare Zero Trust (Optional)** - Protects Dokploy API access

### Best Practices

✅ Use 32+ character random secrets  
✅ Enable HTTPS for webhook endpoint  
✅ Use Doppler HMAC signature verification  
✅ Rotate tokens periodically  
✅ Use environment variables, never commit secrets

## Troubleshooting

### Webhook returns 401 Unauthorized

- Check `WEBHOOK_SECRET` matches in both Doppler and Doppler Bridge
- Verify Authorization header format: `Bearer <secret>`

### Service not found (404)

- Check the path matches exactly in SERVICES config
- Ensure no trailing/leading spaces in path

### No redeployment triggered

- Verify `DOKPLOY_API_TOKEN` has correct permissions
- Check `DOKPLOY_HOST` is accessible from bridge container
- Verify `serviceId` and `serviceType` are correct

### Doppler token issues

- Ensure each service has its own Doppler token
- Token must have read access to the config
- Check token format: `dp.st.xxxxx`

## Development

```bash
# Build
go build -o doppler-bridge .

# Run locally
go run main.go

# Test
go test ./...
```

## License

MIT

## Links

- [Doppler Docs](https://docs.doppler.com)
- [Dokploy Docs](https://docs.dokploy.com)
- [Cloudflare Zero Trust](https://developers.cloudflare.com/cloudflare-one/)