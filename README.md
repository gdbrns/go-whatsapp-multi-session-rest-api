# Go WhatsApp Multi-Session REST API

[![release version](https://img.shields.io/github/v/release/gdbrns/go-whatsapp-multi-session-rest-api)](https://github.com/gdbrns/go-whatsapp-multi-session-rest-api/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/gdbrns/go-whatsapp-multi-session-rest-api)](https://goreportcard.com/report/github.com/gdbrns/go-whatsapp-multi-session-rest-api)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![whatsmeow](https://img.shields.io/badge/whatsmeow-v0.0.0--20251216-brightgreen.svg)](https://pkg.go.dev/go.mau.fi/whatsmeow@v0.0.0-20251216102424-56a8e44b0cec)
[![API Docs](https://img.shields.io/badge/API%20Docs-Bump.sh-blue.svg)](https://bump.sh/gdbrns/doc/go-whatsapp-multi-session-rest)

> A minimal REST API for WhatsApp Multi-Device and Multi-Session implementation built with Go and **[whatsmeow v0.0.0-20251216102424-56a8e44b0cec](https://pkg.go.dev/go.mau.fi/whatsmeow@v0.0.0-20251216102424-56a8e44b0cec)** (current latest). Supports multiple accounts and devices simultaneously with efficient memory use and production-ready deployments.

## 📚 API Documentation

| Format | Link |
|--------|------|
| 🌐 **Interactive Docs** | **[bump.sh/gdbrns/doc/go-whatsapp-multi-session-rest](https://bump.sh/gdbrns/doc/go-whatsapp-multi-session-rest)** |
| 📖 Swagger UI | `http://localhost:7001/docs/` (when running) |
| 📄 OpenAPI JSON | `http://localhost:7001/docs/swagger.json` |
| 📄 OpenAPI YAML | `http://localhost:7001/docs/swagger.yaml` |

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
- 🚀 **120+ Production-Ready Endpoints** - Full whatsmeow coverage including all media types
- 📨 **Rich Messaging** - Text, images, videos, audio, stickers, locations, contacts, documents
- 📊 **Polls** - Create polls, vote, and receive real-time results via webhooks
- 📰 **Newsletter/Channels** - Full channel support - create, follow, message, react, TOS acceptance
- 📱 **Status/Stories** - Post and manage WhatsApp Status updates
- 📞 **Call Handling** - Reject incoming WhatsApp calls programmatically
- 💼 **Business Profiles** - Get business profiles and resolve wa.me/message links
- 🤖 **WhatsApp AI Bots** - List and get profiles of available WhatsApp bots
- 📲 **Contact QR Codes** - Generate and resolve contact QR links
- 👁️ **Presence Subscriptions** - Subscribe to user online/offline/typing status
- 🌳 **RESTful Architecture** - Hierarchical resource-based routes
- 👥 **Group Management** - Full CRUD for groups, participants, admins, community unlinking
- 🔄 **Message Operations** - Edit, react, delete, reply, forward, mark read
- 📊 **Presence & Status** - Online status, typing indicators, disappearing messages, passive mode
- 🔄 **App State Synchronization** - Fetch, send, and manage app state patches
- 🪝 **Webhook Integration** - 31 event types with real-time notifications and retry support
- 🏗️ **Production Ready** - Docker support, environment configuration, logging
- 📖 **OpenAPI/Swagger** - Interactive API documentation at `/docs/`
- 🔑 **Admin Dashboard Ready** - Manage API keys and devices with admin endpoints
- 📈 **Admin Dashboard APIs** - System stats, health monitoring, device status, batch reconnect

## 💡 Why Teams Choose This API

- **Stateless speed**: JWT-first flow eliminates per-request database lookups for messaging throughput.
- **Auto-resilience**: Devices restore and reconnect on startup with health checks for steady uptime.
- **Operational control**: Admin endpoints manage API keys, devices, and webhook delivery visibility.
- **Webhook reliability**: Built-in workers, retries, and per-device quotas keep downstream systems in sync.
- **Deployment-ready**: Docker-compose defaults, environment-first config, and structured logging out of the box.
- **Developer experience**: Full Swagger (JSON/YAML) at `/docs/`, with consistent JSON error envelopes and request IDs for tracing.

## 📋 Requirements

### System Requirements

- **Go**: 1.24 or higher (matches `go.mod` 1.24.5; use the latest stable toolchain)
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

**Online Documentation (Recommended):**
- 🌐 **[bump.sh/gdbrns/doc/go-whatsapp-multi-session-rest](https://bump.sh/gdbrns/doc/go-whatsapp-multi-session-rest)** - Interactive API explorer with examples

**Local Documentation (when running):**
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

### Webhook Events Summary (31 Event Types)

| Category | Events |
|----------|--------|
| **Messages** (5) | `message.received`, `message.delivered`, `message.read`, `message.played`, `message.deleted` |
| **Connection** (6) | `connection.connected`, `connection.disconnected`, `connection.logged_out`, `connection.reconnecting`, `connection.keepalive_timeout`, `connection.temporary_ban` |
| **Calls** (4) | `call.offer`, `call.accept`, `call.terminate`, `call.reject` |
| **Groups** (4) | `group.join`, `group.leave`, `group.participant_update`, `group.info_update` |
| **Newsletter** (4) | `newsletter.join`, `newsletter.leave`, `newsletter.message_received`, `newsletter.update` |
| **Polls** (3) | `poll.created`, `poll.vote`, `poll.update` |
| **Status** (3) | `status.posted`, `status.viewed`, `status.deleted` |
| **Media** (2) | `media.received`, `media.downloaded` |
| **Contact** (2) | `contact.update`, `blocklist.change` |
| **App State** (2) | `appstate.sync_complete`, `appstate.patch_received` |
| **History** (1) | `history.sync` |

📖 See [`docs/WEBHOOK_EVENTS.md`](docs/WEBHOOK_EVENTS.md) for detailed event payloads.

## 📚 API Endpoints

### All Endpoints

**Organized by category** with most commonly used endpoints first. All paths honor optional `HTTP_BASE_URL` (prefix not shown below). Total: **122 endpoints** (123 when `HTTP_BASE_URL` is set because the index path is registered with and without a trailing slash).

**Input & validation notes**
- Phone numbers must be in international format (no leading `0`, digits only, 6-16 chars). Requests with invalid/unknown numbers return 4xx.
- `chat_jid`/`sender_jid` must be provided for message operations; missing values return 4xx.
- All errors are JSON with `{status, code, message, data?}` (and `error` for compatibility).
- Every response carries `X-Request-ID`; you can supply your own header to correlate logs.

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
| 13 | GET | `/admin/api-keys/{id}/devices/status` | Admin | Get live connection status for API key devices |
| 14 | DELETE | `/admin/devices/{device_id}` | Admin | Delete a device |
| | | **Device Creation & Token** | | |
| 15 | POST | `/devices` | API-Key | Create device (returns JWT token) |
| 16 | POST | `/devices/token` | - | Regenerate JWT token |
| | | **Device Operations (JWT Token)** | | |
| 17 | GET | `/devices/me` | JWT | Get current device info |
| 18 | GET | `/devices/me/status` | JWT | Get connection status |
| 19 | POST | `/devices/me/login` | JWT | Login via QR code |
| 20 | POST | `/devices/me/login-code` | JWT | Login with pairing code |
| 21 | POST | `/devices/me/reconnect` | JWT | Reconnect device |
| 22 | DELETE | `/devices/me/session` | JWT | Logout device |
| 23 | GET | `/devices/me/contacts/{phone}/registered` | JWT | Check if phone is registered |
| 24 | POST | `/devices/me/passive` | JWT | Set passive mode (reduce bandwidth) |
| | | **User Management** | | |
| 25 | GET | `/users/{user_jid}` | JWT | Get user info |
| 26 | GET | `/users/{user_jid}/profile-picture` | JWT | Get user profile picture |
| 27 | POST | `/users/{user_jid}/block` | JWT | Block user |
| 28 | DELETE | `/users/{user_jid}/block` | JWT | Unblock user |
| 29 | GET | `/users/me/privacy` | JWT | Get privacy settings |
| 30 | PATCH | `/users/me/privacy` | JWT | Update privacy settings |
| 31 | GET | `/users/me/status-privacy` | JWT | Get status privacy |
| 32 | POST | `/users/me/status` | JWT | Update status/about |
| 33 | GET | `/users/{jid}/devices` | JWT | Get user's linked devices |
| 34 | POST | `/users/me/profile-photo` | JWT | Set profile photo |
| 35 | GET | `/users/me/contacts` | JWT | Get contacts |
| 36 | POST | `/users/me/contacts/sync` | JWT | Sync contacts |
| 37 | GET | `/users/me/blocklist` | JWT | Get blocklist |
| | | **Contact QR (NEW)** | | |
| 38 | GET | `/users/me/contact-qr` | JWT | Get your contact QR link |
| 39 | GET | `/users/contact-qr/{code}` | JWT | Resolve contact QR code |
| | | **Messaging** | | |
| 40 | POST | `/chats/{chat_jid}/messages` | JWT | Send text message |
| 41 | GET | `/chats/{chat_jid}/messages` | JWT | Get chat messages |
| 42 | POST | `/chats/{chat_jid}/images` | JWT | Send image |
| 43 | POST | `/chats/{chat_jid}/videos` | JWT | Send video |
| 44 | POST | `/chats/{chat_jid}/audio` | JWT | Send audio/voice note |
| 45 | POST | `/chats/{chat_jid}/stickers` | JWT | Send sticker (WebP) |
| 46 | POST | `/chats/{chat_jid}/locations` | JWT | Send location |
| 47 | POST | `/chats/{chat_jid}/contacts` | JWT | Send contact vCard |
| 48 | POST | `/chats/{chat_jid}/documents` | JWT | Send document |
| 49 | POST | `/chats/{chat_jid}/archive` | JWT | Archive/unarchive chat |
| 50 | POST | `/chats/{chat_jid}/pin` | JWT | Pin/unpin chat |
| | | **Message Actions** | | |
| 51 | POST | `/messages/{message_id}/read` | JWT | Mark message as read |
| 52 | POST | `/messages/{message_id}/reaction` | JWT | React to message |
| 53 | PATCH | `/messages/{message_id}` | JWT | Edit message |
| 54 | DELETE | `/messages/{message_id}` | JWT | Delete message |
| 55 | POST | `/messages/{message_id}/reply` | JWT | Reply to message |
| 56 | POST | `/messages/{message_id}/forward` | JWT | Forward message |
| | | **Polls** | | |
| 57 | POST | `/chats/{chat_jid}/polls` | JWT | Create poll |
| 58 | POST | `/polls/{poll_id}/vote` | JWT | Vote on poll |
| 59 | GET | `/polls/{poll_id}/results` | JWT | Get poll results |
| 60 | DELETE | `/polls/{poll_id}` | JWT | Delete poll |
| | | **Calls (NEW)** | | |
| 61 | POST | `/calls/reject` | JWT | Reject incoming call |
| | | **Business (NEW)** | | |
| 62 | GET | `/business/{jid}/profile` | JWT | Get business profile |
| 63 | GET | `/business/link/{code}` | JWT | Resolve wa.me/message link |
| | | **Bots (NEW)** | | |
| 64 | GET | `/bots` | JWT | List available WhatsApp bots |
| 65 | GET | `/bots/profiles` | JWT | Get bot profiles |
| | | **Group Management** | | |
| 66 | GET | `/groups` | JWT | List all groups with members |
| 67 | POST | `/groups` | JWT | Create group |
| 68 | GET | `/groups/{group_jid}` | JWT | Get group info |
| 69 | POST | `/groups/{group_jid}/leave` | JWT | Leave group |
| 70 | PATCH | `/groups/{group_jid}/name` | JWT | Update group name |
| 71 | PATCH | `/groups/{group_jid}/description` | JWT | Update group description |
| 72 | POST | `/groups/{group_jid}/photo` | JWT | Update group photo |
| 73 | GET | `/groups/{group_jid}/invite-link` | JWT | Get invite link |
| 74 | PATCH | `/groups/{group_jid}/settings` | JWT | Update group settings |
| 75 | GET | `/groups/{group_jid}/participant-requests` | JWT | Get join requests |
| 76 | POST | `/groups/{group_jid}/join-approval` | JWT | Set join approval mode |
| 77 | GET | `/groups/invite/{invite_code}` | JWT | Preview group from invite |
| 78 | POST | `/groups/{group_jid}/join-invite` | JWT | Join group via invite |
| 79 | PATCH | `/groups/{group_jid}/member-add-mode` | JWT | Set member add mode |
| 80 | PATCH | `/groups/{group_jid}/topic` | JWT | Update group topic |
| 81 | POST | `/groups/{parent_group_jid}/link/{group_jid}` | JWT | Link subgroup |
| 82 | DELETE | `/groups/{parent_jid}/link/{child_jid}` | JWT | Unlink subgroup (NEW) |
| 83 | GET | `/groups/{community_jid}/linked-participants` | JWT | Get community members |
| 84 | GET | `/groups/{community_jid}/subgroups` | JWT | List community subgroups |
| 85 | POST | `/groups/{group_jid}/participants` | JWT | Add participants |
| 86 | DELETE | `/groups/{group_jid}/participants` | JWT | Remove participants |
| 87 | POST | `/groups/{group_jid}/requests/approve` | JWT | Approve join requests |
| 88 | POST | `/groups/{group_jid}/requests/reject` | JWT | Reject join requests |
| 89 | POST | `/groups/{group_jid}/admins` | JWT | Promote to admin |
| 90 | DELETE | `/groups/{group_jid}/admins` | JWT | Demote from admin |
| | | **Presence & Status** | | |
| 91 | POST | `/chats/{chat_jid}/presence` | JWT | Send typing/recording indicator |
| 92 | POST | `/presence/status` | JWT | Update availability status |
| 93 | POST | `/presence/subscribe` | JWT | Subscribe to presence updates (NEW) |
| 94 | PATCH | `/chats/{chat_jid}/disappearing-timer` | JWT | Set disappearing messages timer |
| | | **App State** | | |
| 95 | GET | `/app-state/{name}` | JWT | Fetch app state |
| 96 | POST | `/app-state` | JWT | Send app state patch |
| 97 | POST | `/app-state/mark-clean` | JWT | Mark app state as clean |
| | | **Webhooks** | | |
| 98 | GET | `/webhooks` | JWT | List webhooks |
| 99 | POST | `/webhooks` | JWT | Create webhook |
| 100 | GET | `/webhooks/{webhook_id}` | JWT | Get webhook details |
| 101 | PATCH | `/webhooks/{webhook_id}` | JWT | Update webhook |
| 102 | DELETE | `/webhooks/{webhook_id}` | JWT | Delete webhook |
| 103 | GET | `/webhooks/{webhook_id}/logs` | JWT | Get webhook logs |
| 104 | POST | `/webhooks/{webhook_id}/test` | JWT | Test webhook |
| | | **Newsletter/Channels** | | |
| 105 | GET | `/newsletters` | JWT | List subscribed newsletters |
| 106 | POST | `/newsletters` | JWT | Create newsletter |
| 107 | GET | `/newsletters/{jid}` | JWT | Get newsletter info |
| 108 | POST | `/newsletters/{jid}/follow` | JWT | Follow newsletter |
| 109 | DELETE | `/newsletters/{jid}/follow` | JWT | Unfollow newsletter |
| 110 | GET | `/newsletters/{jid}/messages` | JWT | Get newsletter messages |
| 111 | POST | `/newsletters/{jid}/messages` | JWT | Send newsletter message |
| 112 | POST | `/newsletters/{jid}/reaction` | JWT | React to newsletter message |
| 113 | POST | `/newsletters/{jid}/mute` | JWT | Toggle newsletter mute |
| 114 | POST | `/newsletters/{jid}/viewed` | JWT | Mark messages viewed |
| 115 | GET | `/newsletters/invite/{code}` | JWT | Get info from invite |
| 116 | POST | `/newsletters/{jid}/live` | JWT | Subscribe to live updates |
| 117 | POST | `/newsletters/{jid}/photo` | JWT | Update newsletter photo |
| 118 | GET | `/newsletters/{jid}/updates` | JWT | Get message updates (NEW) |
| 119 | POST | `/newsletters/tos/accept` | JWT | Accept TOS notice (NEW) |
| | | **Status/Stories** | | |
| 120 | POST | `/status` | JWT | Post status (text/image/video) |
| 121 | GET | `/status` | JWT | Get status updates |
| 122 | DELETE | `/status/{status_id}` | JWT | Delete status |
| 123 | GET | `/status/{user_jid}` | JWT | Get user status |
| | | **System** | | |
| 124 | GET | `/` | - | Server status |
| 125 | GET | `/docs/*` | - | Swagger UI |
| 126 | GET | `/docs/swagger.json` | - | OpenAPI JSON spec |
| 127 | GET | `/docs/swagger.yaml` | - | OpenAPI YAML spec |

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

### Stability & health
- Startup automatically restores devices from the datastore and attempts reconnect; see logs for `restored/reconnected/failed` summary.
- Enable periodic health logging with `WHATSAPP_ENABLE_HEALTH_CHECK_CRON=true` (runs every 5 minutes).

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
| `HTTP_BASE_URL` | Base URL path for API | `` | `` |
| `HTTP_CORS_ORIGIN` | CORS allowed origins | `*` | `https://example.com` |
| `HTTP_BODY_LIMIT_SIZE` | Max request body size | `8M` | `8M` |
| `HTTP_GZIP_LEVEL` | GZIP compression level (1-9) | `1` | `6` |
| `HTTP_CACHE_CAPACITY` | In-memory cache capacity | `100` | `500` |
| `HTTP_CACHE_TTL_SECONDS` | Cache TTL in seconds | `5` | `300` |
| **Logging** | | | |
| `LOG_LEVEL` | Log verbosity (`debug`, `info`, `warn`, `error`) | `info` | `debug` |
| `LOG_FORMAT` | Log format (`json` for structured) | `text` | `json` |
| **WhatsApp** | | | |
| `WHATSAPP_DATASTORE_TYPE` | Database type | - | `postgres` |
| `WHATSAPP_DATASTORE_URI` | Database connection URI | - | `postgres://user:pass@host:5432/db` |
| `WHATSAPP_CLIENT_PROXY_URL` | HTTP proxy for WhatsApp | `""` | `http://proxy:8080` |
| `WHATSAPP_MEDIA_IMAGE_CONVERT_WEBP` | Convert images to WebP uploads to PNG | `false` | `true` |
| `WHATSAPP_MEDIA_IMAGE_COMPRESSION` | Enable image compression | `false` | `true` |
| `WHATSAPP_DEVICE_OS_NAME` | Advertised device OS name | `Chrome` | `Linux` |
| `WHATSAPP_LOG_LEVEL` | whatsmeow client log level | `ERROR` | `DEBUG` |
| `WHATSAPP_VERSION_MAJOR` | Override client version (major) | unset | `2` |
| `WHATSAPP_VERSION_MINOR` | Override client version (minor) | unset | `3000` |
| `WHATSAPP_VERSION_PATCH` | Override client version (patch) | unset | `1019175440` |
| `WHATSAPP_GROUP_LIST_CACHE_TTL` | Cache TTL for group list | `5m` | `10m` |
| `WHATSAPP_GROUP_LIST_CACHE_DISABLED` | Disable group list cache | `false` | `true` |
| `WHATSAPP_AUTO_MARK_READ` | Auto mark messages as read | `false` | `true` |
| `WHATSAPP_AUTO_DOWNLOAD_MEDIA` | Auto download media | `false` | `true` |
| `WHATSAPP_AUTO_REPLY_ENABLED` | Enable auto-reply hook | `false` | `true` |
| `WHATSAPP_ENABLE_HEALTH_CHECK_CRON` | Enable 5-min health check cron | `false` | `true` |
| `X-Request-ID` | (Header) Correlate logs per request | - | `123e4567-e89b-12d3-a456-426614174000` |
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
HTTP_BODY_LIMIT_SIZE=8M
# HTTP_GZIP_LEVEL=1
# HTTP_CACHE_CAPACITY=100
# HTTP_CACHE_TTL_SECONDS=5

# Logging
# LOG_LEVEL=info
# LOG_FORMAT=text

# Authentication
ADMIN_SECRET_KEY=your-super-secret-admin-key-change-this
JWT_SECRET_KEY=your-jwt-secret-key-at-least-32-characters-long

# WhatsApp
WHATSAPP_DATASTORE_TYPE=postgres
WHATSAPP_DATASTORE_URI=postgres://whatsapp:secret@localhost:5432/whatsapp?sslmode=disable
WHATSAPP_CLIENT_PROXY_URL=""
WHATSAPP_MEDIA_IMAGE_COMPRESSION=true
WHATSAPP_MEDIA_IMAGE_CONVERT_WEBP=true
# WHATSAPP_DEVICE_OS_NAME=Chrome
# WHATSAPP_LOG_LEVEL=ERROR
# WHATSAPP_VERSION_MAJOR=2
# WHATSAPP_VERSION_MINOR=3000
# WHATSAPP_VERSION_PATCH=1019175440
# WHATSAPP_GROUP_LIST_CACHE_TTL=5m
# WHATSAPP_GROUP_LIST_CACHE_DISABLED=false
# WHATSAPP_AUTO_MARK_READ=false
# WHATSAPP_AUTO_DOWNLOAD_MEDIA=false
# WHATSAPP_AUTO_REPLY_ENABLED=false
# WHATSAPP_ENABLE_HEALTH_CHECK_CRON=false

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

- **[whatsmeow](https://pkg.go.dev/go.mau.fi/whatsmeow)** by [@tulir](https://github.com/tulir) - WhatsApp Multi-Device protocol library. This project wouldn't exist without this incredible reverse-engineering effort of the WhatsApp Web protocol. Currently using **v0.0.0-20251216** (Dec 16, 2025).
- **[Fiber](https://github.com/gofiber/fiber)** - Express-inspired web framework for Go
- **[libsignal](https://pkg.go.dev/go.mau.fi/libsignal)** - Signal Protocol implementation for E2E encryption

### Inspiration & Code References

- **[aldinokemal/go-whatsapp-web-multidevice](https://github.com/aldinokemal/go-whatsapp-web-multidevice)** - Excellent WhatsApp API implementation
- **[dimaskiddo/go-whatsapp-multidevice-rest](https://github.com/dimaskiddo/go-whatsapp-multidevice-rest)** - Clean REST API architecture patterns

### Special Thanks

- **[@tulir](https://github.com/tulir)** - Creator and maintainer of whatsmeow
- **[mautrix](https://github.com/mautrix)** - Matrix bridges and Go libraries ecosystem
- All contributors and users who report issues and help improve this project

---

**Made with ❤️ using Go**

⭐ Star this repository if you find it useful!
