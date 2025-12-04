# Go WhatsApp Multi-Session REST API

[![release version](https://img.shields.io/github/v/release/gdbrns/go-whatsapp-multi-session-rest-api)](https://github.com/gdbrns/go-whatsapp-multi-session-rest-api/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/gdbrns/go-whatsapp-multi-session-rest-api)](https://goreportcard.com/report/github.com/gdbrns/go-whatsapp-multi-session-rest-api)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![whatsmeow](https://img.shields.io/badge/whatsmeow-v0.0.0--20251127-brightgreen.svg)](https://pkg.go.dev/go.mau.fi/whatsmeow@v0.0.0-20251127132918-b9ac3d51d746)

> A minimal REST API for WhatsApp Multi-Device and Multi-Session implementation built with Go and **[whatsmeow v0.0.0-20251127](https://pkg.go.dev/go.mau.fi/whatsmeow@v0.0.0-20251127132918-b9ac3d51d746)**. Supports multiple accounts and devices simultaneously with efficient memory use and production-ready deployments.

**📚 API Documentation:** Interactive Swagger UI at `/docs/` when running

## 📋 Table of Contents

- [Features](#-features)
- [Requirements](#-requirements)
- [Installation](#-installation)
- [Configuration](#-configuration)
- [How to Use](#-how-to-use)
- [API Endpoints](#-api-endpoints)
- [Authentication Flow](#-authentication-flow)
- [Environment Variables](#-environment-variables)
- [Contributing](#-contributing)
- [License](#-license)
- [Disclaimer](#-disclaimer)

## ✨ Features

- 🔐 **Multi-Session Support** - Handle multiple WhatsApp accounts simultaneously
- 📱 **Multi-Device Support** - Up to 4 devices per WhatsApp account
- 🎫 **JWT Token Authentication** - Stateless, high-performance authentication for 1000+ sessions
- 📨 **Core Messaging** - Text messages, images, and documents
- 🌳 **RESTful Architecture** - Hierarchical resource-based routes
- 👥 **Group Management** - Full CRUD for groups, participants, admins, and settings
- 🔄 **Message Operations** - Edit, react, delete, reply, mark read
- 📊 **Presence & Status** - Online status, typing indicators, disappearing messages
- 🔄 **App State Synchronization** - Fetch, send, and manage app state patches
- 🪝 **Webhook Integration** - Real-time event notifications with retry support
- 🏗️ **Production Ready** - Docker support, environment configuration, logging
- 📖 **OpenAPI/Swagger** - Interactive API documentation at `/docs/`
- 🔑 **Admin Dashboard Ready** - Manage API keys and devices with admin endpoints
- 📈 **Admin Dashboard APIs** - System stats, health monitoring, device status, batch reconnect

## 📋 Requirements

### System Requirements

- **Go**: 1.19 or higher (for building from source)
- **PostgreSQL**: Primary datastore for sessions and app metadata
- **FFmpeg**: For media processing (optional but recommended)

### Platform Support

- ✅ Linux (x86_64, ARM64)
- ✅ macOS (Intel, Apple Silicon)
- ✅ Windows (x86_64) - WSL recommended for better compatibility

### Dependencies

#### Mac OS
```bash
brew install ffmpeg
export CGO_CFLAGS_ALLOW="-Xpreprocessor"
```

#### Linux (Debian/Ubuntu)
```bash
sudo apt update
sudo apt install -y ffmpeg postgresql-client
```

#### Windows
- Install FFmpeg from [official website](https://www.ffmpeg.org/download.html#build-windows)
- Add FFmpeg to environment PATH
- **Note**: WSL (Windows Subsystem for Linux) is recommended

## 🚀 Installation

### Using Docker (Recommended)

#### Docker Compose
```yaml
version: '3.8'

services:
  whatsapp-api:
    image: gdbrns/go-whatsapp-multi-session-rest-api
    container_name: whatsapp-api
    restart: always
    ports:
      - "7001:7001"
    environment:
      - ADMIN_SECRET_KEY=your-super-secret-admin-key-change-this
      - JWT_SECRET_KEY=your-jwt-secret-key-at-least-32-chars
      - SERVER_PORT=7001
      - WHATSAPP_DATASTORE_TYPE=postgres
      - WHATSAPP_DATASTORE_URI=postgres://whatsapp:secret@postgres:5432/whatsapp?sslmode=disable

  postgres:
    image: postgres:15
    environment:
      - POSTGRES_USER=whatsapp
      - POSTGRES_PASSWORD=secret
      - POSTGRES_DB=whatsapp
    volumes:
      - postgres-data:/var/lib/postgresql/data

volumes:
  postgres-data:
```

Then run:
```bash
docker-compose up -d
```

### From Source

```bash
# Clone repository
git clone https://github.com/gdbrns/go-whatsapp-multi-session-rest-api.git
cd go-whatsapp-multi-session-rest-api

# Install dependencies
go mod download

# Configure environment
cp .env.example .env
# Edit .env with your settings

# Build
go build -o whatsapp-api cmd/main/main.go

# Run
./whatsapp-api
```

## ⚙️ Configuration

Configuration can be set via:
1. **Environment variables** (highest priority)
2. **`.env` file** in the application directory
3. **Default values** (lowest priority)

See [Environment Variables](#-environment-variables) section for all options.

## 📖 How to Use

### 1. Start the Application

```bash
# Using Docker
docker-compose up -d

# From source
go run cmd/main/main.go
```

### 2. Access API Documentation

Open your browser and navigate to:
- **Swagger UI**: `http://localhost:7001/docs/`
- **OpenAPI JSON**: `http://localhost:7001/docs/swagger.json`
- **OpenAPI YAML**: `http://localhost:7001/docs/swagger.yaml`

### 3. Create an API Key (Admin)

```bash
# Create a new API key for a customer
curl -X POST "http://localhost:7001/admin/api-keys" \
  -H "X-Admin-Secret: YOUR_ADMIN_SECRET" \
  -H "Content-Type: application/json" \
  -d '{
    "customer_name": "Acme Corp",
    "max_devices": 10
  }'

# Response:
# {
#   "status": true,
#   "code": 201,
#   "data": {
#     "api_key": "wam_a1b2c3d4e5f6g7h8..."
#   }
# }
```

### 4. Create a Device (Customer)

```bash
# Create a new device using API key
curl -X POST "http://localhost:7001/devices" \
  -H "X-API-Key: wam_a1b2c3d4e5f6g7h8..." \
  -H "Content-Type: application/json" \
  -d '{
    "device_name": "Production Bot 1"
  }'

# Response (SAVE BOTH device_secret AND token!):
# {
#   "device_id": "550e8400-e29b-41d4-a716-446655440000",
#   "device_secret": "a1b2c3d4e5f6g7h8...",
#   "token": "eyJhbGciOiJIUzI1NiIs...",
#   "message": "Use the token in Authorization header for all API calls."
# }
```

### 5. Login via QR Code

```bash
# Generate QR code for device login (using JWT token)
curl -X POST "http://localhost:7001/devices/me/login" \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIs..." \
  -F "output=json"

# Scan the QR code with your WhatsApp mobile app:
# WhatsApp > Settings > Linked Devices > Link a Device
```

### 6. Send a Message

```bash
# Send text message (using JWT token)
curl -X POST "http://localhost:7001/chats/6281234567890@s.whatsapp.net/messages" \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIs..." \
  -H "Content-Type: application/json" \
  -d '{
    "text": "Hello from WhatsApp API!"
  }'

# Send image
curl -X POST "http://localhost:7001/chats/6281234567890@s.whatsapp.net/images" \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIs..." \
  -F "file=@/path/to/image.jpg" \
  -F "caption=Check this out!"
```

### 7. Regenerate Token (When Needed)

```bash
# Regenerate JWT token using device credentials
# This invalidates all previous tokens
curl -X POST "http://localhost:7001/devices/token" \
  -H "Content-Type: application/json" \
  -d '{
    "device_id": "550e8400-e29b-41d4-a716-446655440000",
    "device_secret": "a1b2c3d4e5f6g7h8..."
  }'

# Response:
# {
#   "device_id": "550e8400-e29b-41d4-a716-446655440000",
#   "token": "eyJhbGciOiJIUzI1NiIs...(NEW TOKEN)",
#   "message": "Token regenerated successfully. All previous tokens are now invalid."
# }
```

### 8. Webhook Configuration

```bash
# Create a webhook
curl -X POST "http://localhost:7001/webhooks" \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIs..." \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://your-server.com/webhook",
    "events": ["message.received", "connection.connected"]
  }'
```

## 📚 API Endpoints

### All Endpoints

**Organized by category** with most commonly used endpoints first.

| # | Method | Endpoint | Auth | Description |
|---|:------:|----------|:----:|-------------|
| | | **Admin Dashboard** | | |
| 1 | GET | `/admin/stats` | Admin | Get system statistics |
| 2 | GET | `/admin/health` | Admin | Get system health info |
| 3 | GET | `/admin/devices` | Admin | List all devices (all API keys) |
| 4 | GET | `/admin/devices/status` | Admin | Get live connection status for all devices |
| 5 | POST | `/admin/devices/reconnect` | Admin | Reconnect all disconnected devices |
| 6 | GET | `/admin/webhooks/stats` | Admin | Get webhook delivery statistics |
| | | **Admin (API Key Management)** | | |
| 7 | POST | `/admin/api-keys` | Admin | Create API key |
| 8 | GET | `/admin/api-keys` | Admin | List all API keys |
| 9 | GET | `/admin/api-keys/{id}` | Admin | Get API key details |
| 10 | PATCH | `/admin/api-keys/{id}` | Admin | Update API key |
| 11 | DELETE | `/admin/api-keys/{id}` | Admin | Delete API key |
| 12 | GET | `/admin/api-keys/{id}/devices` | Admin | List devices for API key |
| 13 | DELETE | `/admin/devices/{device_id}` | Admin | Delete a device |
| | | **Device Creation & Token** | | |
| 14 | POST | `/devices` | API-Key | Create device (returns JWT token) |
| 15 | POST | `/devices/token` | - | Regenerate JWT token |
| | | **Device Operations (JWT Token)** | | |
| 16 | GET | `/devices/me` | JWT | Get current device info |
| 17 | GET | `/devices/me/status` | JWT | Get connection status |
| 18 | POST | `/devices/me/login` | JWT | Login via QR code |
| 19 | POST | `/devices/me/login-code` | JWT | Login with pairing code |
| 20 | POST | `/devices/me/reconnect` | JWT | Reconnect device |
| 21 | DELETE | `/devices/me/session` | JWT | Logout device |
| 22 | GET | `/devices/me/contacts/{phone}/registered` | JWT | Check if phone is registered |
| | | **User Management** | | |
| 23 | GET | `/users/{user_jid}` | JWT | Get user info |
| 24 | GET | `/users/{user_jid}/profile-picture` | JWT | Get user profile picture |
| 25 | POST | `/users/{user_jid}/block` | JWT | Block user |
| 26 | DELETE | `/users/{user_jid}/block` | JWT | Unblock user |
| 27 | GET | `/users/me/privacy` | JWT | Get privacy settings |
| 28 | PATCH | `/users/me/privacy` | JWT | Update privacy settings |
| 29 | GET | `/users/me/status-privacy` | JWT | Get status privacy |
| 30 | POST | `/users/me/status` | JWT | Update status/about |
| 31 | GET | `/users/{jid}/devices` | JWT | Get user's linked devices |
| | | **Messaging** | | |
| 32 | POST | `/chats/{chat_jid}/messages` | JWT | Send text message |
| 33 | GET | `/chats/{chat_jid}/messages` | JWT | Get chat messages |
| 34 | POST | `/chats/{chat_jid}/images` | JWT | Send image |
| 35 | POST | `/chats/{chat_jid}/documents` | JWT | Send document |
| 36 | POST | `/chats/{chat_jid}/archive` | JWT | Archive/unarchive chat |
| 37 | POST | `/chats/{chat_jid}/pin` | JWT | Pin/unpin chat |
| | | **Message Actions** | | |
| 38 | POST | `/messages/{message_id}/read` | JWT | Mark message as read |
| 39 | POST | `/messages/{message_id}/reaction` | JWT | React to message |
| 40 | PATCH | `/messages/{message_id}` | JWT | Edit message |
| 41 | DELETE | `/messages/{message_id}` | JWT | Delete message |
| 42 | POST | `/messages/{message_id}/reply` | JWT | Reply to message |
| | | **Group Management** | | |
| 43 | GET | `/groups` | JWT | List all groups with members |
| 44 | POST | `/groups` | JWT | Create group |
| 45 | GET | `/groups/{group_jid}` | JWT | Get group info |
| 46 | POST | `/groups/{group_jid}/leave` | JWT | Leave group |
| 47 | PATCH | `/groups/{group_jid}/name` | JWT | Update group name |
| 48 | PATCH | `/groups/{group_jid}/description` | JWT | Update group description |
| 49 | POST | `/groups/{group_jid}/photo` | JWT | Update group photo |
| 50 | GET | `/groups/{group_jid}/invite-link` | JWT | Get invite link |
| 51 | PATCH | `/groups/{group_jid}/settings` | JWT | Update group settings |
| 52 | GET | `/groups/{group_jid}/participant-requests` | JWT | Get join requests |
| 53 | POST | `/groups/{group_jid}/join-approval` | JWT | Set join approval mode |
| 54 | GET | `/groups/invite/{invite_code}` | JWT | Preview group from invite |
| 55 | POST | `/groups/{group_jid}/join-invite` | JWT | Join group via invite |
| 56 | PATCH | `/groups/{group_jid}/member-add-mode` | JWT | Set member add mode |
| 57 | PATCH | `/groups/{group_jid}/topic` | JWT | Update group topic |
| 58 | POST | `/groups/{parent_group_jid}/link/{group_jid}` | JWT | Link subgroup |
| 59 | GET | `/groups/{community_jid}/linked-participants` | JWT | Get community members |
| 60 | GET | `/groups/{community_jid}/subgroups` | JWT | List community subgroups |
| 61 | POST | `/groups/{group_jid}/participants` | JWT | Add participants |
| 62 | DELETE | `/groups/{group_jid}/participants` | JWT | Remove participants |
| 63 | POST | `/groups/{group_jid}/requests/approve` | JWT | Approve join requests |
| 64 | POST | `/groups/{group_jid}/requests/reject` | JWT | Reject join requests |
| 65 | POST | `/groups/{group_jid}/admins` | JWT | Promote to admin |
| 66 | DELETE | `/groups/{group_jid}/admins` | JWT | Demote from admin |
| | | **Presence & Status** | | |
| 67 | POST | `/chats/{chat_jid}/presence` | JWT | Send typing/recording indicator |
| 68 | POST | `/presence/status` | JWT | Update availability status |
| 69 | PATCH | `/chats/{chat_jid}/disappearing-timer` | JWT | Set disappearing messages timer |
| | | **App State** | | |
| 70 | GET | `/app-state/{name}` | JWT | Fetch app state |
| 71 | POST | `/app-state` | JWT | Send app state patch |
| 72 | POST | `/app-state/mark-clean` | JWT | Mark app state as clean |
| | | **Webhooks** | | |
| 73 | GET | `/webhooks` | JWT | List webhooks |
| 74 | POST | `/webhooks` | JWT | Create webhook |
| 75 | GET | `/webhooks/{webhook_id}` | JWT | Get webhook details |
| 76 | PATCH | `/webhooks/{webhook_id}` | JWT | Update webhook |
| 77 | DELETE | `/webhooks/{webhook_id}` | JWT | Delete webhook |
| 78 | GET | `/webhooks/{webhook_id}/logs` | JWT | Get webhook logs |
| 79 | POST | `/webhooks/{webhook_id}/test` | JWT | Test webhook |
| | | **System** | | |
| 80 | GET | `/` | - | Server status |
| 81 | GET | `/docs/*` | - | Swagger UI |
| 82 | GET | `/docs/swagger.json` | - | OpenAPI JSON spec |
| 83 | GET | `/docs/swagger.yaml` | - | OpenAPI YAML spec |

**Authentication Types:**
- **Admin**: `X-Admin-Secret` header
- **API-Key**: `X-API-Key` header
- **JWT**: `Authorization: Bearer <token>` header
- **-**: No authentication required

## 🔐 Authentication Flow

This API uses a **JWT Token** authentication system optimized for high-volume operations (1000+ sessions with non-stop messaging).

### Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           ADMIN (X-Admin-Secret)                            │
├─────────────────────────────────────────────────────────────────────────────┤
│  GET  /admin/stats             → System statistics (API keys, devices)      │
│  GET  /admin/health            → System health (memory, uptime, DB status)  │
│  GET  /admin/devices           → List all devices (all customers)           │
│  GET  /admin/devices/status    → Live connection status for all devices     │
│  POST /admin/devices/reconnect → Batch reconnect all disconnected devices   │
│  GET  /admin/webhooks/stats    → Webhook delivery statistics                │
│  POST /admin/api-keys          → Create API Key for customer                │
│  GET  /admin/api-keys          → List all API keys                          │
│  PATCH /admin/api-keys/{id}    → Update API key                             │
│  DELETE /admin/api-keys/{id}   → Delete API key                             │
└─────────────────────────────────────────────────────────────────────────────┘
                                     │
                                     ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                         CUSTOMER (X-API-Key)                                │
├─────────────────────────────────────────────────────────────────────────────┤
│  POST /devices                 → Create device (returns JWT token)          │
└─────────────────────────────────────────────────────────────────────────────┘
                                     │
                                     ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                    TOKEN REGENERATION (No Auth)                             │
├─────────────────────────────────────────────────────────────────────────────┤
│  POST /devices/token           → Regenerate JWT (uses device_secret)        │
└─────────────────────────────────────────────────────────────────────────────┘
                                     │
                                     ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                   DEVICE OPERATIONS (Authorization: Bearer)                 │
├─────────────────────────────────────────────────────────────────────────────┤
│  All WhatsApp operations (login, messaging, groups, webhooks, etc.)         │
│  ⚡ STATELESS - No database hit per request = High performance              │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Why JWT? Performance at Scale

| Metric | DB Lookup (Old) | JWT (Current) |
|--------|-----------------|---------------|
| **Auth per request** | 1-5ms (DB query) | 0.1ms (CPU only) |
| **1000 sessions × 10 msg/sec** | 10,000 DB queries/sec | 0 DB queries |
| **Horizontal scaling** | Limited by DB | Unlimited |
| **Connection pressure** | High | None |

### Step 1: Create API Key (Admin)

```bash
curl -X POST "http://localhost:7001/admin/api-keys" \
  -H "X-Admin-Secret: YOUR_ADMIN_SECRET" \
  -H "Content-Type: application/json" \
  -d '{
    "customer_name": "Acme Corp",
    "max_devices": 10
  }'
```

### Step 2: Create Device & Get Token (Customer)

```bash
curl -X POST "http://localhost:7001/devices" \
  -H "X-API-Key: wam_a1b2c3d4e5f6g7h8..." \
  -H "Content-Type: application/json" \
  -d '{"device_name": "Bot 1"}'

# Response:
{
  "device_id": "550e8400-e29b-41d4-a716-446655440000",
  "device_secret": "a1b2c3d4...",  # Save this for token regeneration
  "token": "eyJhbGciOiJIUzI1NiIs...",  # Use this for all API calls
  "message": "Use the token in Authorization header"
}
```

### Step 3: Use JWT for All Operations

```bash
# All device operations use Bearer token
curl -X POST "http://localhost:7001/chats/628xxx@s.whatsapp.net/messages" \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIs..." \
  -H "Content-Type: application/json" \
  -d '{"text": "Hello!"}'
```

### Step 4: Regenerate Token (When Needed)

```bash
# Use device_secret to get a new token (invalidates old tokens)
curl -X POST "http://localhost:7001/devices/token" \
  -H "Content-Type: application/json" \
  -d '{
    "device_id": "550e8400-e29b-41d4-a716-446655440000",
    "device_secret": "a1b2c3d4..."
  }'
```

### Token Lifecycle

| Event | What Happens |
|-------|--------------|
| **Create Device** | Returns initial JWT + device_secret |
| **Use Token** | Stateless validation, no expiry |
| **Regenerate Token** | Old tokens invalidated, new token issued |
| **Delete Device** | All tokens for that device become invalid |

## 🔧 Environment Variables

| Variable | Description | Default | Example |
|----------|-------------|---------|---------|
| **Authentication** | | | |
| `ADMIN_SECRET_KEY` | Admin secret for /admin/* endpoints | - | `your-admin-secret-key` |
| `JWT_SECRET_KEY` | JWT signing secret (min 32 chars) | - | `your-jwt-secret-32-chars` |
| **Server** | | | |
| `SERVER_ADDRESS` | Server listening address | `127.0.0.1` | `0.0.0.0` |
| `SERVER_PORT` | Server listening port | `7001` | `3000` |
| **HTTP Configuration** | | | |
| `HTTP_BASE_URL` | Base URL path for API | `` | `/api/v1` |
| `HTTP_CORS_ORIGIN` | CORS allowed origins | `*` | `https://example.com` |
| `HTTP_BODY_LIMIT_SIZE` | Max request body size | `8M` | `50M` |
| `HTTP_GZIP_LEVEL` | GZIP compression level (1-9) | `1` | `6` |
| `HTTP_CACHE_CAPACITY` | In-memory cache capacity | `100` | `500` |
| `HTTP_CACHE_TTL_SECONDS` | Cache TTL in seconds | `5` | `300` |
| **WhatsApp** | | | |
| `WHATSAPP_DATASTORE_TYPE` | Database type | - | `postgres` |
| `WHATSAPP_DATASTORE_URI` | Database connection URI | - | `postgres://user:pass@host:5432/db` |
| `WHATSAPP_CLIENT_PROXY_URL` | HTTP proxy for WhatsApp | - | `http://proxy:8080` |
| `WHATSAPP_MEDIA_IMAGE_CONVERT_WEBP` | Convert images to WebP | `false` | `true` |
| `WHATSAPP_MEDIA_IMAGE_COMPRESSION` | Enable image compression | `false` | `true` |
| **Webhooks** | | | |
| `WEBHOOKS_ENABLED` | Enable webhook system | `true` | `true` |
| `WEBHOOK_WORKERS` | Number of webhook workers | `4` | `8` |
| `WEBHOOK_RETRY_LIMIT` | Max retry attempts | `3` | `5` |
| `WEBHOOK_MAX_PER_DEVICE` | Max webhooks per device | `5` | `10` |

### Example `.env` File

```env
# Server
SERVER_ADDRESS=0.0.0.0
SERVER_PORT=7001

# HTTP
HTTP_BASE_URL=
HTTP_CORS_ORIGIN=*
HTTP_BODY_LIMIT_SIZE=50M

# Authentication
ADMIN_SECRET_KEY=your-super-secret-admin-key-change-this
JWT_SECRET_KEY=your-jwt-secret-key-at-least-32-characters-long

# WhatsApp
WHATSAPP_DATASTORE_TYPE=postgres
WHATSAPP_DATASTORE_URI=postgres://whatsapp:secret@localhost:5432/whatsapp?sslmode=disable
WHATSAPP_MEDIA_IMAGE_COMPRESSION=true

# Webhooks
WEBHOOKS_ENABLED=true
WEBHOOK_WORKERS=4
WEBHOOK_RETRY_LIMIT=3
WEBHOOK_MAX_PER_DEVICE=5
```

## 🧪 Testing

```bash
# Run tests
go test ./...

# Run tests with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## ⚠️ Disclaimer

**Important Legal Information:**

- This project is **unofficial** and **not affiliated** with WhatsApp Inc. or Meta Platforms Inc.
- This software is provided "as is" without warranty of any kind
- Use at your own risk and in accordance with [WhatsApp's Terms of Service](https://www.whatsapp.com/legal/terms-of-service)
- The authors and contributors are not responsible for any misuse or violation of WhatsApp's terms
- This project is intended for **educational and personal use** only

**Recommended Usage:**

- ✅ Personal automation and notifications
- ✅ Small business internal tools
- ✅ Educational purposes
- ❌ Mass messaging / spam
- ❌ Violating WhatsApp Terms of Service
- ❌ Commercial use without proper authorization

## 📞 Support & Community

- 🐛 **Issues**: [GitHub Issues](https://github.com/gdbrns/go-whatsapp-multi-session-rest-api/issues)
- 💬 **Discussions**: [GitHub Discussions](https://github.com/gdbrns/go-whatsapp-multi-session-rest-api/discussions)
- 💻 **Repository**: [github.com/gdbrns/go-whatsapp-multi-session-rest-api](https://github.com/gdbrns/go-whatsapp-multi-session-rest-api)

## 🙏 Acknowledgments

### Core Technologies

- **[whatsmeow](https://pkg.go.dev/go.mau.fi/whatsmeow)** - WhatsApp Multi-Device protocol library
- **[Fiber](https://github.com/gofiber/fiber)** - Express-inspired web framework for Go

### Inspiration & Code References

- **[aldinokemal/go-whatsapp-web-multidevice](https://github.com/aldinokemal/go-whatsapp-web-multidevice)** - Excellent WhatsApp API implementation
- **[dimaskiddo/go-whatsapp-multidevice-rest](https://github.com/dimaskiddo/go-whatsapp-multidevice-rest)** - Clean REST API architecture patterns

---

**Made with ❤️ using Go**

⭐ Star this repository if you find it useful!
