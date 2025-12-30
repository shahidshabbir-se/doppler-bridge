# Doppler Bridge

A lightweight bridge service that automatically syncs secrets from [Doppler](https://doppler.com) to [Dokploy](https://dokploy.com). When secrets change in Doppler, this service receives a webhook, fetches the latest secrets, and updates your Dokploy application or compose service with automatic redeployment.

Built with **Go** for simplicity, performance, and easy deployment.

## Features

- ✅ Automatic secret synchronization from Doppler to Dokploy
- ✅ Support for both **Application** and **Docker Compose** deployments
- ✅ Secure webhook authentication with bearer token
- ✅ Optional Doppler webhook signature verification (HMAC-SHA256)
- ✅ **Cloudflare Zero Trust support** for secure Dokploy access
- ✅ Automatic redeployment after secret updates
- ✅ Simple configuration via environment variables or CLI flags
- ✅ Single binary deployment - no dependencies
- ✅ Multi-platform Docker images (amd64, arm64)

## How It Works

```
┌─────────┐    webhook     ┌────────────────┐    fetch secrets    ┌─────────┐
│ Doppler │ ─────────────> │ Doppler Bridge │ ──────────────────> │ Doppler │
└─────────┘                └────────────────┘                     └─────────┘
                                   │
                                   │ 1. Update env
                                   │ 2. Redeploy
                                   ▼
                           ┌────────────────┐
                           │    Dokploy     │
                           │ (via ZeroTrust)│
                           └────────────────┘
```

1. Doppler sends a webhook when secrets are updated
2. Bridge validates request (bearer token + optional HMAC signature)
3. Fetches latest secrets from Doppler API
4. Updates environment variables in Dokploy (through Cloudflare Zero Trust)
5. Triggers automatic redeployment

## Quick Start

### Installation

**Option 1: Docker (Recommended)**
```bash
docker pull ghcr.io/craetivohq/doppler-bridge:latest
```

**Option 2: Build from Source**
```bash
git clone https://github.com/craetivohq/doppler-bridge
cd doppler-bridge
go build -o doppler-bridge .
```

## Configuration

### Required Environment Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `DOKPLOY_HOST` | Your Dokploy instance URL | `https://dokploy.example.com` |
| `DOKPLOY_API_TOKEN` | API token for Dokploy | `dpl_xxxxx` |
| `DOKPLOY_APPLICATION_ID` | Application or Compose ID | `abc123xyz` |
| `DOPPLER_TOKEN` | Doppler service token | `dp.st.xxxxx` |
| `WEBHOOK_SECRET` | Secret token for webhook auth | `your-random-secret-32-chars` |

### Optional Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `3000` | Port to listen on |
| `DOKPLOY_SERVICE_TYPE` | `application` | Service type: `application` or `compose` |
| `DOPPLER_SECRET` | - | Doppler webhook signing secret (HMAC) |
| `CF_ACCESS_CLIENT_ID` | - | Cloudflare Access Client ID |
| `CF_ACCESS_CLIENT_SECRET` | - | Cloudflare Access Client Secret |

## Step-by-Step Setup

### 1. Get Dokploy API Token

1. Log into your Dokploy dashboard
2. Go to **Settings** > **API Tokens**
3. Click **Generate Token**
4. Save as `DOKPLOY_API_TOKEN`

### 2. Get Dokploy Application/Compose ID

**For Application:**
- URL: `https://dokploy.example.com/project/{project-id}/services/application/{application-id}`
- Copy `{application-id}`

**For Compose:**
- URL: `https://dokploy.example.com/project/{project-id}/services/compose/{compose-id}`
- Copy `{compose-id}`
- Set `DOKPLOY_SERVICE_TYPE=compose`

### 3. Get Doppler Service Token

**Via CLI:**
```bash
doppler configs tokens create --name "doppler-bridge" --plain
```

**Via Dashboard:**
1. Go to [Doppler Dashboard](https://dashboard.doppler.com)
2. Select your project and config
3. Go to **Access** > **Service Tokens**
4. Click **Generate** (read access)
5. Save as `DOPPLER_TOKEN`

### 4. Generate Webhook Secret

```bash
openssl rand -hex 32
```

Save as `WEBHOOK_SECRET`.

### 5. Setup Cloudflare Zero Trust (If Your Dokploy is Protected)

#### Create Service Token in Cloudflare

1. Go to [Cloudflare Zero Trust Dashboard](https://one.dash.cloudflare.com/)
2. Navigate to **Access** > **Service Auth** > **Service Tokens**
3. Click **Create Service Token**
4. Name it: `doppler-bridge`
5. Copy the **Client ID** → save as `CF_ACCESS_CLIENT_ID`
6. Copy the **Client Secret** → save as `CF_ACCESS_CLIENT_SECRET`

#### Add Service Token to Access Policy

1. Go to **Access** > **Applications**
2. Find your Dokploy application
3. Click **Edit** > **Policies**
4. Add a new policy or edit existing:
   - **Policy name**: `Allow Doppler Bridge`
   - **Action**: `Service Auth`
   - **Include**: Select your `doppler-bridge` service token
5. Save

### 6. Setup Doppler Webhook

1. In Doppler dashboard, go to **Integrations** > **Webhooks**
2. Click **Add Webhook**
3. Configure:
   - **URL**: `https://your-bridge-host.com/webhook`
   - **Custom Headers**: 
     ```
     Authorization: Bearer your-webhook-secret
     ```
   - **Signing Secret** (optional but recommended): Generate one and save as `DOPPLER_SECRET`
4. Select projects/configs to trigger on
5. Save

## Running the Service

### Using Docker Compose

Create `docker-compose.yml`:

```yaml
version: '3.8'

services:
  doppler-bridge:
    image: ghcr.io/craetivohq/doppler-bridge:latest
    ports:
      - "3000:3000"
    environment:
      # Dokploy configuration
      DOKPLOY_HOST: https://dokploy.example.com
      DOKPLOY_API_TOKEN: dpl_xxxxx
      DOKPLOY_APPLICATION_ID: abc123
      DOKPLOY_SERVICE_TYPE: application
      
      # Doppler configuration
      DOPPLER_TOKEN: dp.st.xxxxx
      DOPPLER_SECRET: whsec_xxxxx
      
      # Webhook authentication
      WEBHOOK_SECRET: your-webhook-secret
      
      # Cloudflare Zero Trust (if needed)
      CF_ACCESS_CLIENT_ID: your-client-id
      CF_ACCESS_CLIENT_SECRET: your-client-secret
      
      PORT: 3000
    
    restart: unless-stopped
    
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:3000/health"]
      interval: 30s
      timeout: 10s
      retries: 3
```

Run:
```bash
docker-compose up -d
```

### Using Docker CLI

```bash
docker run -d \
  --name doppler-bridge \
  -p 3000:3000 \
  -e DOKPLOY_HOST="https://dokploy.example.com" \
  -e DOKPLOY_API_TOKEN="dpl_xxxxx" \
  -e DOKPLOY_APPLICATION_ID="abc123" \
  -e DOKPLOY_SERVICE_TYPE="application" \
  -e DOPPLER_TOKEN="dp.st.xxxxx" \
  -e WEBHOOK_SECRET="your-secret" \
  -e CF_ACCESS_CLIENT_ID="your-client-id" \
  -e CF_ACCESS_CLIENT_SECRET="your-client-secret" \
  --restart unless-stopped \
  ghcr.io/craetivohq/doppler-bridge:latest
```

### Local Development

```bash
export DOKPLOY_HOST="https://dokploy.example.com"
export DOKPLOY_API_TOKEN="dpl_xxxxx"
export DOKPLOY_APPLICATION_ID="abc123"
export DOPPLER_TOKEN="dp.st.xxxxx"
export WEBHOOK_SECRET="your-secret"
export CF_ACCESS_CLIENT_ID="your-client-id"
export CF_ACCESS_CLIENT_SECRET="your-client-secret"

./doppler-bridge
```

## API Endpoints

### `POST /webhook`
Receives Doppler webhook events.

**Headers:**
- `Authorization: Bearer <WEBHOOK_SECRET>` (required)
- `X-Doppler-Signature: sha256=<hmac>` (optional, verified if `DOPPLER_SECRET` is set)

**Response:**
- `200 OK` - Success
- `401 Unauthorized` - Invalid auth or signature
- `500 Internal Server Error` - Processing failed

### `GET /health`
Health check endpoint. No authentication required.

**Response:**
- `200 OK`

## Security

### Three-Layer Security Model

1. **Webhook Bearer Token** (Required)
   - Simple token-based authentication
   - Validates incoming webhook requests
   - Set via `WEBHOOK_SECRET`

2. **Doppler HMAC Signature** (Optional, Recommended)
   - Cryptographic verification using SHA-256
   - Ensures webhook genuinely from Doppler
   - Prevents replay attacks
   - Set via `DOPPLER_SECRET`

3. **Cloudflare Zero Trust** (Optional)
   - Protects Dokploy API access
   - Service token authentication
   - Set via `CF_ACCESS_CLIENT_ID` and `CF_ACCESS_CLIENT_SECRET`

### Best Practices

✅ Use strong random tokens (32+ characters)  
✅ Enable HTTPS for your bridge endpoint  
✅ Set `DOPPLER_SECRET` for signature verification  
✅ Use Cloudflare Zero Trust for Dokploy protection  
✅ Rotate secrets periodically  
✅ Use environment variables, never commit secrets  
✅ Run bridge in a private network when possible  
✅ Monitor logs for unauthorized access attempts  

## Troubleshooting

### Webhook not triggering

**Check:**
- Correct `WEBHOOK_SECRET` in Doppler webhook headers
- Webhook URL is accessible from internet
- Check logs: `docker logs doppler-bridge`

### Signature verification failing

**Check:**
- `DOPPLER_SECRET` matches Doppler webhook signing secret exactly
- `X-Doppler-Signature` header is present in webhook

### Dokploy API errors

**Check:**
- `DOKPLOY_API_TOKEN` is valid and has permissions
- `DOKPLOY_APPLICATION_ID` or compose ID is correct
- `DOKPLOY_SERVICE_TYPE` matches (application vs compose)

### Cloudflare Zero Trust blocking requests

**Check:**
- `CF_ACCESS_CLIENT_ID` and `CF_ACCESS_CLIENT_SECRET` are correct
- Service token is added to Access policy
- Service token hasn't expired
- Check Cloudflare Access logs

**Test Cloudflare Access:**
```bash
curl -H "CF-Access-Client-Id: your-id" \
     -H "CF-Access-Client-Secret: your-secret" \
     -H "x-api-key: your-dokploy-token" \
     https://dokploy.example.com/api/application.one?applicationId=abc123
```

## CLI Flags

All configuration can be done via CLI flags:

```bash
./doppler-bridge \
  --port 3000 \
  --dokploy-host https://dokploy.example.com \
  --dokploy-api-token dpl_xxxxx \
  --dokploy-application-id abc123 \
  --dokploy-service-type application \
  --doppler-token dp.st.xxxxx \
  --doppler-secret whsec_xxxxx \
  --webhook-secret your-secret \
  --cf-access-client-id your-id \
  --cf-access-client-secret your-secret
```

View all options:
```bash
./doppler-bridge --help
```

## Development

```bash
# Clone repository
git clone https://github.com/craetivohq/doppler-bridge
cd doppler-bridge

# Install dependencies
go mod download

# Run locally
go run main.go

# Build
go build -o doppler-bridge .

# Build for specific platform
GOOS=linux GOARCH=amd64 go build -o doppler-bridge-linux-amd64 .
GOOS=linux GOARCH=arm64 go build -o doppler-bridge-linux-arm64 .

# Build Docker image
docker build -t doppler-bridge .

# Run tests
go test ./...
```

## Examples

### Example 1: Basic Application

```bash
docker run -d \
  -p 3000:3000 \
  -e DOKPLOY_HOST="https://dokploy.example.com" \
  -e DOKPLOY_API_TOKEN="dpl_xxxxx" \
  -e DOKPLOY_APPLICATION_ID="my-app-id" \
  -e DOPPLER_TOKEN="dp.st.xxxxx" \
  -e WEBHOOK_SECRET="$(openssl rand -hex 32)" \
  ghcr.io/craetivohq/doppler-bridge:latest
```

### Example 2: Docker Compose with Full Security

```bash
docker run -d \
  -p 3000:3000 \
  -e DOKPLOY_HOST="https://dokploy.example.com" \
  -e DOKPLOY_API_TOKEN="dpl_xxxxx" \
  -e DOKPLOY_APPLICATION_ID="my-compose-id" \
  -e DOKPLOY_SERVICE_TYPE="compose" \
  -e DOPPLER_TOKEN="dp.st.xxxxx" \
  -e DOPPLER_SECRET="whsec_xxxxx" \
  -e WEBHOOK_SECRET="$(openssl rand -hex 32)" \
  -e CF_ACCESS_CLIENT_ID="cf_client_id" \
  -e CF_ACCESS_CLIENT_SECRET="cf_client_secret" \
  ghcr.io/craetivohq/doppler-bridge:latest
```

## License

MIT

## Links

- **Doppler Docs**: https://docs.doppler.com
- **Dokploy Docs**: https://docs.dokploy.com
- **Cloudflare Zero Trust**: https://developers.cloudflare.com/cloudflare-one/
- **Issues**: https://github.com/craetivohq/doppler-bridge/issues
