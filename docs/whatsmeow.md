# WhatsMeow Client Functions (120 Total)

This document lists all 120 methods available in the latest `go.mau.fi/whatsmeow` Client type, organized by functionality.

> Sessions in the REST API are now keyed by `{jid, device_id}`; every helper function resolves the correct whatsmeow client using the JWT payload.

## 1. Connection & Authentication (8 functions)

### Core Connection Management
- ✅ `Connect()` - **Input:** None **Output:** `error` - Implemented as `WhatsAppLogin()`, `WhatsAppReconnect()`
- ✅ `Disconnect()` - **Input:** None **Output:** None - Implemented as `WhatsAppReconnect()`, `WhatsAppLogout()`
- ✅ `IsConnected()` - **Input:** None **Output:** `bool` - Implemented as `WhatsAppIsClientOK()`
- ✅ `IsLoggedIn()` - **Input:** None **Output:** `bool` - Implemented as `WhatsAppIsClientOK()`
- ❌ `WaitForConnection(timeout)` - **Input:** `time.Duration` **Output:** `bool` - Waits for connection establishment with timeout

### Authentication
- ✅ `Logout(ctx)` - **Input:** `context.Context` **Output:** `error` - Implemented as `WhatsAppLogout()`
- ✅ `PairPhone(ctx, phone, showPushNotification, ...)` - **Input:** `context.Context, string, bool, ...` **Output:** `string, error` - Implemented as `WhatsAppLoginPair()`
- ✅ `GetQRChannel(ctx)` - **Input:** `context.Context` **Output:** `<-chan QRChannelItem, error` - Implemented as `WhatsAppLogin()`, `WhatsAppGenerateQR()`

## 2. Message Building & Sending (15 functions)

### Message Sending
- ✅ `SendMessage(ctx, to, message, ...)` - **Input:** `context.Context, types.JID, *waE2E.Message, ...` **Output:** `SendResponse, error` - Implemented as various `WhatsAppSend*()` functions
- ✅ `SendPresence(state)` - **Input:** `types.Presence` **Output:** `error` - Implemented as `WhatsAppPresence()`
- ✅ `SendChatPresence(jid, state, media)` - **Input:** `types.JID, types.ChatPresence, types.ChatPresenceMedia` **Output:** `error` - Implemented as `WhatsAppComposeStatus()`
- ❌ `SendFBMessage(ctx, to, message, ...)` - **Input:** `context.Context, types.JID, armadillo.RealMessageApplicationSub, ...` **Output:** `SendResponse, error` - Sends Facebook Messenger format messages
- ❌ `SendAppState(ctx, patch)` - **Input:** `context.Context, appstate.PatchInfo` **Output:** `error` - Sends application state synchronization patches

### Message Building
- ✅ `BuildEdit(chat, id, newContent)` - **Input:** `types.JID, types.MessageID, *waE2E.Message` **Output:** `*waE2E.Message` - Implemented as `WhatsAppMessageEdit()`
- ✅ `BuildPollCreation(question, options, selectableCount)` - **Input:** `string, []string, int` **Output:** `*waE2E.Message` - Implemented as `WhatsAppSendPoll()`
- ❌ `BuildPollVote(ctx, pollInfo, options)` - **Input:** `context.Context, *types.MessageInfo, []string` **Output:** `*waE2E.Message, error` - Creates poll vote message for responding to polls
- ✅ `BuildReaction(chat, sender, id, reaction)` - **Input:** `types.JID, types.JID, types.MessageID, string` **Output:** `*waE2E.Message` - Implemented as `WhatsAppMessageReact()`
- ✅ `BuildRevoke(chat, sender, id)` - **Input:** `types.JID, types.JID, types.MessageID` **Output:** `*waE2E.Message` - Implemented as `WhatsAppMessageDelete()`
- ❌ `BuildHistorySyncRequest(lastKnownMessage, count)` - **Input:** `*types.MessageInfo, int` **Output:** `*waE2E.Message` - Creates request to sync message history
- ❌ `BuildMessageKey(chat, sender, id)` - **Input:** `types.JID, types.JID, types.MessageID` **Output:** `*waCommon.MessageKey` - Creates message key structure for message identification
- ❌ `BuildUnavailableMessageRequest(chat, sender, id)` - **Input:** `types.JID, types.JID, string` **Output:** `*waE2E.Message` - Creates request for unavailable message content

### Message Management
- ✅ `GenerateMessageID()` - **Input:** None **Output:** `types.MessageID` - Implemented as part of `SendMessage` calls
- ✅ `RevokeMessage(chat, id)` - **Input:** `types.JID, types.MessageID` **Output:** `SendResponse, error` - Implemented as `WhatsAppMessageDelete()`
- ❌ `SendMediaRetryReceipt(message, mediaKey)` - **Input:** `*types.MessageInfo, []byte` **Output:** `error` - Sends retry receipt for failed media uploads

## 3. Media Operations (12 functions)

### Media Upload
- ✅ `Upload(ctx, data, mediaType)` - **Input:** `context.Context, []byte, MediaType` **Output:** `UploadResponse, error` - Implemented in various `WhatsAppSend*()` functions
- ❌ `UploadReader(ctx, reader, tempFile, ...)` - **Input:** `context.Context, io.Reader, io.ReadWriteSeeker, ...` **Output:** `UploadResponse, error` - Uploads media from stream reader
- ❌ `UploadNewsletter(ctx, data, mediaType)` - **Input:** `context.Context, []byte, MediaType` **Output:** `UploadResponse, error` - Uploads media for newsletter/channel messages
- ❌ `UploadNewsletterReader(ctx, reader, mediaType)` - **Input:** `context.Context, io.ReadSeeker, MediaType` **Output:** `UploadResponse, error` - Uploads newsletter media from stream

### Media Download
- ✅ `Download(ctx, message)` - **Input:** `context.Context, DownloadableMessage` **Output:** `[]byte, error` - Implemented as part of media handling
- ❌ `DownloadAny(ctx, message)` - **Input:** `context.Context, *waE2E.Message` **Output:** `[]byte, error` - Downloads any type of media from message
- ❌ `DownloadFB(ctx, transport, ...)` - **Input:** `context.Context, *waMediaTransport.WAMediaTransport_Integral, ...` **Output:** `[]byte, error` - Downloads Facebook Messenger media
- ❌ `DownloadFBToFile(ctx, transport, ...)` - **Input:** `context.Context, *waMediaTransport.WAMediaTransport_Integral, ...` **Output:** `error` - Downloads FB media directly to file
- ❌ `DownloadHistorySync(ctx, notification, ...)` - **Input:** `context.Context, *waE2E.HistorySyncNotification, ...` **Output:** `*waHistorySync.HistorySync, error` - Downloads history synchronization data
- ❌ `DownloadMediaWithPath(ctx, directPath, ...)` - **Input:** `context.Context, string, []byte, []byte, []byte, ...` **Output:** `[]byte, error` - Downloads media using direct path
- ❌ `DownloadMediaWithPathToFile(ctx, directPath, ...)` - **Input:** `context.Context, string, []byte, []byte, []byte, ...` **Output:** `error` - Downloads media to file using direct path
- ❌ `DownloadThumbnail(ctx, message)` - **Input:** `context.Context, DownloadableThumbnail` **Output:** `[]byte, error` - Downloads message thumbnail image
- ❌ `DownloadToFile(ctx, message, file)` - **Input:** `context.Context, DownloadableMessage, File` **Output:** `error` - Downloads media directly to specified file

## 4. Group Management (20 functions)

### Group Information
- ❌ `GetGroupInfo(jid)` - **Input:** `types.JID` **Output:** `*types.GroupInfo, error` - Gets detailed information about a group
- ❌ `GetGroupInfoFromInvite(jid, inviter, code, expiration)` - **Input:** `types.JID, types.JID, string, int64` **Output:** `*types.GroupInfo, error` - Gets group info using invite details
- ❌ `GetGroupInfoFromLink(code)` - **Input:** `string` **Output:** `*types.GroupInfo, error` - Gets group info from invite link code
- ❌ `GetGroupInviteLink(jid, reset)` - **Input:** `types.JID, bool` **Output:** `string, error` - Gets or resets group invitation link
- ❌ `GetGroupRequestParticipants(jid)` - **Input:** `types.JID` **Output:** `[]types.GroupParticipantRequest, error` - Gets list of pending join requests
- ✅ `GetJoinedGroups()` - **Input:** None **Output:** `[]*types.GroupInfo, error` - Implemented as `WhatsAppGroupGet()`
- ❌ `GetLinkedGroupsParticipants(community)` - **Input:** `types.JID` **Output:** `[]types.JID, error` - Gets participants of linked community groups
- ❌ `GetSubGroups(community)` - **Input:** `types.JID` **Output:** `[]*types.GroupLinkTarget, error` - Gets subgroups within a community

### Group Creation & Membership
- ❌ `CreateGroup(req)` - **Input:** `ReqCreateGroup` **Output:** `*types.GroupInfo, error` - Creates a new WhatsApp group
- ❌ `JoinGroupWithInvite(jid, inviter, code, expiration)` - **Input:** `types.JID, types.JID, string, int64` **Output:** `error` - Joins group using invite details
- ✅ `JoinGroupWithLink(code)` - **Input:** `string` **Output:** `types.JID, error` - Implemented as `WhatsAppGroupJoin()`
- ✅ `LeaveGroup(jid)` - **Input:** `types.JID` **Output:** `error` - Implemented as `WhatsAppGroupLeave()`
- ❌ `LinkGroup(parent, child)` - **Input:** `types.JID, types.JID` **Output:** `error` - Links a subgroup to a community
- ❌ `UnlinkGroup(parent, child)` - **Input:** `types.JID, types.JID` **Output:** `error` - Unlinks a subgroup from a community

### Group Settings
- ❌ `SetGroupAnnounce(jid, announce)` - **Input:** `types.JID, bool` **Output:** `error` - Toggles announcement mode (only admins can send messages)
- ❌ `SetGroupDescription(jid, description)` - **Input:** `types.JID, string` **Output:** `error` - Updates group description
- ❌ `SetGroupJoinApprovalMode(jid, mode)` - **Input:** `types.JID, bool` **Output:** `error` - Enables/disables admin approval for joins
- ❌ `SetGroupLocked(jid, locked)` - **Input:** `types.JID, bool` **Output:** `error` - Locks/unlocks group settings
- ❌ `SetGroupMemberAddMode(jid, mode)` - **Input:** `types.JID, types.GroupMemberAddMode` **Output:** `error` - Sets who can add members (all/admin only)
- ❌ `SetGroupName(jid, name)` - **Input:** `types.JID, string` **Output:** `error` - Changes group name
- ❌ `SetGroupPhoto(jid, avatar)` - **Input:** `types.JID, []byte` **Output:** `string, error` - Updates group profile picture
- ❌ `SetGroupTopic(jid, previousID, newID, topic)` - **Input:** `types.JID, string, string, string` **Output:** `error` - Sets or updates group topic

### Group Participants
- ❌ `UpdateGroupParticipants(jid, participants, action)` - **Input:** `types.JID, []types.JID, ParticipantChange` **Output:** `[]types.GroupParticipant, error` - Adds/removes group participants
- ❌ `UpdateGroupRequestParticipants(jid, participants, action)` - **Input:** `types.JID, []types.JID, ParticipantRequestChange` **Output:** `[]types.GroupParticipant, error` - Approves/rejects join requests

## 5. Contact & Profile Operations (10 functions)

- ❌ `GetBusinessProfile(jid)` - **Input:** `types.JID` **Output:** `*types.BusinessProfile, error` - Gets business account profile information
- ❌ `GetContactQRLink(revoke)` - **Input:** `bool` **Output:** `string, error` - Gets contact QR code link for sharing
- ❌ `GetProfilePictureInfo(jid, params)` - **Input:** `types.JID, *GetProfilePictureParams` **Output:** `*types.ProfilePictureInfo, error` - Gets profile picture URL and metadata
- ❌ `GetUserDevices(jids)` - **Input:** `[]types.JID` **Output:** `[]types.JID, error` - Gets list of user's linked devices
- ❌ `GetUserDevicesContext(ctx, jids)` - **Input:** `context.Context, []types.JID` **Output:** `[]types.JID, error` - Gets user devices with context
- ❌ `GetUserInfo(jids)` - **Input:** `[]types.JID` **Output:** `map[types.JID]types.UserInfo, error` - Gets detailed user information and status
- ✅ `IsOnWhatsApp(phones)` - **Input:** `[]string` **Output:** `[]types.IsOnWhatsAppResponse, error` - Implemented as `WhatsAppCheckRegistered()`, `WhatsAppGetJID()`
- ❌ `SubscribePresence(jid)` - **Input:** `types.JID` **Output:** `error` - Subscribes to real-time presence updates
- ❌ `SetStatusMessage(msg)` - **Input:** `string` **Output:** `error` - Updates profile status/about message

## 6. Privacy & Security (7 functions)

- ❌ `GetBlocklist()` - **Input:** None **Output:** `*types.Blocklist, error` - Gets list of blocked contacts
- ❌ `GetPrivacySettings(ctx)` - **Input:** `context.Context` **Output:** `types.PrivacySettings` - Gets all privacy settings
- ❌ `GetStatusPrivacy()` - **Input:** None **Output:** `[]types.StatusPrivacy, error` - Gets status visibility privacy settings
- ❌ `SetDefaultDisappearingTimer(timer)` - **Input:** `time.Duration` **Output:** `error` - Sets default disappearing messages timer
- ❌ `SetDisappearingTimer(chat, timer)` - **Input:** `types.JID, time.Duration` **Output:** `error` - Sets disappearing messages for specific chat
- ❌ `SetPrivacySetting(ctx, name, value)` - **Input:** `context.Context, types.PrivacySettingType, types.PrivacySetting` **Output:** `types.PrivacySettings, error` - Updates specific privacy setting
- ❌ `TryFetchPrivacySettings(ctx, ignoreCache)` - **Input:** `context.Context, bool` **Output:** `*types.PrivacySettings, error` - Fetches latest privacy settings from server
- ❌ `UpdateBlocklist(jid, action)` - **Input:** `types.JID, events.BlocklistChangeAction` **Output:** `*types.Blocklist, error` - Blocks or unblocks a contact

## 7. Newsletter/Channel Operations (12 functions)

### Newsletter Management
- ❌ `CreateNewsletter(params)` - **Input:** `CreateNewsletterParams` **Output:** `*types.NewsletterMetadata, error` - Creates a new WhatsApp channel/newsletter
- ❌ `FollowNewsletter(jid)` - **Input:** `types.JID` **Output:** `error` - Follows a WhatsApp channel
- ❌ `GetNewsletterInfo(jid)` - **Input:** `types.JID` **Output:** `*types.NewsletterMetadata, error` - Gets channel/newsletter information
- ❌ `GetNewsletterInfoWithInvite(key)` - **Input:** `string` **Output:** `*types.NewsletterMetadata, error` - Gets newsletter info using invite key
- ❌ `GetNewsletterMessageUpdates(jid, params)` - **Input:** `types.JID, *GetNewsletterUpdatesParams` **Output:** `[]*types.NewsletterMessage, error` - Gets recent message updates from channel
- ❌ `GetNewsletterMessages(jid, params)` - **Input:** `types.JID, *GetNewsletterMessagesParams` **Output:** `[]*types.NewsletterMessage, error` - Gets messages from a newsletter
- ❌ `GetSubscribedNewsletters()` - **Input:** None **Output:** `[]*types.NewsletterMetadata, error` - Gets list of followed newsletters
- ❌ `UnfollowNewsletter(jid)` - **Input:** `types.JID` **Output:** `error` - Unfollows a WhatsApp channel

### Newsletter Interactions
- ❌ `NewsletterMarkViewed(jid, serverIDs)` - **Input:** `types.JID, []types.MessageServerID` **Output:** `error` - Marks channel messages as viewed
- ❌ `NewsletterSendReaction(jid, serverID, reaction, ...)` - **Input:** `types.JID, types.MessageServerID, string, ...` **Output:** `error` - Reacts to channel messages
- ❌ `NewsletterSubscribeLiveUpdates(ctx, jid)` - **Input:** `context.Context, types.JID` **Output:** `time.Duration, error` - Subscribes to live channel updates
- ❌ `NewsletterToggleMute(jid, mute)` - **Input:** `types.JID, bool` **Output:** `error` - Mutes/unmutes a newsletter

## 8. Event Handlers & Advanced Features (8 functions)

### Event Management
- ❌ `AddEventHandler(handler)` - **Input:** `EventHandler` **Output:** `uint32` - Adds event handler for incoming events
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

## 9. Push Notifications & Business Features (4 functions)

- ❌ `RegisterForPushNotifications(ctx, config)` - **Input:** `context.Context, PushConfig` **Output:** `error` - Registers device for push notifications
- ❌ `GetServerPushNotificationConfig(ctx)` - **Input:** `context.Context` **Output:** `*waBinary.Node, error` - Gets push notification configuration
- ❌ `ResolveBusinessMessageLink(code)` - **Input:** `string` **Output:** `*types.BusinessMessageLinkTarget, error` - Resolves business message link to contact
- ❌ `ResolveContactQRLink(code)` - **Input:** `string` **Output:** `*types.ContactQRLinkTarget, error` - Resolves contact QR code to user info

## 10. Bot Operations (2 functions)

- ❌ `GetBotListV2()` - **Input:** None **Output:** `[]types.BotListInfo, error` - Gets list of available WhatsApp bots
- ❌ `GetBotProfiles(botInfo)` - **Input:** `[]types.BotListInfo` **Output:** `[]types.BotProfileInfo, error` - Gets detailed information about bots

## 11. Proxy & Connection Settings (8 functions)

- ❌ `SetProxy(proxy, ...)` - **Input:** `Proxy, ...SetProxyOptions` **Output:** None - Sets HTTP proxy for connections
- ✅ `SetProxyAddress(addr, ...)` - **Input:** `string, ...SetProxyOptions` **Output:** `error` - Implemented as `WhatsAppClientProxyURL` in `WhatsAppInitClient()`
- ❌ `SetSOCKSProxy(proxy, ...)` - **Input:** `proxy.Dialer, ...SetProxyOptions` **Output:** None - Sets SOCKS proxy for connections
- ❌ `SetWSDialer(dialer)` - **Input:** `*websocket.Dialer` **Output:** None - Sets custom WebSocket dialer
- ❌ `ToggleProxyOnlyForLogin(only)` - **Input:** `bool` **Output:** None - Uses proxy only during login
- ❌ `SetPassive(ctx, passive)` - **Input:** `context.Context, bool` **Output:** `error` - Sets passive connection mode
- ❌ `SetForceActiveDeliveryReceipts(active)` - **Input:** `bool` **Output:** None - Forces active delivery receipts
- ❌ `StoreLIDPNMapping(ctx, first, second)` - **Input:** `context.Context, types.JID, types.JID` **Output:** None - Stores LID to PN identifier mapping

## 12. Miscellaneous (4 functions)

- ❌ `AcceptTOSNotice(noticeID, stage)` - **Input:** `string, string` **Output:** `error` - Accepts terms of service notices
- ❌ `DangerousInternals()` - **Input:** None **Output:** `*DangerousInternalClient` - Accesses dangerous internal functions
- ❌ `FetchAppState(ctx, name, fullSync, onlyIfNotSynced)` - **Input:** `context.Context, appstate.WAPatchName, bool, bool` **Output:** `error` - Fetches application state patches
- ❌ `RejectCall(callFrom, callID)` - **Input:** `types.JID, string` **Output:** `error` - Rejects incoming voice/video calls
- ❌ `MarkRead(ids, timestamp, chat, sender, ...)` - **Input:** `[]types.MessageID, time.Time, types.JID, types.JID, ...` **Output:** `error` - Marks messages as read

---

## Summary

- **Total Functions**: 120
- **Implemented Functions**: ~22 (18%)
- **Missing Functions**: ~98 (82%)
- **Most Populated Category**: Group Management (20 functions)
- **Least Populated Category**: Bot Operations (2 functions)
- **Most Implemented Category**: Connection & Authentication (7/8 ✅)
- **Least Implemented Category**: Privacy & Security, Newsletter/Channel Operations, Event Handlers (0/7-12 ❌)
- **Key Missing Features**: Group management (17/20 missing), Privacy settings (7/7 missing), Newsletter operations (12/12 missing), Advanced message features (7/8 missing)

**Priority Implementation Areas:**
1. **High Priority**: Complete group management functions (participant management, group settings, creation)
2. **Medium Priority**: Privacy and security settings (blocking, disappearing messages)
3. **Low Priority**: Newsletter features, advanced message processing, push notifications

**Legend**: ✅ = Implemented, ❌ = Not Implemented

This comprehensive list represents all available functionality in the latest whatsmeow library for interacting with WhatsApp's API.
