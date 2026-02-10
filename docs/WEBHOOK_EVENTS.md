# Webhook Events Reference

Complete documentation for all WhatsApp API webhook event types and their payload structures.

## Table of Contents

- [Overview](#overview)
- [Event Envelope](#event-envelope)
- [Message Events](#message-events)
- [Connection Events](#connection-events)
- [Call Events](#call-events)
- [Group Events](#group-events)
- [Contact Events](#contact-events)
- [App State Events](#app-state-events)
- [History Sync Events](#history-sync-events)
- [Newsletter Events](#newsletter-events)
- [Webhook Configuration](#webhook-configuration)
- [Security](#security)

---

## Overview

When WhatsApp events occur, webhooks are triggered and HTTP POST requests are sent to your configured endpoint. Each device can have multiple webhooks configured, and each webhook can filter which event types it receives.

### Supported Event Types (85 Total)

This API emits all events available in the latest whatsmeow release, plus a few derived convenience events (poll/media/status). See the **Quick Reference** section for the full list.

Category summary:

- Messages, polls, media, status
- Connection and pairing
- Calls
- Groups
- Contacts, profiles, privacy, presence
- App state, labels, chat settings
- History and offline sync
- Newsletters
- Blocklist

---

## Event Envelope

All webhook payloads follow this structure:

```json
{
  "event_type": "message.received",
  "device_id": "abc123def456-ghi789",
  "timestamp": "2024-12-09T13:37:04.123456Z",
  "data": {
    // Event-specific data
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| `event_type` | string | The event type identifier |
| `device_id` | string | UUID of the device that triggered the event |
| `timestamp` | string | ISO 8601 timestamp when the event occurred |
| `data` | object | Event-specific payload data |

---

## Message Events

### `message.received`

Triggered when a new message is received.

```json
{
  "event_type": "message.received",
  "device_id": "abc123def456-ghi789",
  "timestamp": "2024-12-09T13:37:04.123456Z",
  "data": {
    "message_id": "3EB0ABC123DEF456789",
    "from": "6281234567890@s.whatsapp.net",
    "chat": "6281234567890@s.whatsapp.net",
    "timestamp": 1702129024,
    "is_from_me": false
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| `message_id` | string | Unique message identifier |
| `from` | string | Sender's JID (WhatsApp ID) |
| `chat` | string | Chat JID (same as `from` for private chats, group JID for groups) |
| `timestamp` | integer | Unix timestamp of the message |
| `is_from_me` | boolean | `true` if sent by the connected device |

**Note:** For group messages, `from` is the sender and `chat` is the group JID (e.g., `120363123456789012@g.us`).

---

### `message.delivered`

Triggered when a sent message is delivered to the recipient's device.

```json
{
  "event_type": "message.delivered",
  "device_id": "abc123def456-ghi789",
  "timestamp": "2024-12-09T13:37:04.123456Z",
  "data": {
    "message_id": "3EB0ABC123DEF456789",
    "chat": "6281234567890@s.whatsapp.net",
    "sender": "6281234567890@s.whatsapp.net",
    "timestamp": 1702129024
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| `message_id` | string | Unique message identifier |
| `chat` | string | Chat JID where the message was delivered |
| `sender` | string | JID of the recipient who received the message |
| `timestamp` | integer | Unix timestamp of the delivery receipt |

---

### `message.read`

Triggered when a sent message is read by the recipient.

```json
{
  "event_type": "message.read",
  "device_id": "abc123def456-ghi789",
  "timestamp": "2024-12-09T13:37:04.123456Z",
  "data": {
    "message_id": "3EB0ABC123DEF456789",
    "chat": "6281234567890@s.whatsapp.net",
    "sender": "6281234567890@s.whatsapp.net",
    "timestamp": 1702129024
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| `message_id` | string | Unique message identifier |
| `chat` | string | Chat JID where the message was read |
| `sender` | string | JID of the recipient who read the message |
| `timestamp` | integer | Unix timestamp of the read receipt |

---

### `message.played`

Triggered when a voice message or video is played by the recipient.

```json
{
  "event_type": "message.played",
  "device_id": "abc123def456-ghi789",
  "timestamp": "2024-12-09T13:37:04.123456Z",
  "data": {
    "message_id": "3EB0ABC123DEF456789",
    "chat": "6281234567890@s.whatsapp.net",
    "sender": "6281234567890@s.whatsapp.net",
    "timestamp": 1702129024
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| `message_id` | string | Unique message identifier |
| `chat` | string | Chat JID where the media was played |
| `sender` | string | JID of the recipient who played the media |
| `timestamp` | integer | Unix timestamp of the played receipt |

---

### `message.deleted`

Triggered when a message is deleted (revoked) by the sender.

```json
{
  "event_type": "message.deleted",
  "device_id": "abc123def456-ghi789",
  "timestamp": "2024-12-09T13:37:04.123456Z",
  "data": {
    "message_id": "3EB0ABC123DEF456789",
    "from": "6281234567890@s.whatsapp.net",
    "chat": "6281234567890@s.whatsapp.net",
    "timestamp": 1702129024,
    "deleted_by": "6281234567890@s.whatsapp.net",
    "is_from_me": false
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| `message_id` | string | ID of the deleted message |
| `from` | string | Original sender's JID |
| `chat` | string | Chat JID where the message was deleted |
| `timestamp` | integer | Unix timestamp of the deletion |
| `deleted_by` | string | JID of who deleted the message |
| `is_from_me` | boolean | `true` if deleted by the connected device |

---

## Connection Events

### `connection.connected`

Triggered when the device successfully connects to WhatsApp.

```json
{
  "event_type": "connection.connected",
  "device_id": "abc123def456-ghi789",
  "timestamp": "2024-12-09T13:37:04.123456Z",
  "data": {
    "jid": "6281234567890",
    "phone_number": "6281234567890",
    "push_name": "John Doe",
    "platform": "smba",
    "business_name": "My Business",
    "is_logged_in": true,
    "is_connected": true
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| `jid` | string | Phone number (without @s.whatsapp.net suffix) |
| `phone_number` | string | Same as `jid` |
| `push_name` | string | Display name set by the user |
| `platform` | string | Platform identifier (e.g., "smba" for business) |
| `business_name` | string | Business name (if business account) |
| `is_logged_in` | boolean | Login status |
| `is_connected` | boolean | Connection status |

---

### `connection.disconnected`

Triggered when the device disconnects from WhatsApp.

```json
{
  "event_type": "connection.disconnected",
  "device_id": "abc123def456-ghi789",
  "timestamp": "2024-12-09T13:37:04.123456Z",
  "data": {
    "jid": "6281234567890"
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| `jid` | string | Phone number of the disconnected device |

---

### `connection.logged_out`

Triggered when the device is logged out (session ended by WhatsApp or user).

```json
{
  "event_type": "connection.logged_out",
  "device_id": "abc123def456-ghi789",
  "timestamp": "2024-12-09T13:37:04.123456Z",
  "data": {
    "jid": "6281234567890"
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| `jid` | string | Phone number of the logged out device |

**Note:** After this event, you'll need to re-scan the QR code to reconnect.

---

### `connection.reconnecting`

Triggered when the device is attempting to reconnect after a connection loss.

```json
{
  "event_type": "connection.reconnecting",
  "device_id": "abc123def456-ghi789",
  "timestamp": "2024-12-09T13:37:04.123456Z",
  "data": {
    "jid": "6281234567890"
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| `jid` | string | Phone number of the reconnecting device |

---

### `connection.keepalive_timeout`

Triggered when the connection keep-alive mechanism times out.

```json
{
  "event_type": "connection.keepalive_timeout",
  "device_id": "abc123def456-ghi789",
  "timestamp": "2024-12-09T13:37:04.123456Z",
  "data": {
    "jid": "6281234567890",
    "error_count": 3,
    "last_success": "2024-12-09T13:35:04.123456Z"
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| `jid` | string | Phone number of the affected device |
| `error_count` | integer | Number of consecutive keepalive failures |
| `last_success` | string | ISO 8601 timestamp of last successful keepalive |

---

### `connection.temporary_ban`

Triggered when the account receives a temporary ban from WhatsApp.

```json
{
  "event_type": "connection.temporary_ban",
  "device_id": "abc123def456-ghi789",
  "timestamp": "2024-12-09T13:37:04.123456Z",
  "data": {
    "jid": "6281234567890",
    "reason": "spam",
    "expires": "2024-12-10T13:37:04.123456Z"
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| `jid` | string | Phone number of the banned device |
| `reason` | string | Ban reason code |
| `expires` | string | ISO 8601 timestamp when the ban expires |

**Warning:** Temporary bans indicate policy violations. Review your usage patterns.

---

## Call Events

### `call.offer`

Triggered when an incoming call is received.

```json
{
  "event_type": "call.offer",
  "device_id": "abc123def456-ghi789",
  "timestamp": "2024-12-09T13:37:04.123456Z",
  "data": {
    "jid": "6281234567890",
    "call_id": "CALL123ABC456DEF",
    "from": "6289876543210@s.whatsapp.net",
    "timestamp": 1702129024
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| `jid` | string | Phone number of the receiving device |
| `call_id` | string | Unique call identifier |
| `from` | string | Caller's JID |
| `timestamp` | integer | Unix timestamp when call was initiated |

---

### `call.accept`

Triggered when a call is accepted.

```json
{
  "event_type": "call.accept",
  "device_id": "abc123def456-ghi789",
  "timestamp": "2024-12-09T13:37:04.123456Z",
  "data": {
    "jid": "6281234567890",
    "call_id": "CALL123ABC456DEF",
    "from": "6289876543210@s.whatsapp.net",
    "timestamp": 1702129024
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| `jid` | string | Phone number of the device |
| `call_id` | string | Unique call identifier |
| `from` | string | Other party's JID |
| `timestamp` | integer | Unix timestamp when call was accepted |

---

### `call.terminate`

Triggered when a call ends.

```json
{
  "event_type": "call.terminate",
  "device_id": "abc123def456-ghi789",
  "timestamp": "2024-12-09T13:37:04.123456Z",
  "data": {
    "jid": "6281234567890",
    "call_id": "CALL123ABC456DEF",
    "from": "6289876543210@s.whatsapp.net",
    "timestamp": 1702129024,
    "reason": "normal"
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| `jid` | string | Phone number of the device |
| `call_id` | string | Unique call identifier |
| `from` | string | Other party's JID |
| `timestamp` | integer | Unix timestamp when call ended |
| `reason` | string | Termination reason (e.g., "normal", "busy", "rejected") |

---

## Group Events

### `group.join`

Triggered when the device joins a group.

```json
{
  "event_type": "group.join",
  "device_id": "abc123def456-ghi789",
  "timestamp": "2024-12-09T13:37:04.123456Z",
  "data": {
    "jid": "6281234567890",
    "group_jid": "120363123456789012@g.us",
    "reason": "invite"
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| `jid` | string | Phone number of the device that joined |
| `group_jid` | string | Group JID |
| `reason` | string | Join reason (e.g., "invite", "link", "add") |

---

### `group.info_update`

Triggered when group information is updated (name, description, photo, settings).

```json
{
  "event_type": "group.info_update",
  "device_id": "abc123def456-ghi789",
  "timestamp": "2024-12-09T13:37:04.123456Z",
  "data": {
    "jid": "6281234567890",
    "group_jid": "120363123456789012@g.us",
    "sender": "6289876543210@s.whatsapp.net",
    "timestamp": 1702129024
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| `jid` | string | Phone number of the device |
| `group_jid` | string | Group JID |
| `sender` | string | JID of the user who made the change |
| `timestamp` | integer | Unix timestamp of the change |

---

## Contact Events

### `contact.update`

Triggered when a contact's information changes.

```json
{
  "event_type": "contact.update",
  "device_id": "abc123def456-ghi789",
  "timestamp": "2024-12-09T13:37:04.123456Z",
  "data": {
    "jid": "6281234567890",
    "contact_jid": "6289876543210@s.whatsapp.net",
    "action": "update"
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| `jid` | string | Phone number of the device |
| `contact_jid` | string | JID of the updated contact |
| `action` | string | Action type (e.g., "update", "add", "remove") |

---

### `blocklist.change`

Triggered when the blocklist is modified.

```json
{
  "event_type": "blocklist.change",
  "device_id": "abc123def456-ghi789",
  "timestamp": "2024-12-09T13:37:04.123456Z",
  "data": {
    "jid": "6281234567890",
    "action": "block"
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| `jid` | string | Phone number of the device |
| `action` | string | Action type ("block" or "unblock") |

---

## App State Events

### `appstate.sync_complete`

Triggered when app state synchronization completes.

```json
{
  "event_type": "appstate.sync_complete",
  "device_id": "abc123def456-ghi789",
  "timestamp": "2024-12-09T13:37:04.123456Z",
  "data": {
    "jid": "6281234567890",
    "name": "regular_high"
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| `jid` | string | Phone number of the device |
| `name` | string | App state patch name that completed syncing |

---

### `appstate.patch_received`

Triggered when an app state patch is received.

```json
{
  "event_type": "appstate.patch_received",
  "device_id": "abc123def456-ghi789",
  "timestamp": "2024-12-09T13:37:04.123456Z",
  "data": {
    "jid": "6281234567890",
    "index": ["contact", "6289876543210@s.whatsapp.net"]
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| `jid` | string | Phone number of the device |
| `index` | array | Index path of the received patch |

**Note:** App state events are disabled by default. Enable with `WHATSAPP_APPSTATE_WEBHOOK_ENABLED=true`.

---

## History Sync Events

### `history.sync`

Triggered during message history synchronization.

```json
{
  "event_type": "history.sync",
  "device_id": "abc123def456-ghi789",
  "timestamp": "2024-12-09T13:37:04.123456Z",
  "data": {
    "jid": "6281234567890",
    "sync_type": "RECENT",
    "progress": 75,
    "conversation_count": 42
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| `jid` | string | Phone number of the device |
| `sync_type` | string | Type of sync (e.g., "RECENT", "FULL", "ON_DEMAND") |
| `progress` | integer | Sync progress percentage (0-100) |
| `conversation_count` | integer | Number of conversations synced |

---

### `history.sync_complete`

Triggered when a history sync reports completion (progress >= 100).

```json
{
  "event_type": "history.sync_complete",
  "device_id": "abc123def456-ghi789",
  "timestamp": "2024-12-09T13:37:04.123456Z",
  "data": {
    "jid": "6281234567890",
    "sync_type": "RECENT"
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| `jid` | string | Phone number of the device |
| `sync_type` | string | Type of sync (e.g., "RECENT", "FULL", "ON_DEMAND") |

---

## Newsletter Events

### `newsletter.join`

Triggered when subscribing to a newsletter/channel.

```json
{
  "event_type": "newsletter.join",
  "device_id": "abc123def456-ghi789",
  "timestamp": "2024-12-09T13:37:04.123456Z",
  "data": {
    "jid": "6281234567890",
    "newsletter_jid": "120363123456789012@newsletter"
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| `jid` | string | Phone number of the device |
| `newsletter_jid` | string | JID of the newsletter joined |

---

### `newsletter.leave`

Triggered when unsubscribing from a newsletter/channel.

```json
{
  "event_type": "newsletter.leave",
  "device_id": "abc123def456-ghi789",
  "timestamp": "2024-12-09T13:37:04.123456Z",
  "data": {
    "jid": "6281234567890",
    "newsletter_jid": "120363123456789012@newsletter",
    "role": "subscriber"
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| `jid` | string | Phone number of the device |
| `newsletter_jid` | string | JID of the newsletter left |
| `role` | string | Role before leaving (e.g., "subscriber", "admin") |

---

### `newsletter.update`

Triggered when a newsletter is updated (muted, settings changed, etc.).

```json
{
  "event_type": "newsletter.update",
  "device_id": "abc123def456-ghi789",
  "timestamp": "2024-12-09T13:37:04.123456Z",
  "data": {
    "jid": "6281234567890",
    "newsletter_jid": "120363123456789012@newsletter",
    "mute": true
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| `jid` | string | Phone number of the device |
| `newsletter_jid` | string | JID of the newsletter |
| `mute` | boolean | Mute status |

---

### `newsletter.message_received`

Triggered when a newsletter message is received.

```json
{
  "event_type": "newsletter.message_received",
  "device_id": "abc123def456-ghi789",
  "timestamp": "2024-12-09T13:37:04.123456Z",
  "data": {
    "jid": "6281234567890",
    "message_id": "3EB0ABC123DEF456789",
    "chat": "120363123456789012@newsletter",
    "timestamp": 1702129024,
    "is_edit": false
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| `jid` | string | Phone number of the device |
| `message_id` | string | Unique message identifier |
| `chat` | string | Newsletter JID |
| `timestamp` | integer | Unix timestamp of the message |
| `is_edit` | boolean | Whether this is an edit update |

---

### `newsletter.mute_change`

Triggered when a newsletter mute state changes.

```json
{
  "event_type": "newsletter.mute_change",
  "device_id": "abc123def456-ghi789",
  "timestamp": "2024-12-09T13:37:04.123456Z",
  "data": {
    "jid": "6281234567890",
    "newsletter_jid": "120363123456789012@newsletter",
    "mute": true
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| `jid` | string | Phone number of the device |
| `newsletter_jid` | string | Newsletter JID |
| `mute` | boolean | Mute state |

---

### `newsletter.live_update`

Triggered when a newsletter live update is received.

```json
{
  "event_type": "newsletter.live_update",
  "device_id": "abc123def456-ghi789",
  "timestamp": "2024-12-09T13:37:04.123456Z",
  "data": {
    "jid": "6281234567890",
    "newsletter_jid": "120363123456789012@newsletter",
    "time": "2024-12-09T13:37:04Z",
    "message_count": 1
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| `jid` | string | Phone number of the device |
| `newsletter_jid` | string | Newsletter JID |
| `time` | string | Timestamp of the update |
| `message_count` | integer | Number of messages included |

---

## Additional Events (Whatsmeow)

These events are emitted directly from whatsmeow or derived from message content. All payloads follow the standard envelope and use the `data` fields below.

### Connection (Additional)

| Event | Data fields |
|-------|-------------|
| `connection.qr` | `jid`, `codes` |
| `connection.qr_scanned_without_multidevice` | `jid` |
| `connection.pair_success` | `jid`, `pair_jid`, `pair_lid`, `business_name`, `platform` |
| `connection.pair_error` | `jid`, `pair_jid`, `pair_lid`, `business_name`, `platform`, `error` |
| `connection.client_outdated` | `jid` |
| `connection.cat_refresh_error` | `jid`, `error` |
| `connection.connect_failure` | `jid`, `reason`, `message` |
| `connection.stream_error` | `jid`, `code` |
| `connection.stream_replaced` | `jid` |
| `connection.manual_login_reconnect` | `jid` |
| `connection.keepalive_restored` | `jid` |

### Calls (Additional)

| Event | Data fields |
|-------|-------------|
| `call.pre_accept` | `jid`, `call_id`, `from`, `timestamp` |
| `call.offer_notice` | `jid`, `call_id`, `from`, `timestamp`, `media`, `type` |
| `call.transport` | `jid`, `call_id`, `from`, `timestamp` |
| `call.relay_latency` | `jid`, `call_id`, `from`, `timestamp` |
| `call.reject` | `jid`, `call_id`, `from`, `timestamp` |
| `call.unknown` | `jid` |

### Groups (Additional)

| Event | Data fields |
|-------|-------------|
| `group.participant_update` | `jid`, `group_jid`, `join`, `leave`, `promote`, `demote` |
| `group.leave` | `jid`, `group_jid`, `reason` |

### Messages, Polls, Media, Status

| Event | Data fields |
|-------|-------------|
| `message.undecryptable` | `jid`, `message_id`, `from`, `chat`, `timestamp`, `is_from_me`, `is_unavailable`, `unavailable_type`, `fail_mode` |
| `message.fb_received` | `jid`, `message_id`, `from`, `chat`, `timestamp`, `retry_count` |
| `poll.created` | `jid`, `message_id`, `chat`, `poll_name`, `options_count`, `selectable_options` |
| `poll.vote` | `jid`, `message_id`, `chat`, `timestamp` |
| `poll.update` | `jid`, `message_id`, `chat`, `timestamp` |
| `poll.vote_decrypted` | `jid`, `message_id`, `chat`, `selected_options` |
| `media.received` | `jid`, `message_id`, `chat`, `media_type` |
| `media.retry` | `jid`, `message_id`, `chat`, `sender`, `timestamp`, `from_me`, `error` |
| `status.posted` | `jid`, `message_id`, `from`, `timestamp`, `is_from_me` |
| `status.viewed` | `jid`, `message_id`, `chat`, `sender`, `timestamp` |
| `status.deleted` | `jid`, `message_id`, `from`, `timestamp`, `is_from_me` |
| `status.mute` | `jid`, `target`, `timestamp`, `from_full`, `action_raw` |
| `status.comment` | Reserved for future use |
| `media.downloaded` | Reserved for API-triggered downloads |

### Presence and Profile

| Event | Data fields |
|-------|-------------|
| `chat.presence` | `jid`, `chat`, `sender`, `is_from_me`, `state`, `media` |
| `presence.update` | `jid`, `from`, `unavailable`, `last_seen` |
| `identity.change` | `jid`, `target`, `timestamp`, `implicit` |
| `picture.update` | `jid`, `target`, `author`, `timestamp`, `removed`, `picture_id` |
| `user.about` | `jid`, `target`, `status`, `timestamp` |
| `privacy.settings` | `jid`, `settings`, `group_add_changed`, `last_seen_changed`, `status_changed`, `profile_changed`, `read_receipts_changed`, `online_changed`, `call_add_changed` |
| `pushname.update` | `jid`, `target`, `target_alt`, `old`, `new` |
| `pushname.setting` | `jid`, `timestamp`, `from_full`, `push_name`, `action_raw` |
| `business.name_update` | `jid`, `target`, `old`, `new` |

### Chat and Label State

| Event | Data fields |
|-------|-------------|
| `chat.mute` | `jid`, `chat`, `timestamp`, `from_full`, `action_raw` |
| `chat.archive` | `jid`, `chat`, `timestamp`, `from_full`, `action_raw` |
| `chat.pin` | `jid`, `chat`, `timestamp`, `from_full`, `action_raw` |
| `chat.star` | `jid`, `chat`, `sender`, `message_id`, `timestamp`, `is_from_me`, `from_full`, `action_raw` |
| `chat.delete_for_me` | `jid`, `chat`, `sender`, `message_id`, `timestamp`, `is_from_me`, `from_full`, `action_raw` |
| `chat.delete` | `jid`, `chat`, `timestamp`, `delete_media`, `from_full`, `action_raw` |
| `chat.clear` | `jid`, `chat`, `timestamp`, `delete_media`, `from_full`, `action_raw` |
| `chat.mark_read` | `jid`, `chat`, `timestamp`, `from_full`, `action_raw` |
| `label.edit` | `jid`, `label_id`, `timestamp`, `from_full`, `action_raw` |
| `label.chat` | `jid`, `chat`, `label_id`, `timestamp`, `from_full`, `action_raw` |
| `label.message` | `jid`, `chat`, `label_id`, `message_id`, `timestamp`, `from_full`, `action_raw` |
| `settings.unarchive_chats` | `jid`, `timestamp`, `from_full`, `action_raw` |

### App State and Offline Sync

| Event | Data fields |
|-------|-------------|
| `appstate.sync_error` | `jid`, `name`, `full_sync`, `error` |
| `offline.sync_preview` | `jid`, `total`, `app_data_changes`, `messages`, `notifications`, `receipts` |
| `offline.sync_completed` | `jid`, `count` |

---

## Webhook Configuration

### Creating a Webhook

```bash
curl -X POST https://your-api.com/webhooks \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://your-server.com/webhook",
    "secret": "your-webhook-secret",
    "events": ["message.received", "connection.connected"]
  }'
```

### Event Filtering

| Configuration | Behavior |
|--------------|----------|
| `"events": []` or omitted | Receives **ALL** event types |
| `"events": ["message.received"]` | Receives **only** `message.received` events |
| `"events": ["message.*"]` | Not supported - must specify exact event names |

### Multiple Webhooks

Each device can have up to 5 webhooks (configurable via `WEBHOOK_MAX_PER_DEVICE`).

---

## Security

### Signature Verification

All webhook requests include HMAC-SHA256 signatures for verification:

| Header | Description |
|--------|-------------|
| `X-Webhook-Signature` | `sha256=<hex_signature>` |
| `X-Hub-Signature-256` | Same as above (GitHub-compatible) |
| `X-Webhook-Event` | Event type (e.g., `message.received`) |

**Verification Example (Node.js):**

```javascript
const crypto = require('crypto');

function verifyWebhook(payload, signature, secret) {
  const expectedSignature = 'sha256=' + 
    crypto.createHmac('sha256', secret)
      .update(payload)
      .digest('hex');
  
  return crypto.timingSafeEqual(
    Buffer.from(signature),
    Buffer.from(expectedSignature)
  );
}

// In your webhook handler:
app.post('/webhook', (req, res) => {
  const signature = req.headers['x-webhook-signature'];
  const payload = JSON.stringify(req.body);
  
  if (!verifyWebhook(payload, signature, 'your-webhook-secret')) {
    return res.status(401).send('Invalid signature');
  }
  
  // Process the webhook...
  console.log('Event:', req.body.event_type);
  res.status(200).send('OK');
});
```

**Verification Example (Python):**

```python
import hmac
import hashlib

def verify_webhook(payload: bytes, signature: str, secret: str) -> bool:
    expected = 'sha256=' + hmac.new(
        secret.encode(),
        payload,
        hashlib.sha256
    ).hexdigest()
    return hmac.compare_digest(signature, expected)
```

**Verification Example (Go):**

```go
import (
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
)

func verifyWebhook(payload []byte, signature, secret string) bool {
    mac := hmac.New(sha256.New, []byte(secret))
    mac.Write(payload)
    expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))
    return hmac.Equal([]byte(signature), []byte(expected))
}
```

### URL Requirements

- **HTTPS only** - HTTP URLs are rejected
- **No private IPs** - localhost, 127.0.0.1, 192.168.x.x, 10.x.x.x, 172.x.x.x are blocked
- **Response timeout** - 10 seconds

### Retry Policy

| Attempt | Delay |
|---------|-------|
| 1 | Immediate |
| 2 | 4 seconds |
| 3 | 6 seconds |

After 3 failed attempts, the delivery is marked as failed and logged.

---

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `WEBHOOKS_ENABLED` | `true` | Enable/disable webhook system |
| `WEBHOOK_WORKERS` | `4` | Number of concurrent delivery workers |
| `WEBHOOK_RETRY_LIMIT` | `3` | Maximum delivery attempts |
| `WEBHOOK_MAX_PER_DEVICE` | `5` | Maximum webhooks per device |
| `WHATSAPP_APPSTATE_WEBHOOK_ENABLED` | `false` | Enable app state events |

---

## Quick Reference

### All Event Types

```
message.received
message.delivered
message.read
message.played
message.deleted
message.undecryptable
message.fb_received
connection.connected
connection.disconnected
connection.logged_out
connection.reconnecting
connection.keepalive_timeout
connection.keepalive_restored
connection.temporary_ban
connection.client_outdated
connection.cat_refresh_error
connection.connect_failure
connection.stream_error
connection.stream_replaced
connection.manual_login_reconnect
connection.qr
connection.qr_scanned_without_multidevice
connection.pair_success
connection.pair_error
call.offer
call.accept
call.pre_accept
call.offer_notice
call.transport
call.relay_latency
call.terminate
call.reject
call.unknown
group.join
group.leave
group.participant_update
group.info_update
newsletter.join
newsletter.leave
newsletter.message_received
newsletter.update
newsletter.mute_change
newsletter.live_update
contact.update
chat.presence
presence.update
identity.change
picture.update
user.about
privacy.settings
pushname.update
pushname.setting
business.name_update
chat.mute
chat.archive
chat.pin
chat.star
chat.delete_for_me
chat.delete
chat.clear
chat.mark_read
label.edit
label.chat
label.message
settings.unarchive_chats
status.mute
blocklist.change
appstate.sync_complete
appstate.sync_error
appstate.patch_received
history.sync
history.sync_complete
offline.sync_preview
offline.sync_completed
poll.created
poll.vote
poll.update
poll.vote_decrypted
status.posted
status.viewed
status.deleted
status.comment
media.received
media.downloaded
media.retry
```

---

*Last updated: February 2026*

