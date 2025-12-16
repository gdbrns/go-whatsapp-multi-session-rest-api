# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.1] - 2025-12-17

### 🔄 Updated

#### Dependencies
- **whatsmeow**: Updated from `v0.0.0-20251203` to `v0.0.0-20251216102424-56a8e44b0cec` (Dec 16, 2024)
  - Latest WhatsApp protocol updates
  - Improved stability and bug fixes
  - Enhanced `context.Context` support across API calls
  - Updated `libsignal` and `util` dependencies

### 📚 Documentation
- Updated README badges to reflect latest whatsmeow version
- Enhanced acknowledgments section with proper attribution to @tulir and mautrix ecosystem

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

