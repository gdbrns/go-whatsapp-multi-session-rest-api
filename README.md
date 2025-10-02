# Go WhatsApp Multi-Session REST API

[![release version](https://img.shields.io/github/v/release/gdbrns/go-whatsapp-multi-session-rest-api)](https://github.com/gdbrns/go-whatsapp-multi-session-rest-api/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/gdbrns/go-whatsapp-multi-session-rest-api)](https://goreportcard.com/report/github.com/gdbrns/go-whatsapp-multi-session-rest-api)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![whatsmeow](https://img.shields.io/badge/whatsmeow-v0.0.0--20250930-brightgreen.svg)](https://pkg.go.dev/go.mau.fi/whatsmeow@v0.0.0-20250930215512-38f9aaa3ba7c)

> A powerful REST API for WhatsApp automation built with Go and **[whatsmeow v0.0.0-20250930215512](https://pkg.go.dev/go.mau.fi/whatsmeow@v0.0.0-20250930215512-38f9aaa3ba7c)** (September 30, 2025 - **latest version**), supporting multiple accounts and devices simultaneously. Built for efficient memory use and production-ready deployments.

**📚 API Documentation:** [https://bump.sh/gdbrns/doc/go-whatsapp-multi-session-rest-api](https://bump.sh/gdbrns/doc/go-whatsapp-multi-session-rest-api)

## 📋 Table of Contents

- [Features](#-features)
- [Requirements](#-requirements)
- [Installation](#-installation)
  - [Using Docker](#using-docker-recommended)
  - [Using Binary](#using-binary)
  - [From Source](#from-source)
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
- 📱 **Multi-Device Support** - Up to 4 devices per WhatsApp account (WhatsApp Multi-Device protocol)
- 🎫 **Device-scoped JWT Authentication** - Secure token-based authentication per device
- 📨 **Complete Messaging** - Text, media (image, video, audio), location, contacts, links, polls, and stickers
- 🌳 **RESTful Architecture** - Hierarchical resource-based routes following REST principles
- 👥 **Full Group Management** - CRUD operations for groups, participants, settings, and communities
- 📰 **Newsletter/Channel Support** - Create, manage, and publish to WhatsApp newsletters
- 🔄 **Message Operations** - Edit, react, delete, forward, reply, mark read
- 📊 **Presence Management** - Online status, typing indicators, disappearing messages
- 🤖 **Bot Integration** - List and interact with WhatsApp bots
- 💼 **Business Features** - Access business profiles and resolve business links
- 🎬 **Media Processing** - Download media from messages and external URLs
- 🔍 **Contact Management** - Check registered contacts in batch
- 📍 **Push Notifications** - Register and configure push notification settings
- 🏗️ **Production Ready** - Docker support, environment configuration, logging
- 📖 **OpenAPI/Swagger** - Interactive API documentation built-in at `/docs/`

## 📋 Requirements

### System Requirements

- **Go**: 1.19 or higher (for building from source)
- **SQLite3**: Default database (or PostgreSQL)
- **FFmpeg**: For media processing (optional but recommended)
- **whatsmeow**: Using **[v0.0.0-20250930215512](https://pkg.go.dev/go.mau.fi/whatsmeow@v0.0.0-20250930215512-38f9aaa3ba7c)** (September 30, 2025) - the **very latest version** for maximum compatibility and newest WhatsApp features

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
sudo apt install -y ffmpeg sqlite3
```

#### Windows
- Install FFmpeg from [official website](https://www.ffmpeg.org/download.html#build-windows)
- Add FFmpeg to [environment PATH](https://www.google.com/search?q=windows+add+to+environment+path)
- **Note**: WSL (Windows Subsystem for Linux) is recommended for better compatibility

## 🚀 Installation

### Using Docker (Recommended)

#### Docker Run
```bash
docker run -d \
  --name whatsapp-api \
  --restart always \
  -p 3000:3000 \
  -v whatsapp-data:/app/dbs \
  -e AUTH_BASIC_USERNAME=admin \
  -e AUTH_BASIC_PASSWORD=secret \
  -e AUTH_JWT_SECRET=your-jwt-secret-key \
  gdbrns/go-whatsapp-multi-session-rest-api
```

#### Docker Compose
Create a `docker-compose.yml` file:

```yaml
version: '3.8'

services:
  whatsapp-api:
    image: gdbrns/go-whatsapp-multi-session-rest-api
    container_name: whatsapp-api
    restart: always
    ports:
      - "3000:3000"
    volumes:
      - whatsapp-data:/app/dbs
    environment:
      - AUTH_BASIC_USERNAME=admin
      - AUTH_BASIC_PASSWORD=secret
      - AUTH_JWT_SECRET=your-jwt-secret-key-here
      - AUTH_JWT_EXPIRED_HOUR=24
      - SERVER_PORT=3000
      - WHATSAPP_DATASTORE_TYPE=sqlite3
      - WHATSAPP_DATASTORE_URI=file:dbs/WhatsApp.db?_foreign_keys=on

volumes:
  whatsapp-data:
```

Then run:
```bash
docker-compose up -d
```

### Using Binary

1. Download the latest binary from [Releases](https://github.com/gdbrns/go-whatsapp-multi-session-rest-api/releases)
2. Extract the archive
3. Create `.env` file from `.env.example`:
   ```bash
   cp .env.example .env
   # Edit .env with your configuration
   ```
4. Run the application:
   ```bash
   # Linux / Mac OS
   chmod +x whatsapp-api
   ./whatsapp-api

   # Windows
   whatsapp-api.exe
   ```

### From Source

```bash
# Clone repository
git clone https://github.com/gdbrns/go-whatsapp-multi-session-rest-api.git
cd go-whatsapp-multi-session-rest-api/rest-api

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

Configuration can be set in three ways (in order of priority):

1. **Environment variables** (highest priority)
2. **`.env` file** in the application directory
3. **Default values** (lowest priority)

### Quick Setup

```bash
# Copy example configuration
cp .env.example .env

# Edit configuration
nano .env  # or use your preferred editor
```

See [Environment Variables](#-environment-variables) section for all available options.

## 📖 How to Use

### 1. Start the Application

```bash
# Using Docker
docker-compose up -d

# Using binary
./whatsapp-api

# From source
go run cmd/main/main.go
```

### 2. Access API Documentation

Open your browser and navigate to:
- **Swagger UI**: `http://localhost:3000/docs/`
- **OpenAPI Spec**: `http://localhost:3000/docs/swagger.json`
- **Full Documentation**: [bump.sh](https://bump.sh/gdbrns/doc/go-whatsapp-multi-session-rest-api)

### 3. Register Device & Get JWT Token

```bash
# Register a new device (requires Basic Auth)
curl -X POST "http://localhost:3000/device/add" \
  -u "6281234567890:your_password" \
  -H "Content-Type: application/json"

# Response:
# {
#   "code": 200,
#   "message": "Success",
#   "data": {
#     "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
#     "jid": "6281234567890",
#     "device_id": "550e8400-e29b-41d4-a716-446655440000"
#   }
# }
```

### 4. Login via QR Code

```bash
# Generate QR code for device login
curl -X POST "http://localhost:3000/devices/{device_id}/login" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -F "output=json"

# Scan the QR code with your WhatsApp mobile app:
# WhatsApp > Settings > Linked Devices > Link a Device
```

### 5. Send a Message

```bash
# Send text message
curl -X POST "http://localhost:3000/chats/6281234567890@s.whatsapp.net/messages" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "text": "Hello from WhatsApp API!"
  }'

# Send image
curl -X POST "http://localhost:3000/chats/6281234567890@s.whatsapp.net/images" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -F "image=@/path/to/image.jpg" \
  -F "caption=Check this out!"
```

## 📚 API Endpoints

Complete API reference: [https://bump.sh/gdbrns/doc/go-whatsapp-multi-session-rest-api](https://bump.sh/gdbrns/doc/go-whatsapp-multi-session-rest-api)

### All Endpoints (97 Total)

| # | Status | Description | Method | Endpoint |
|---|:------:|-------------|:------:|----------|
| 1 | ✅ | Register device & get JWT token | POST | `/device/add` |
| 2 | ✅ | Login via QR code | POST | `/devices/{device_id}/login` |
| 3 | ✅ | Login with pairing code | POST | `/devices/{device_id}/login-code` |
| 4 | ✅ | Send text message | POST | `/chats/{chat_jid}/messages` |
| 5 | ✅ | Send image | POST | `/chats/{chat_jid}/images` |
| 6 | ✅ | Send video | POST | `/chats/{chat_jid}/videos` |
| 7 | ✅ | Send audio | POST | `/chats/{chat_jid}/audios` |
| 8 | ✅ | Send document/file | POST | `/chats/{chat_jid}/documents` |
| 9 | ✅ | Send sticker | POST | `/chats/{chat_jid}/stickers` |
| 10 | ✅ | Send location | POST | `/chats/{chat_jid}/locations` |
| 11 | ✅ | Send contact | POST | `/chats/{chat_jid}/contacts` |
| 12 | ✅ | Send link message | POST | `/chats/{chat_jid}/links` |
| 13 | ✅ | Send poll | POST | `/chats/{chat_jid}/polls` |
| 14 | ✅ | Mark message as read | POST | `/messages/{message_id}/read` |
| 15 | ✅ | React to message | POST | `/messages/{message_id}/reaction` |
| 16 | ✅ | Reply to message | POST | `/messages/{message_id}/reply` |
| 17 | ✅ | Forward message | POST | `/messages/{message_id}/forward` |
| 18 | ✅ | Delete message | DELETE | `/messages/{message_id}` |
| 19 | ✅ | Edit message | PATCH | `/messages/{message_id}` |
| 20 | ✅ | Get chat messages | GET | `/chats/{chat_jid}/messages` |
| 21 | ✅ | Download media from message | POST | `/media/download` |
| 22 | ✅ | Get message thumbnail | GET | `/media/{message_id}/thumbnail` |
| 23 | ✅ | List all devices | GET | `/devices` |
| 24 | ✅ | Get device info | GET | `/devices/{device_id}` |
| 25 | ✅ | Get device status | GET | `/devices/{device_id}/status` |
| 26 | ✅ | Reconnect device | POST | `/devices/{device_id}/reconnect` |
| 27 | ✅ | Logout & delete session | DELETE | `/devices/{device_id}/session` |
| 28 | ✅ | Check if phone is registered | GET | `/devices/{device_id}/contacts/{phone}/registered` |
| 29 | ✅ | Get user info by JID | GET | `/users/{user_jid}` |
| 30 | ✅ | Get user profile picture | GET | `/users/{user_jid}/profile-picture` |
| 31 | ✅ | Update WhatsApp status | POST | `/users/me/status` |
| 32 | ✅ | Get my privacy settings | GET | `/users/me/privacy` |
| 33 | ✅ | Update privacy settings | PATCH | `/users/me/privacy` |
| 34 | ✅ | Get status privacy | GET | `/users/me/status-privacy` |
| 35 | ✅ | Block user | POST | `/users/{user_jid}/block` |
| 36 | ✅ | Unblock user | DELETE | `/users/{user_jid}/block` |
| 37 | ✅ | Get user's linked devices | GET | `/users/{jid}/devices` |
| 38 | ✅ | List all groups | GET | `/groups` |
| 39 | ✅ | Get joined groups | GET | `/groups/joined` |
| 40 | ✅ | Get group info | GET | `/groups/{group_jid}` |
| 41 | ✅ | Create new group | POST | `/groups` |
| 42 | ✅ | Join group via invite link | POST | `/groups/{group_jid}/join-invite` |
| 43 | ✅ | Get group invite link | GET | `/groups/{group_jid}/invite-link` |
| 44 | ✅ | Get group info from invite | GET | `/groups/invite/{invite_code}` |
| 45 | ✅ | Leave group | POST | `/groups/{group_jid}/leave` |
| 46 | ✅ | Add participants | POST | `/groups/{group_jid}/participants` |
| 47 | ✅ | Remove participants | DELETE | `/groups/{group_jid}/participants` |
| 48 | ✅ | Promote to admin | POST | `/groups/{group_jid}/admins` |
| 49 | ✅ | Demote admin | DELETE | `/groups/{group_jid}/admins` |
| 50 | ✅ | Update group name | PATCH | `/groups/{group_jid}/name` |
| 51 | ✅ | Update group description | PATCH | `/groups/{group_jid}/description` |
| 52 | ✅ | Update group photo | POST | `/groups/{group_jid}/photo` |
| 53 | ✅ | Update group settings | PATCH | `/groups/{group_jid}/settings` |
| 54 | ✅ | Set group topic | PATCH | `/groups/{group_jid}/topic` |
| 55 | ✅ | Get participant requests | GET | `/groups/{group_jid}/participant-requests` |
| 56 | ✅ | Approve join requests | POST | `/groups/{group_jid}/requests/approve` |
| 57 | ✅ | Reject join requests | POST | `/groups/{group_jid}/requests/reject` |
| 58 | ✅ | Set join approval mode | POST | `/groups/{group_jid}/join-approval` |
| 59 | ✅ | Set member add mode | PATCH | `/groups/{group_jid}/member-add-mode` |
| 60 | ✅ | Link group to community | POST | `/groups/{parent_group_jid}/link/{group_jid}` |
| 61 | ✅ | Get community participants | GET | `/groups/{community_jid}/linked-participants` |
| 62 | ✅ | Get community subgroups | GET | `/groups/{community_jid}/subgroups` |
| 63 | ✅ | Send chat presence (typing) | POST | `/chats/{chat_jid}/presence` |
| 64 | ✅ | Update presence status | POST | `/presence/status` |
| 65 | ✅ | Archive/unarchive chat | POST | `/chats/{chat_jid}/archive` |
| 66 | ✅ | Pin/unpin chat | POST | `/chats/{chat_jid}/pin` |
| 67 | ✅ | Set disappearing timer | PATCH | `/chats/{chat_jid}/disappearing-timer` |
| 68 | ✅ | Send poll vote | POST | `/chats/{chat_jid}/polls/{message_id}/votes` |
| 69 | ✅ | Get subscribed newsletters | GET | `/newsletters/subscribed` |
| 70 | ✅ | Get newsletter info | GET | `/newsletters/{jid}` |
| 71 | ✅ | Follow newsletter | POST | `/newsletters/{jid}/follow` |
| 72 | ✅ | Unfollow newsletter | POST | `/newsletters/{jid}/unfollow` |
| 73 | ✅ | Get newsletter messages | GET | `/newsletters/{jid}/messages` |
| 74 | ✅ | Publish message to newsletter | POST | `/newsletters/{jid}/messages` |
| 75 | ✅ | React to newsletter | POST | `/newsletters/{jid}/reactions` |
| 76 | ✅ | Mark newsletter as viewed | POST | `/newsletters/{jid}/mark-viewed` |
| 77 | ✅ | Create newsletter | POST | `/newsletters` |
| 78 | ✅ | Get newsletter updates | GET | `/newsletters/{jid}/updates` |
| 79 | ✅ | Mute/unmute newsletter | POST | `/newsletters/{jid}/mute` |
| 80 | ✅ | Subscribe to live updates | POST | `/newsletters/{jid}/live-updates` |
| 81 | ✅ | Unsubscribe from updates | DELETE | `/newsletters/{jid}/live-updates` |
| 82 | ✅ | Get info from invite | GET | `/newsletters/invite/{invite_key}` |
| 83 | ✅ | Get business profile | GET | `/business/{jid}` |
| 84 | ✅ | Resolve business link | GET | `/business/links/{code}` |
| 85 | ✅ | Check contacts in batch | POST | `/contacts/check` |
| 86 | ✅ | Download Facebook media | POST | `/media/fb/download` |
| 87 | ✅ | List available bots | GET | `/bots` |
| 88 | ✅ | Get bot profile | GET | `/bots/{bot_jid}` |
| 89 | ✅ | Get contact QR code | GET | `/qr/contact` |
| 90 | ✅ | Revoke contact QR | POST | `/qr/contact/revoke` |
| 91 | ✅ | Resolve contact QR | POST | `/qr/contact/resolve/{code}` |
| 92 | ✅ | Register push config | POST | `/push/register` |
| 93 | ✅ | Get push config | GET | `/push/config` |
| 94 | ✅ | Swagger UI | GET | `/docs/*` |
| 95 | ✅ | OpenAPI JSON | GET | `/docs/swagger.json` |
| 96 | ✅ | OpenAPI YAML | GET | `/docs/swagger.yaml` |
| 97 | ✅ | Function reference | GET | `/docs/function` |

✅ = Available and tested | **Total: 97 endpoints**

## 🔐 Authentication Flow

This API uses a two-layer authentication system:

### 1. Device Registration (Basic Auth)

First, register a device to obtain a JWT token:

```bash
POST /device/add
Authorization: Basic base64(phone_number:password)

# Example:
curl -X POST "http://localhost:3000/device/add" \
  -u "6281234567890:your_password"
```

**Response:**
```json
{
  "code": 200,
  "message": "Success",
  "data": {
    "token": "eyJhbGc...",
    "jid": "6281234567890",
    "device_id": "550e8400-e29b-41d4-a716-446655440000"
  }
}
```

### 2. JWT Token for All Operations

Use the obtained JWT token for all subsequent API calls:

```bash
Authorization: Bearer eyJhbGc...
```

### JWT Payload Structure

- `dat.jid` — WhatsApp account JID (from Basic Auth username)
- `dat.device_id` — Application-generated UUID for this device session

### Device-Scoped Sessions

Each JWT token represents **exactly one device session**. Key points:

- ✅ Each device has its own independent session
- ✅ Multiple devices per WhatsApp account (up to 4)
- ✅ Re-calling `/device/add` creates a new device session
- ✅ Token expiration is configurable via `AUTH_JWT_EXPIRED_HOUR`
- ✅ Sessions persist across app restarts (stored in SQLite)

### Session Lifecycle

1. **Register** → Call `POST /device/add` to get JWT
2. **Login** → Use JWT with `POST /devices/{device_id}/login` or `POST /devices/{device_id}/login-code`
3. **Operate** → Use same JWT for all API operations
4. **Logout** → Call `DELETE /devices/{device_id}/session` to disconnect

## 🔧 Environment Variables

| Variable | Description | Default | Example |
|----------|-------------|---------|---------|
| **Authentication** | | | |
| `AUTH_BASIC_USERNAME` | Basic auth username (optional) | - | `admin` |
| `AUTH_BASIC_PASSWORD` | Basic auth password (optional) | - | `secret123` |
| `AUTH_JWT_SECRET` | JWT signing secret key | - | `your-jwt-secret-key` |
| `AUTH_JWT_EXPIRED_HOUR` | JWT expiration in hours (0=never) | `0` | `24` |
| **Server** | | | |
| `SERVER_ADDRESS` | Server listening address | `127.0.0.1` | `0.0.0.0` |
| `SERVER_PORT` | Server listening port | `3000` | `8080` |
| **HTTP Configuration** | | | |
| `HTTP_BASE_URL` | Base URL path for API | `` | `/api/v1` |
| `HTTP_CORS_ORIGIN` | CORS allowed origins | `*` | `https://example.com` |
| `HTTP_BODY_LIMIT_SIZE` | Max request body size | `8M` | `50M` |
| `HTTP_GZIP_LEVEL` | GZIP compression level (1-9) | `1` | `6` |
| `HTTP_CACHE_CAPACITY` | In-memory cache capacity | `100` | `500` |
| `HTTP_CACHE_TTL_SECONDS` | Cache TTL in seconds | `5` | `300` |
| **WhatsApp** | | | |
| `WHATSAPP_DATASTORE_TYPE` | Database type | - | `sqlite3` |
| `WHATSAPP_DATASTORE_URI` | Database connection URI | - | `file:dbs/WhatsApp.db?_foreign_keys=on` |
| `WHATSAPP_CLIENT_PROXY_URL` | HTTP proxy for WhatsApp | - | `http://proxy.example.com:8080` |
| `WHATSAPP_VERSION_MAJOR` | WhatsApp client major version | - | `2` |
| `WHATSAPP_VERSION_MINOR` | WhatsApp client minor version | - | `3000` |
| `WHATSAPP_VERSION_PATCH` | WhatsApp client patch version | - | `1015901307` |
| `WHATSAPP_MEDIA_IMAGE_CONVERT_WEBP` | Convert images to WebP | `false` | `true` |
| `WHATSAPP_MEDIA_IMAGE_COMPRESSION` | Enable image compression | `false` | `true` |

### Example `.env` File

```env
# Authentication
AUTH_BASIC_USERNAME=admin
AUTH_BASIC_PASSWORD=ThisIsSecretPassword
AUTH_JWT_SECRET=your-super-secret-jwt-key-change-this
AUTH_JWT_EXPIRED_HOUR=24

# Server
SERVER_ADDRESS=0.0.0.0
SERVER_PORT=3000

# HTTP
HTTP_BASE_URL=
HTTP_CORS_ORIGIN=*
HTTP_BODY_LIMIT_SIZE=50M

# WhatsApp
WHATSAPP_DATASTORE_TYPE=sqlite3
WHATSAPP_DATASTORE_URI=file:dbs/WhatsApp.db?_foreign_keys=on
WHATSAPP_MEDIA_IMAGE_COMPRESSION=true
```

## 🐳 Docker Deployment

### Basic Docker Run

```bash
docker run -d \
  --name whatsapp-api \
  --restart always \
  -p 3000:3000 \
  -v $(pwd)/dbs:/app/dbs \
  -e AUTH_BASIC_USERNAME=admin \
  -e AUTH_BASIC_PASSWORD=secret \
  -e AUTH_JWT_SECRET=your-jwt-secret \
  gdbrns/go-whatsapp-multi-session-rest-api
```

### Docker Compose (Production)

```yaml
version: '3.8'

services:
  whatsapp-api:
    image: gdbrns/go-whatsapp-multi-session-rest-api:latest
    container_name: whatsapp-api
    restart: always
    ports:
      - "3000:3000"
    volumes:
      - whatsapp-data:/app/dbs
    environment:
      # Authentication
      AUTH_BASIC_USERNAME: admin
      AUTH_BASIC_PASSWORD: ${WHATSAPP_PASSWORD}
      AUTH_JWT_SECRET: ${JWT_SECRET}
      AUTH_JWT_EXPIRED_HOUR: 24
      
      # Server
      SERVER_ADDRESS: 0.0.0.0
      SERVER_PORT: 3000
      
      # WhatsApp
      WHATSAPP_DATASTORE_TYPE: sqlite3
      WHATSAPP_DATASTORE_URI: file:dbs/WhatsApp.db?_foreign_keys=on
      WHATSAPP_MEDIA_IMAGE_COMPRESSION: "true"
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:3000/"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s

volumes:
  whatsapp-data:
    driver: local
```

### Build Custom Image

```bash
# Clone repository
git clone https://github.com/gdbrns/go-whatsapp-multi-session-rest-api.git
cd go-whatsapp-multi-session-rest-api/rest-api

# Build image
docker build -t whatsapp-api:custom .

# Run custom image
docker run -d -p 3000:3000 \
  -v whatsapp-data:/app/dbs \
  whatsapp-api:custom
```

## 🧪 Testing

```bash
# Run tests
go test ./...

# Run tests with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run specific package tests
go test ./pkg/whatsapp/...

# Integration tests (requires running instance)
go test -tags=integration ./tests/...
```

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request. For major changes:

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

Please make sure to:
- ✅ Update tests as appropriate
- ✅ Update documentation
- ✅ Follow Go best practices
- ✅ Add examples if needed

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## ⚠️ Disclaimer

**Important Legal Information:**

- This project is **unofficial** and **not affiliated** with WhatsApp Inc. or Meta Platforms Inc.
- This software is provided "as is" without warranty of any kind
- Use at your own risk and in accordance with [WhatsApp's Terms of Service](https://www.whatsapp.com/legal/terms-of-service)
- The authors and contributors are not responsible for any misuse or violation of WhatsApp's terms
- This project is intended for **educational and personal use** only
- Commercial use should comply with WhatsApp Business API terms

**Recommended Usage:**

- ✅ Personal automation and notifications
- ✅ Small business internal tools
- ✅ Educational purposes
- ❌ Mass messaging / spam
- ❌ Violating WhatsApp Terms of Service
- ❌ Commercial use without proper authorization

## 📞 Support & Community

- 📖 **Documentation**: [bump.sh](https://bump.sh/gdbrns/doc/go-whatsapp-multi-session-rest-api)
- 🐛 **Issues**: [GitHub Issues](https://github.com/gdbrns/go-whatsapp-multi-session-rest-api/issues)
- 💬 **Discussions**: [GitHub Discussions](https://github.com/gdbrns/go-whatsapp-multi-session-rest-api/discussions)
- 💻 **Repository**: [github.com/gdbrns/go-whatsapp-multi-session-rest-api](https://github.com/gdbrns/go-whatsapp-multi-session-rest-api)

## 🙏 Acknowledgments

This project builds upon excellent work from the community:

### Core Technologies

- **[whatsmeow v0.0.0-20250930215512](https://pkg.go.dev/go.mau.fi/whatsmeow@v0.0.0-20250930215512-38f9aaa3ba7c)** - The foundation of this project. We use the **very latest version (September 30, 2025)** of whatsmeow for maximum compatibility and newest features with WhatsApp Web Multi-Device protocol. This ensures cutting-edge support for all WhatsApp features including groups, newsletters, business accounts, and media handling.
  - 📦 **Package**: `go.mau.fi/whatsmeow@v0.0.0-20250930215512-38f9aaa3ba7c`
  - 📅 **Version Date**: September 30, 2025
  - 🔗 **Documentation**: [pkg.go.dev](https://pkg.go.dev/go.mau.fi/whatsmeow@v0.0.0-20250930215512-38f9aaa3ba7c)
  - 💡 **Status**: Latest stable release with full Multi-Device support

- **[Fiber](https://github.com/gofiber/fiber)** - Express-inspired web framework for Go, providing high performance and clean API routing

### Inspiration & Code References

This project was built with inspiration and friendly code references from:

- **[aldinokemal/go-whatsapp-web-multidevice](https://github.com/aldinokemal/go-whatsapp-web-multidevice)** - Excellent WhatsApp API implementation with UI and comprehensive features. Many API design patterns and approaches were inspired by this project.

- **[dimaskiddo/go-whatsapp-multidevice-rest](https://github.com/dimaskiddo/go-whatsapp-multidevice-rest)** - Clean REST API architecture and authentication patterns that influenced our implementation approach.

**Special Thanks** to these projects for paving the way and making WhatsApp automation accessible to the Go community. This project aims to provide a production-ready, RESTful alternative with device-scoped sessions and the latest whatsmeow features.

### Additional Libraries
- [go-sqlite3](https://github.com/mattn/go-sqlite3) - SQLite driver for Go
- [jwt-go](https://github.com/golang-jwt/jwt) - JWT implementation for Go
- [godotenv](https://github.com/joho/godotenv) - Environment variable management

## 🔗 Related Projects

- [whatsmeow](https://github.com/tulir/whatsmeow) - WhatsApp Web Multi-Device protocol library
- [go-whatsapp-web-multidevice](https://github.com/aldinokemal/go-whatsapp-web-multidevice) - Alternative WhatsApp API with UI

---

**Made with ❤️ using Go**

⭐ Star this repository if you find it useful!
