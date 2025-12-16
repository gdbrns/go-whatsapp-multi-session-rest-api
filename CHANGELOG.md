# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.1.0] - 2025-12-17

### 🚀 New Features

#### CI/CD Automation
- **GitHub Actions CI**: Automated testing, linting (golangci-lint), and multi-platform builds (Linux, Windows, macOS)
- **GitHub Actions Release**: Automated releases with GoReleaser on tag push
- **GitHub Actions Docker**: Multi-platform Docker images (amd64, arm64) published to GitHub Container Registry
- **SBOM Generation**: Software Bill of Materials for supply chain security

### 🔒 Security Hardening

#### Docker Security
- Run container as non-root user (`appuser:appgroup`)
- Pin Alpine base image to v3.21 for reproducibility
- Enable `no-new-privileges` security option
- Read-only root filesystem with tmpfs for `/tmp`
- Resource limits (CPU: 2 cores, Memory: 512M)

#### Build Security
- Static binary linking with `-extldflags '-static'`
- Reproducible builds with `-trimpath` flag
- Dependency verification with `go mod verify`

### ⚡ Performance Improvements
- Init process for proper signal handling and zombie reaping
- Graceful shutdown with 30s stop grace period
- Optimized Docker layer caching

### 🔄 Updated

#### Dependencies
- **whatsmeow**: `v0.0.0-20251203` → `v0.0.0-20251216102424-56a8e44b0cec`
- **golang.org/x/sync**: `v0.18.0` → `v0.19.0`
- **golang.org/x/crypto**: `v0.44.0` → `v0.46.0`
- **google.golang.org/protobuf**: `v1.36.10` → `v1.36.11`
- **go.mau.fi/util**: `v0.9.3` → `v0.9.4`

### 📚 Documentation
- Updated README badges for whatsmeow v0.0.0-20251216
- Enhanced acknowledgments for @tulir and mautrix ecosystem
- Added release notes templates in GoReleaser

---

## [1.0.0] - 2025-12-11

### 🎉 First Official Release

Enterprise-grade RESTful API for WhatsApp Multi-Device and Multi-Session implementation with **111+ endpoints** and **31 webhook event types**.

### ✨ Features

#### Core Messaging
- Send text messages with typing/presence simulation
- Send images with captions and view-once support
- Send videos (MP4, 3GP, MOV, WebM)
- Send audio and voice notes (MP3, OGG, WAV)
- Send stickers (WebP format)
- Send locations with coordinates
- Send contacts (vCard format)
- Send documents with custom filenames
- Message actions: read, react, edit, delete, reply, forward

#### 📊 Polls
- Create polls with 2-12 options
- Single or multi-answer polls
- Vote on existing polls
- Poll results via webhook events
- Delete poll messages

#### 📢 Newsletter/Channels
- List subscribed newsletters
- Create new newsletters/channels
- Follow/unfollow newsletters
- Get newsletter info and messages
- Send messages to newsletters (admin)
- React to newsletter messages
- Mute/unmute newsletters
- Mark messages as viewed
- Get newsletter info from invite codes
- Subscribe to live updates
- Update newsletter photos

#### 📱 Status/Stories
- Post text status with background colors
- Post image status with captions
- Post video status
- Get status updates
- Delete own status
- Get user's about status

#### 👥 Group Management
- Create, update, delete groups
- Manage participants (add, remove, promote, demote)
- Update group settings (name, description, photo)
- Generate/revoke invite links
- Join approval management
- Community and subgroup support
- Link/unlink groups

#### 👤 User Management
- Get user info and profile pictures
- Block/unblock users
- Privacy settings management
- Contact sync
- Get blocklist

#### 🔐 Authentication (3-Tier)
- **Admin Auth** (X-Admin-Secret): API key management
- **API Key Auth** (X-API-Key): Device creation
- **JWT Bearer**: All device operations

#### 🔔 Webhooks (31 Event Types)
- **Messages** (5): received, delivered, read, played, deleted
- **Connection** (6): connected, disconnected, logged_out, reconnecting, keepalive_timeout, temporary_ban
- **Calls** (4): offer, accept, terminate, reject
- **Groups** (4): join, leave, participant_update, info_update
- **Newsletter** (4): join, leave, message_received, update
- **Polls** (3): created, vote, update
- **Status** (3): posted, viewed, deleted
- **Media** (2): received, downloaded
- **Contact** (2): update, blocklist.change
- **App State** (2): sync_complete, patch_received
- **History** (1): sync

#### ⚡ Performance
- Rate limiting per device
- Typing/presence simulation
- Read receipt jitter
- Group list caching with TTL
- Request deduplication (singleflight)
- Parallel group processing
- IsOnWhatsApp caching and batching

#### 🛡️ Security
- HMAC-SHA256 webhook signatures
- JWT tokens with optional expiry
- Multi-tenant API key isolation
- Input validation on all endpoints

### 📚 Documentation
- OpenAPI/Swagger 2.0 specification
- Comprehensive webhook events documentation
- API organized by usage frequency

### 🏗️ Technical Stack
- Go 1.21+
- Fiber v2 web framework
- whatsmeow library
- PostgreSQL/MySQL support
- Docker ready

---

## Links

- **GitHub**: https://github.com/gdbrns/go-whatsapp-multi-session-rest-api
- **Documentation**: `/docs/` endpoint (Swagger UI)
- **Issues**: https://github.com/gdbrns/go-whatsapp-multi-session-rest-api/issues

