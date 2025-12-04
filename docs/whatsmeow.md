# WhatsMeow Client Functions (130+ Total)

This document lists all methods available in `go.mau.fi/whatsmeow@v0.0.0-20251127132918-b9ac3d51d746` Client type, organized by functionality.

> Sessions in the REST API are now keyed by `{jid, device_id}`; every helper function resolves the correct whatsmeow client using the JWT payload.

## 1. Connection & Authentication (9 functions)

### Core Connection Management
- ✅ `Connect()` - **Input:** None **Output:** `error` - Implemented as `WhatsAppLogin()`, `WhatsAppReconnect()`
- ✅ `ConnectContext(ctx)` - **Input:** `context.Context` **Output:** `error` - Connect with context (NEW)
- ✅ `Disconnect()` - **Input:** None **Output:** None - Implemented as `WhatsAppReconnect()`, `WhatsAppLogout()`
- ✅ `IsConnected()` - **Input:** None **Output:** `bool` - Implemented as `WhatsAppIsClientOK()`
- ✅ `IsLoggedIn()` - **Input:** None **Output:** `bool` - Implemented as `WhatsAppIsClientOK()`
- ✅ `WaitForConnection(timeout)` - **Input:** `time.Duration` **Output:** `bool` - Waits for connection establishment with timeout

### Authentication
- ✅ `Logout(ctx)` - **Input:** `context.Context` **Output:** `error` - Implemented as `WhatsAppLogout()`
- ✅ `PairPhone(ctx, phone, showPushNotification, ...)` - **Input:** `context.Context, string, bool, ...` **Output:** `string, error` - Implemented as `WhatsAppLoginPair()`
- ✅ `GetQRChannel(ctx)` - **Input:** `context.Context` **Output:** `<-chan QRChannelItem, error` - Implemented as `WhatsAppLogin()`, `WhatsAppGenerateQR()`

## 2. Message Building & Sending (16 functions)

### Message Sending
- ✅ `SendMessage(ctx, to, message, ...)` - **Input:** `context.Context, types.JID, *waE2E.Message, ...SendRequestExtra` **Output:** `SendResponse, error` - Implemented as various `WhatsAppSend*()` functions
- ✅ `SendPresence(ctx, state)` - **Input:** `context.Context, types.Presence` **Output:** `error` - Implemented as `WhatsAppPresence()`
- ✅ `SendChatPresence(ctx, jid, state, media)` - **Input:** `context.Context, types.JID, types.ChatPresence, types.ChatPresenceMedia` **Output:** `error` - Implemented as `WhatsAppComposeStatus()`
- ❌ `SendFBMessage(ctx, to, message, metadata, ...)` - **Input:** `context.Context, types.JID, armadillo.RealMessageApplicationSub, *waMsgApplication.MessageApplication_Metadata, ...SendRequestExtra` **Output:** `SendResponse, error` - Sends Facebook Messenger format messages
- ❌ `SendAppState(ctx, patch)` - **Input:** `context.Context, appstate.PatchInfo` **Output:** `error` - Sends application state synchronization patches

### Message Building
- ✅ `BuildEdit(chat, id, newContent)` - **Input:** `types.JID, types.MessageID, *waE2E.Message` **Output:** `*waE2E.Message` - Implemented as `WhatsAppMessageEdit()`
- ✅ `BuildPollCreation(name, optionNames, selectableOptionCount)` - **Input:** `string, []string, int` **Output:** `*waE2E.Message` - Implemented as `WhatsAppSendPoll()`
- ❌ `BuildPollVote(ctx, pollInfo, optionNames)` - **Input:** `context.Context, *types.MessageInfo, []string` **Output:** `*waE2E.Message, error` - Creates poll vote message for responding to polls
- ✅ `BuildReaction(chat, sender, id, reaction)` - **Input:** `types.JID, types.JID, types.MessageID, string` **Output:** `*waE2E.Message` - Implemented as `WhatsAppMessageReact()`
- ✅ `BuildRevoke(chat, sender, id)` - **Input:** `types.JID, types.JID, types.MessageID` **Output:** `*waE2E.Message` - Implemented as `WhatsAppMessageDelete()`
- ❌ `BuildHistorySyncRequest(lastKnownMessageInfo, count)` - **Input:** `*types.MessageInfo, int` **Output:** `*waE2E.Message` - Creates request to sync message history
- ❌ `BuildMessageKey(chat, sender, id)` - **Input:** `types.JID, types.JID, types.MessageID` **Output:** `*waCommon.MessageKey` - Creates message key structure for message identification
- ❌ `BuildUnavailableMessageRequest(chat, sender, id)` - **Input:** `types.JID, types.JID, string` **Output:** `*waE2E.Message` - Creates request for unavailable message content

### Message Management
- ✅ `GenerateMessageID()` - **Input:** None **Output:** `types.MessageID` - Implemented as part of `SendMessage` calls
- ⚠️ `RevokeMessage(ctx, chat, id)` - **Input:** `context.Context, types.JID, types.MessageID` **Output:** `SendResponse, error` - **DEPRECATED** - Use `BuildRevoke` + `SendMessage` instead
- ❌ `SendMediaRetryReceipt(ctx, message, mediaKey)` - **Input:** `context.Context, *types.MessageInfo, []byte` **Output:** `error` - Sends retry receipt for failed media uploads

## 3. Media Operations (13 functions)

### Media Upload
- ✅ `Upload(ctx, plaintext, appInfo)` - **Input:** `context.Context, []byte, MediaType` **Output:** `UploadResponse, error` - Implemented in various `WhatsAppSend*()` functions
- ❌ `UploadReader(ctx, plaintext, tempFile, ...)` - **Input:** `context.Context, io.Reader, io.ReadWriteSeeker, ...` **Output:** `UploadResponse, error` - Uploads media from stream reader
- ❌ `UploadNewsletter(ctx, data, appInfo)` - **Input:** `context.Context, []byte, MediaType` **Output:** `UploadResponse, error` - Uploads media for newsletter/channel messages
- ❌ `UploadNewsletterReader(ctx, data, appInfo)` - **Input:** `context.Context, io.ReadSeeker, MediaType` **Output:** `UploadResponse, error` - Uploads newsletter media from stream

### Media Download
- ✅ `Download(ctx, message)` - **Input:** `context.Context, DownloadableMessage` **Output:** `[]byte, error` - Implemented as part of media handling
- ⚠️ `DownloadAny(ctx, message)` - **Input:** `context.Context, *waE2E.Message` **Output:** `[]byte, error` - **DEPRECATED** - Use `Download` instead
- ❌ `DownloadFB(ctx, transport, ...)` - **Input:** `context.Context, *waMediaTransport.WAMediaTransport_Integral, ...` **Output:** `[]byte, error` - Downloads Facebook Messenger media
- ❌ `DownloadFBToFile(ctx, transport, ...)` - **Input:** `context.Context, *waMediaTransport.WAMediaTransport_Integral, ...` **Output:** `error` - Downloads FB media directly to file
- ❌ `DownloadHistorySync(ctx, notification, ...)` - **Input:** `context.Context, *waE2E.HistorySyncNotification, ...` **Output:** `*waHistorySync.HistorySync, error` - Downloads history synchronization data
- ❌ `DownloadMediaWithPath(ctx, directPath, ...)` - **Input:** `context.Context, string, []byte, []byte, []byte, ...` **Output:** `[]byte, error` - Downloads media using direct path
- ❌ `DownloadMediaWithPathToFile(ctx, directPath, ...)` - **Input:** `context.Context, string, []byte, []byte, []byte, ...` **Output:** `error` - Downloads media to file using direct path
- ❌ `DownloadThumbnail(ctx, message)` - **Input:** `context.Context, DownloadableThumbnail` **Output:** `[]byte, error` - Downloads message thumbnail image
- ❌ `DownloadToFile(ctx, message, file)` - **Input:** `context.Context, DownloadableMessage, File` **Output:** `error` - Downloads media directly to specified file

## 4. Group Management (21 functions)

### Group Information
- ❌ `GetGroupInfo(ctx, jid)` - **Input:** `context.Context, types.JID` **Output:** `*types.GroupInfo, error` - Gets detailed information about a group
- ❌ `GetGroupInfoFromInvite(ctx, jid, inviter, code, expiration)` - **Input:** `context.Context, types.JID, types.JID, string, int64` **Output:** `*types.GroupInfo, error` - Gets group info using invite details
- ❌ `GetGroupInfoFromLink(ctx, code)` - **Input:** `context.Context, string` **Output:** `*types.GroupInfo, error` - Gets group info from invite link code
- ❌ `GetGroupInviteLink(ctx, jid, reset)` - **Input:** `context.Context, types.JID, bool` **Output:** `string, error` - Gets or resets group invitation link
- ❌ `GetGroupRequestParticipants(ctx, jid)` - **Input:** `context.Context, types.JID` **Output:** `[]types.GroupParticipantRequest, error` - Gets list of pending join requests
- ✅ `GetJoinedGroups(ctx)` - **Input:** `context.Context` **Output:** `[]*types.GroupInfo, error` - Implemented as `WhatsAppGroupGet()`, `WhatsAppGroupGetWithMembers()`
- ❌ `GetLinkedGroupsParticipants(ctx, community)` - **Input:** `context.Context, types.JID` **Output:** `[]types.JID, error` - Gets participants of linked community groups
- ❌ `GetSubGroups(ctx, community)` - **Input:** `context.Context, types.JID` **Output:** `[]*types.GroupLinkTarget, error` - Gets subgroups within a community

### Group Creation & Membership
- ✅ `CreateGroup(ctx, req)` - **Input:** `context.Context, ReqCreateGroup` **Output:** `*types.GroupInfo, error` - Implemented as `WhatsAppGroupCreate()`
- ❌ `JoinGroupWithInvite(ctx, jid, inviter, code, expiration)` - **Input:** `context.Context, types.JID, types.JID, string, int64` **Output:** `error` - Joins group using invite details
- ✅ `JoinGroupWithLink(ctx, code)` - **Input:** `context.Context, string` **Output:** `types.JID, error` - Implemented as `WhatsAppGroupJoin()`
- ✅ `LeaveGroup(ctx, jid)` - **Input:** `context.Context, types.JID` **Output:** `error` - Implemented as `WhatsAppGroupLeave()`
- ❌ `LinkGroup(ctx, parent, child)` - **Input:** `context.Context, types.JID, types.JID` **Output:** `error` - Links a subgroup to a community
- ❌ `UnlinkGroup(ctx, parent, child)` - **Input:** `context.Context, types.JID, types.JID` **Output:** `error` - Unlinks a subgroup from a community

### Group Settings
- ❌ `SetGroupAnnounce(ctx, jid, announce)` - **Input:** `context.Context, types.JID, bool` **Output:** `error` - Toggles announcement mode (only admins can send messages)
- ❌ `SetGroupDescription(ctx, jid, description)` - **Input:** `context.Context, types.JID, string` **Output:** `error` - Updates group description
- ❌ `SetGroupJoinApprovalMode(ctx, jid, mode)` - **Input:** `context.Context, types.JID, bool` **Output:** `error` - Enables/disables admin approval for joins
- ❌ `SetGroupLocked(ctx, jid, locked)` - **Input:** `context.Context, types.JID, bool` **Output:** `error` - Locks/unlocks group settings
- ❌ `SetGroupMemberAddMode(ctx, jid, mode)` - **Input:** `context.Context, types.JID, types.GroupMemberAddMode` **Output:** `error` - Sets who can add members (all/admin only)
- ❌ `SetGroupName(ctx, jid, name)` - **Input:** `context.Context, types.JID, string` **Output:** `error` - Changes group name
- ❌ `SetGroupPhoto(ctx, jid, avatar)` - **Input:** `context.Context, types.JID, []byte` **Output:** `string, error` - Updates group profile picture
- ❌ `SetGroupTopic(ctx, jid, previousID, newID, topic)` - **Input:** `context.Context, types.JID, string, string, string` **Output:** `error` - Sets or updates group topic

### Group Participants
- ❌ `UpdateGroupParticipants(ctx, jid, participantChanges, action)` - **Input:** `context.Context, types.JID, []types.JID, ParticipantChange` **Output:** `[]types.GroupParticipant, error` - Adds/removes group participants
- ❌ `UpdateGroupRequestParticipants(ctx, jid, participantChanges, action)` - **Input:** `context.Context, types.JID, []types.JID, ParticipantRequestChange` **Output:** `[]types.GroupParticipant, error` - Approves/rejects join requests

## 5. Contact & Profile Operations (11 functions)

- ❌ `GetBusinessProfile(ctx, jid)` - **Input:** `context.Context, types.JID` **Output:** `*types.BusinessProfile, error` - Gets business account profile information
- ❌ `GetContactQRLink(ctx, revoke)` - **Input:** `context.Context, bool` **Output:** `string, error` - Gets contact QR code link for sharing
- ❌ `GetProfilePictureInfo(ctx, jid, params)` - **Input:** `context.Context, types.JID, *GetProfilePictureParams` **Output:** `*types.ProfilePictureInfo, error` - Gets profile picture URL and metadata
- ❌ `GetUserDevices(ctx, jids)` - **Input:** `context.Context, []types.JID` **Output:** `[]types.JID, error` - Gets list of user's linked devices
- ❌ `GetUserDevicesContext(ctx, jids)` - **Input:** `context.Context, []types.JID` **Output:** `[]types.JID, error` - Gets user devices with context (same as above)
- ❌ `GetUserInfo(ctx, jids)` - **Input:** `context.Context, []types.JID` **Output:** `map[types.JID]types.UserInfo, error` - Gets detailed user information and status
- ✅ `IsOnWhatsApp(ctx, phones)` - **Input:** `context.Context, []string` **Output:** `[]types.IsOnWhatsAppResponse, error` - Implemented as `WhatsAppCheckRegistered()`, `WhatsAppGetJID()`
- ❌ `SubscribePresence(ctx, jid)` - **Input:** `context.Context, types.JID` **Output:** `error` - Subscribes to real-time presence updates
- ❌ `SetStatusMessage(ctx, msg)` - **Input:** `context.Context, string` **Output:** `error` - Updates profile status/about message
- ❌ `ResolveBusinessMessageLink(ctx, code)` - **Input:** `context.Context, string` **Output:** `*types.BusinessMessageLinkTarget, error` - Resolves business message link to contact
- ❌ `ResolveContactQRLink(ctx, code)` - **Input:** `context.Context, string` **Output:** `*types.ContactQRLinkTarget, error` - Resolves contact QR code to user info

## 6. Privacy & Security (8 functions)

- ❌ `GetBlocklist(ctx)` - **Input:** `context.Context` **Output:** `*types.Blocklist, error` - Gets list of blocked contacts
- ❌ `GetPrivacySettings(ctx)` - **Input:** `context.Context` **Output:** `types.PrivacySettings` - Gets all privacy settings
- ❌ `GetStatusPrivacy(ctx)` - **Input:** `context.Context` **Output:** `[]types.StatusPrivacy, error` - Gets status visibility privacy settings
- ❌ `SetDefaultDisappearingTimer(ctx, timer)` - **Input:** `context.Context, time.Duration` **Output:** `error` - Sets default disappearing messages timer
- ❌ `SetDisappearingTimer(ctx, chat, timer, settingTS)` - **Input:** `context.Context, types.JID, time.Duration, time.Time` **Output:** `error` - Sets disappearing messages for specific chat
- ❌ `SetPrivacySetting(ctx, name, value)` - **Input:** `context.Context, types.PrivacySettingType, types.PrivacySetting` **Output:** `types.PrivacySettings, error` - Updates specific privacy setting
- ❌ `TryFetchPrivacySettings(ctx, ignoreCache)` - **Input:** `context.Context, bool` **Output:** `*types.PrivacySettings, error` - Fetches latest privacy settings from server
- ❌ `UpdateBlocklist(ctx, jid, action)` - **Input:** `context.Context, types.JID, events.BlocklistChangeAction` **Output:** `*types.Blocklist, error` - Blocks or unblocks a contact

## 7. Newsletter/Channel Operations (13 functions)

### Newsletter Management
- ❌ `CreateNewsletter(ctx, params)` - **Input:** `context.Context, CreateNewsletterParams` **Output:** `*types.NewsletterMetadata, error` - Creates a new WhatsApp channel/newsletter
- ❌ `FollowNewsletter(ctx, jid)` - **Input:** `context.Context, types.JID` **Output:** `error` - Follows a WhatsApp channel
- ❌ `GetNewsletterInfo(ctx, jid)` - **Input:** `context.Context, types.JID` **Output:** `*types.NewsletterMetadata, error` - Gets channel/newsletter information
- ❌ `GetNewsletterInfoWithInvite(ctx, key)` - **Input:** `context.Context, string` **Output:** `*types.NewsletterMetadata, error` - Gets newsletter info using invite key
- ❌ `GetNewsletterMessageUpdates(ctx, jid, params)` - **Input:** `context.Context, types.JID, *GetNewsletterUpdatesParams` **Output:** `[]*types.NewsletterMessage, error` - Gets recent message updates from channel
- ❌ `GetNewsletterMessages(ctx, jid, params)` - **Input:** `context.Context, types.JID, *GetNewsletterMessagesParams` **Output:** `[]*types.NewsletterMessage, error` - Gets messages from a newsletter
- ❌ `GetSubscribedNewsletters(ctx)` - **Input:** `context.Context` **Output:** `[]*types.NewsletterMetadata, error` - Gets list of followed newsletters
- ❌ `UnfollowNewsletter(ctx, jid)` - **Input:** `context.Context, types.JID` **Output:** `error` - Unfollows a WhatsApp channel

### Newsletter Interactions
- ❌ `NewsletterMarkViewed(ctx, jid, serverIDs)` - **Input:** `context.Context, types.JID, []types.MessageServerID` **Output:** `error` - Marks channel messages as viewed
- ❌ `NewsletterSendReaction(ctx, jid, serverID, reaction, ...)` - **Input:** `context.Context, types.JID, types.MessageServerID, string, ...` **Output:** `error` - Reacts to channel messages
- ❌ `NewsletterSubscribeLiveUpdates(ctx, jid)` - **Input:** `context.Context, types.JID` **Output:** `time.Duration, error` - Subscribes to live channel updates
- ❌ `NewsletterToggleMute(ctx, jid, mute)` - **Input:** `context.Context, types.JID, bool` **Output:** `error` - Mutes/unmutes a newsletter
- ❌ `AcceptTOSNotice(ctx, noticeID, stage)` - **Input:** `context.Context, string, string` **Output:** `error` - Accepts terms of service notices (required for newsletters)

## 8. Event Handlers & Advanced Features (12 functions)

### Event Management
- ✅ `AddEventHandler(handler)` - **Input:** `EventHandler` **Output:** `uint32` - Implemented in `WhatsAppInitClient()` for connection events
- ❌ `AddEventHandlerWithSuccessStatus(handler)` - **Input:** `EventHandlerWithSuccessStatus` **Output:** `uint32` - Adds handler with success status tracking
- ❌ `RemoveEventHandler(id)` - **Input:** `uint32` **Output:** `bool` - Removes specific event handler
- ❌ `RemoveEventHandlers()` - **Input:** None **Output:** None - Removes all event handlers

### Message Processing
- ❌ `ParseWebMessage(chatJID, webMsg)` - **Input:** `types.JID, *waWeb.WebMessageInfo` **Output:** `*events.Message, error` - Parses WhatsApp web message format
- ❌ `DecryptComment(ctx, comment)` - **Input:** `context.Context, *events.Message` **Output:** `*waE2E.Message, error` - Decrypts comment messages
- ❌ `DecryptPollVote(ctx, vote)` - **Input:** `context.Context, *events.Message` **Output:** `*waE2E.PollVoteMessage, error` - Decrypts poll vote messages
- ❌ `DecryptReaction(ctx, reaction)` - **Input:** `context.Context, *events.Message` **Output:** `*waE2E.ReactionMessage, error` - Decrypts reaction messages
- ❌ `EncryptComment(ctx, rootMsgInfo, comment)` - **Input:** `context.Context, *types.MessageInfo, *waE2E.Message` **Output:** `*waE2E.Message, error` - Encrypts comment messages
- ❌ `EncryptPollVote(ctx, pollInfo, vote)` - **Input:** `context.Context, *types.MessageInfo, *waE2E.PollVoteMessage` **Output:** `*waE2E.PollUpdateMessage, error` - Encrypts poll vote messages
- ❌ `EncryptReaction(ctx, rootMsgInfo, ...)` - **Input:** `context.Context, *types.MessageInfo, ...` **Output:** `*waE2E.EncReactionMessage, error` - Encrypts reaction messages

### App State
- ❌ `FetchAppState(ctx, name, fullSync, onlyIfNotSynced)` - **Input:** `context.Context, appstate.WAPatchName, bool, bool` **Output:** `error` - Fetches application state patches

## 9. Push Notifications & Business Features (4 functions)

- ❌ `RegisterForPushNotifications(ctx, config)` - **Input:** `context.Context, PushConfig` **Output:** `error` - Registers device for push notifications
- ❌ `GetServerPushNotificationConfig(ctx)` - **Input:** `context.Context` **Output:** `*waBinary.Node, error` - Gets push notification configuration
- ❌ `ResolveBusinessMessageLink(ctx, code)` - **Input:** `context.Context, string` **Output:** `*types.BusinessMessageLinkTarget, error` - Resolves business message link to contact
- ❌ `ResolveContactQRLink(ctx, code)` - **Input:** `context.Context, string` **Output:** `*types.ContactQRLinkTarget, error` - Resolves contact QR code to user info

## 10. Bot Operations (2 functions)

- ❌ `GetBotListV2(ctx)` - **Input:** `context.Context` **Output:** `[]types.BotListInfo, error` - Gets list of available WhatsApp bots
- ❌ `GetBotProfiles(ctx, botInfo)` - **Input:** `context.Context, []types.BotListInfo` **Output:** `[]types.BotProfileInfo, error` - Gets detailed information about bots

## 11. Proxy & Connection Settings (12 functions)

- ❌ `SetProxy(proxy, ...)` - **Input:** `Proxy, ...SetProxyOptions` **Output:** None - Sets HTTP proxy for connections
- ✅ `SetProxyAddress(addr, ...)` - **Input:** `string, ...SetProxyOptions` **Output:** `error` - Implemented as `WhatsAppClientProxyURL` in `WhatsAppInitClient()`
- ❌ `SetSOCKSProxy(proxy, ...)` - **Input:** `proxy.Dialer, ...SetProxyOptions` **Output:** None - Sets SOCKS proxy for connections
- ❌ `SetWSDialer(dialer)` - **Input:** `*websocket.Dialer` **Output:** None - Sets custom WebSocket dialer (REMOVED - use SetProxy)
- ❌ `SetMediaHTTPClient(h)` - **Input:** `*http.Client` **Output:** None - Sets custom HTTP client for media operations
- ❌ `SetPreLoginHTTPClient(h)` - **Input:** `*http.Client` **Output:** None - Sets HTTP client for pre-login operations
- ❌ `SetWebsocketHTTPClient(h)` - **Input:** `*http.Client` **Output:** None - Sets HTTP client for websocket connections
- ❌ `SetPassive(ctx, passive)` - **Input:** `context.Context, bool` **Output:** `error` - Sets passive connection mode
- ❌ `SetForceActiveDeliveryReceipts(active)` - **Input:** `bool` **Output:** None - Forces active delivery receipts
- ❌ `StoreLIDPNMapping(ctx, first, second)` - **Input:** `context.Context, types.JID, types.JID` **Output:** None - Stores LID to PN identifier mapping

## 12. Miscellaneous (5 functions)

- ❌ `MarkNotDirty(ctx, cleanType, ts)` - **Input:** `context.Context, string, time.Time` **Output:** `error` - Marks app state as not dirty
- ⚠️ `DangerousInternals()` - **Input:** None **Output:** `*DangerousInternalClient` - **DEPRECATED** - Accesses dangerous internal functions
- ❌ `RejectCall(ctx, callFrom, callID)` - **Input:** `context.Context, types.JID, string` **Output:** `error` - Rejects incoming voice/video calls
- ❌ `MarkRead(ctx, ids, timestamp, chat, sender, ...)` - **Input:** `context.Context, []types.MessageID, time.Time, types.JID, types.JID, ...` **Output:** `error` - Marks messages as read

---

## Summary

- **Total Functions**: 130+ (Public Client methods)
- **Implemented Functions**: ~28 (22%)
- **Not Implemented**: ~102 (78%)
- **Most Populated Category**: Group Management (21 functions)
- **Least Populated Category**: Bot Operations (2 functions)
- **Most Implemented Category**: Connection & Authentication (9/9 ✅)
- **Least Implemented Category**: Privacy & Security, Newsletter/Channel Operations (0/8-13 ❌)

### Key Changes in v0.0.0-20251127
- Most functions now require `context.Context` as first parameter
- `RevokeMessage` is **deprecated** - use `BuildRevoke()` + `SendMessage()` instead
- `DownloadAny` is **deprecated** - use `Download()` instead
- `DangerousInternals()` is **deprecated**
- Added `ConnectContext()` for context-aware connection
- Added `SetMediaHTTPClient()`, `SetPreLoginHTTPClient()`, `SetWebsocketHTTPClient()`
- Added `MarkNotDirty()` for app state management

**Priority Implementation Areas:**
1. **High Priority**: Complete group management functions (participant management, group settings)
2. **Medium Priority**: Privacy and security settings (blocking, disappearing messages)
3. **Low Priority**: Newsletter features, advanced message processing, push notifications

**Legend**: ✅ = Implemented, ❌ = Not Implemented, ⚠️ = Deprecated

This comprehensive list represents all available functionality in whatsmeow v0.0.0-20251127132918-b9ac3d51d746 for interacting with WhatsApp's API.
