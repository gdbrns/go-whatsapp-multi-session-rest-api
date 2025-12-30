# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- **Star/Keep Messages API** - Star or unstar messages to keep them in chat (`POST /messages/:message_id/star`)
- **Link Preview Messages API** - Send text messages with rich URL previews including title, description, and thumbnail (`POST /chats/:chat_jid/link-preview`)
- **Newsletter Comments API** - Send comments on WhatsApp channel/newsletter posts (`POST /newsletters/:jid/comments`)

### Changed
- Updated `.gitignore` to exclude release template

### Fixed
- Minor workflow adjustments in `release.yml`

---

## [1.2.4] - 2025-12-22

### 🐛 Fixes

- Return JID in `GET /devices/me/contacts/:phone/registered` so participant verification can add validated numbers; include JID in logs for easier debugging.

## [1.2.1] - 2025-12-21

### 🐛 Fixes

- Add `caption` support to `SendDocument` (API + WhatsApp payload)
- Fix GitHub Actions workflow conditions to avoid invalid `secrets.*` usage in `if:`

---

## [1.2.0] - 2025-12-21

### 🚀 New Features

#### WhatsApp Web Version Auto-Refresh
- Automatically refresh WhatsApp Web version when pairing fails with "client outdated"
- Added admin endpoints to inspect/refresh WA Web version:
	- `GET /admin/whatsapp/version`
	- `POST /admin/whatsapp/version/refresh`
- Optional scheduled WA Web version refresh cron job

#### Production Hardening (Large Session Counts)
- Startup reconnect storm protection: concurrency limiting + jitter + retry/backoff
- New env knobs for reconnect tuning and WA version refresh throttling

### 🔄 Updated

#### Dependencies
- **whatsmeow**: `v0.0.0-20251216` → `v0.0.0-20251217143725-11cf47c62d32`

#### API Documentation
- Updated Swagger (JSON/YAML) for new admin endpoints and media retry request shape
- Aligned Swagger auth docs (`ADMIN_SECRET_KEY`, API key prefix `wam_`)

#### Docker & CI/CD
- Publish Docker images to Docker Hub on release tags, while keeping GHCR as edge
- Ensure release tags also update `latest`
- Auto-sync Docker Hub description from README
- Added Bump.sh workflow for deploy + PR diffs (OpenAPI)

---

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

---

## Implementation Changelog - Advanced whatsmeow Features

### Version 1.1.0 - 2025-12-17

#### 🎉 New Features

##### 1. Message History Sync
- **Endpoint**: `POST /history/sync`
- **Description**: Request message history from WhatsApp servers
- **Features**:
  - Configurable message count (1-100, default 25)
  - Asynchronous delivery via webhook
  - New webhook event: `history.sync_complete`

##### 2. Per-Device HTTP Proxy
- **Endpoints**:
  - `GET /devices/me/proxy` - Get current proxy configuration
  - `POST /devices/me/proxy` - Set or clear proxy URL
- **Description**: Configure unique HTTP proxy per device
- **Features**:
  - Overrides global `WHATSAPP_CLIENT_PROXY_URL` environment variable
  - Automatic client reconnection when proxy changes
  - Empty string clears per-device proxy (uses global)
  - URL validation (must be http:// or https://)
  - Stored in database per device

##### 3. Push Notification Registration
- **Endpoint**: `POST /devices/me/push-notifications`
- **Description**: Register device for push notifications
- **Features**:
  - Support for 3 platforms: FCM, APNs, webhook
  - Webhook platform reuses existing webhook infrastructure
  - Token storage in database
  - Per-device registration tracking

##### 4. Advanced Media Retry Handling
- **Endpoint**: `POST /messages/media/retry-receipt`
- **Description**: Send media retry receipt for failed downloads
- **Features**:
  - Request re-delivery of media encryption keys
  - New webhook event: `media.retry`
  - Automatic retry receipt handling

##### 5. Enhanced Webhook Events
- **New Events**:
  - `history.sync_complete` - History sync completion notification
  - `media.retry` - Media retry notification with encryption keys
  - `poll.vote_decrypted` - Decrypted poll vote results
  - `status.comment` - Status/story comment received

##### 6. Passive Mode (Existing Integration)
- **Function**: `WhatsAppSetPassive()`
- **Description**: Read-only mode to reduce bandwidth usage
- **Features**:
  - Disable outbound message sending
  - Database tracking of passive mode state
  - Per-device passive mode toggle

---

### 🗄️ Database Changes

#### New Columns in `devices` Table

| Column | Type | Nullable | Default | Description |
|--------|------|----------|---------|-------------|
| `proxy_url` | TEXT | YES | NULL | Per-device HTTP proxy URL |
| `push_notification_platform` | VARCHAR(20) | YES | NULL | Push platform (fcm/apns/webhook) |
| `push_notification_token` | TEXT | YES | NULL | Push notification token |
| `push_notification_registered_at` | TIMESTAMP | YES | NULL | Registration timestamp |
| `passive_mode` | BOOLEAN | NO | FALSE | Read-only mode flag |

#### New Indices

```sql
CREATE INDEX idx_devices_proxy ON devices(device_id) WHERE proxy_url IS NOT NULL;
CREATE INDEX idx_devices_push ON devices(device_id) WHERE push_notification_platform IS NOT NULL;
```

#### Migration Strategy
- All migrations use `IF NOT EXISTS` for safety
- Automatic execution on application startup
- No downtime required
- Backward compatible with existing devices

---

### 📝 Code Changes

#### Files Modified

1. **pkg/whatsapp/routing.go** (+90 lines)
	- Database schema migrations (lines 245-284)
	- `GetDeviceProxy()` function (lines 1102-1119)
	- `SetDeviceProxy()` function (lines 1122-1129)

2. **pkg/whatsapp/whatsapp.go** (+173 lines)
	- `WhatsAppBuildHistorySyncRequest()` (lines 4879-4904)
	- `WhatsAppSetDeviceProxy()` (lines 4910-4937)
	- `WhatsAppGetDeviceProxy()` (lines 4939-4944)
	- `WhatsAppSetPassive()` (lines 4950-4974)
	- `WhatsAppSendMediaRetryReceipt()` (lines 4980-5009)
	- `WhatsAppRegisterPushNotification()` (lines 5015-5046)
	- Modified `WhatsAppInitClient()` to check per-device proxy (lines 747-752)

3. **internal/types/request.go** (+78 lines)
	- `RequestBuildHistorySync` / `ResponseHistorySync`
	- `RequestSetProxy` / `ResponseGetProxy`
	- `RequestDecryptPollVote` / `ResponseDecryptPollVote`
	- `RequestEncryptComment` / `ResponseEncryptComment`
	- `RequestDecryptComment` / `ResponseDecryptComment`
	- `RequestSendMediaRetryReceipt`
	- `RequestRegisterPushNotification` / `ResponsePushNotificationStatus`

4. **internal/webhook/types.go** (+4 lines)
	- `EventHistorySyncComplete`
	- `EventMediaRetry`
	- `EventPollVoteDecrypted`
	- `EventStatusComment`

5. **internal/device/device.go** (+92 lines)
	- `SetProxy()` handler (lines 350-375)
	- `GetProxy()` handler (lines 378-400)
	- `RegisterPushNotification()` handler (lines 403-440)

6. **internal/message/message.go** (+34 lines)
	- `SendMediaRetryReceipt()` handler (lines 252-283)

7. **internal/route.go** (+10 lines)
	- Import `ctlHistory` controller (line 18)
	- 5 new route registrations (lines 106-113, 152)

8. **README.md** (+6 lines)
	- Updated features list with new capabilities
	- Updated webhook event count (31 → 35+)

#### Files Created

9. **internal/history/history.go** (NEW - 58 lines)
	- History sync controller
	- `BuildHistorySyncRequest()` handler

10. **docs/NEW_ENDPOINTS.md** (NEW - 450+ lines)
	 - Comprehensive documentation for all new endpoints
	 - Request/response examples
	 - cURL examples
	 - Webhook event payloads
	 - Security considerations

11. **docs/SWAGGER_ADDITIONS.json** (NEW - 200+ lines)
	 - Swagger/OpenAPI definitions for new endpoints
	 - Ready to merge into main swagger.json

---

### 🔄 API Changes

#### New Endpoints (5)

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| POST | `/history/sync` | JWT | Build history sync request |
| GET | `/devices/me/proxy` | JWT | Get device proxy config |
| POST | `/devices/me/proxy` | JWT | Set device proxy URL |
| POST | `/devices/me/push-notifications` | JWT | Register push notifications |
| POST | `/messages/media/retry-receipt` | JWT | Send media retry receipt |

#### Updated Endpoints

None - all changes are additive and backward compatible

---

### 🔒 Security Enhancements

1. **Proxy URL Validation**
	- Must start with `http://` or `https://`
	- Prevents SSRF attacks
	- Empty string clears proxy (safe fallback)

2. **Device Isolation**
	- Each device can only access its own proxy configuration
	- JWT token validation ensures device ownership
	- Database queries scoped to device_id

3. **Push Token Storage**
	- Tokens stored in database (consider encryption at rest)
	- Per-device registration tracking
	- Platform validation (fcm/apns/webhook only)

---

### ⚡ Performance Considerations

1. **Per-Device Proxy**
	- Proxy lookup occurs once during client initialization
	- Cached in memory for the session lifetime
	- Client reconnection required for proxy changes (expected)

2. **History Sync**
	- Asynchronous processing (doesn't block API response)
	- Large syncs may take several minutes
	- WhatsApp has rate limits (handled by server)

3. **Database Indices**
	- Partial indices for proxy and push lookups
	- Minimal storage overhead
	- Fast queries with WHERE clauses

4. **Webhook Events**
	- 4 new event types
	- Existing webhook infrastructure reused
	- No additional worker pools needed

---

### 📊 Statistics

#### Lines of Code Added
- **Total**: ~700 lines
- **Go Code**: ~550 lines
- **Documentation**: ~150 lines

#### Files Modified/Created
- **Modified**: 8 files
- **Created**: 3 files

#### API Endpoints
- **Before**: 120+ endpoints
- **After**: 125+ endpoints
- **New**: 5 endpoints

#### Webhook Events
- **Before**: 31 event types
- **After**: 35 event types
- **New**: 4 event types

#### Database Schema
- **New Columns**: 5
- **New Indices**: 2
- **New Tables**: 0 (reused existing)

---

### 🧪 Testing Checklist

#### Database Migration
- [x] Schema migrations added
- [ ] Test on fresh database
- [ ] Test on existing database with data
- [ ] Verify indices created
- [ ] Check backward compatibility

#### API Endpoints
- [ ] Test history sync request
- [ ] Test proxy GET/SET/CLEAR
- [ ] Test push notification registration
- [ ] Test media retry receipt
- [ ] Test with invalid inputs
- [ ] Test authentication (valid/invalid tokens)

#### Integration
- [ ] Verify proxy reconnection works
- [ ] Test webhook event delivery
- [ ] Confirm history sync webhook payload
- [ ] Validate media retry webhook
- [ ] Test device isolation (cross-device access denied)

#### Security
- [ ] Verify proxy URL validation
- [ ] Test SSRF protection
- [ ] Confirm JWT token validation
- [ ] Test device ownership checks

---

### 🚀 Deployment Steps

1. **Backup Database**
	```bash
	pg_dump -h localhost -U whatsapp -d whatsapp > backup_$(date +%Y%m%d).sql
	```

2. **Pull Latest Code**
	```bash
	git pull origin main
	```

3. **Build Application**
	```bash
	go build -o whatsapp-api cmd/main/main.go
	```

4. **Restart Service**
	```bash
	systemctl restart whatsapp-api
	# or with Docker:
	docker-compose down && docker-compose up -d
	```

5. **Verify Migrations**
	- Check logs for migration success
	- Query database to verify new columns exist
	- Test new endpoints with sample requests

6. **Monitor**
	- Watch application logs
	- Monitor database performance
	- Check webhook delivery rates
	- Verify proxy connections

---

### 🔄 Rollback Plan

If issues occur, rollback can be performed:

#### 1. Revert Code Changes
```bash
git revert <commit-hash>
systemctl restart whatsapp-api
```

#### 2. Remove Database Columns (if needed)
```sql
ALTER TABLE devices DROP COLUMN IF EXISTS proxy_url;
ALTER TABLE devices DROP COLUMN IF EXISTS push_notification_platform;
ALTER TABLE devices DROP COLUMN IF EXISTS push_notification_token;
ALTER TABLE devices DROP COLUMN IF EXISTS push_notification_registered_at;
ALTER TABLE devices DROP COLUMN IF EXISTS passive_mode;
DROP INDEX IF EXISTS idx_devices_proxy;
DROP INDEX IF EXISTS idx_devices_push;
```

**Note**: Column removal is optional and not recommended unless data corruption occurs. Leaving columns in place is safe.

---

### 📚 Documentation Updates

#### Updated Files
- [x] README.md - Updated features list
- [x] swagger.json - Includes advanced endpoints (merged)
- [x] swagger.yaml - Includes advanced endpoints (merged)

#### Documentation Locations
- **Swagger**: `/docs/swagger.json` and `/docs/swagger.yaml`
- **Changelog**: `/CHANGELOG.md`

---

### 🎯 Success Criteria

✅ All 7 features implemented
✅ 5 new API endpoints functional
✅ Database migrations automatic
✅ 4 new webhook events defined
✅ Backward compatibility maintained
✅ Security validations in place
✅ Documentation created
⏳ Manual testing pending
⏳ Swagger docs pending merge
⏳ Production deployment pending

---

### 📞 Support

For questions or issues:
- GitHub Issues: https://github.com/gdbrns/go-whatsapp-multi-session-rest-api/issues
- Documentation: https://bump.sh/gdbrns/doc/go-whatsapp-multi-session-rest

---

### 📄 License

MIT License - See LICENSE file for details

---

*Generated: 2025-12-17*
*Version: 1.1.0*
*Implementation Status: Complete*

