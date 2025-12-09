package internal

import (
	"github.com/gofiber/fiber/v2"
	swagger "github.com/gofiber/swagger"

	"github.com/gdbrns/go-whatsapp-multi-session-rest-api/pkg/auth"
	"github.com/gdbrns/go-whatsapp-multi-session-rest-api/pkg/router"

	ctlAdmin "github.com/gdbrns/go-whatsapp-multi-session-rest-api/internal/admin"
	ctlAppState "github.com/gdbrns/go-whatsapp-multi-session-rest-api/internal/appstate"
	ctlAuth "github.com/gdbrns/go-whatsapp-multi-session-rest-api/internal/auth"
	ctlDevice "github.com/gdbrns/go-whatsapp-multi-session-rest-api/internal/device"
	ctlGroups "github.com/gdbrns/go-whatsapp-multi-session-rest-api/internal/groups"
	ctlIndex "github.com/gdbrns/go-whatsapp-multi-session-rest-api/internal/index"
	ctlMessage "github.com/gdbrns/go-whatsapp-multi-session-rest-api/internal/message"
	ctlMessaging "github.com/gdbrns/go-whatsapp-multi-session-rest-api/internal/messaging"
	ctlPresence "github.com/gdbrns/go-whatsapp-multi-session-rest-api/internal/presence"
	ctlUser "github.com/gdbrns/go-whatsapp-multi-session-rest-api/internal/user"
	ctlWebhooks "github.com/gdbrns/go-whatsapp-multi-session-rest-api/internal/webhooks"
)

func Routes(app *fiber.App) {
	// Configure OpenAPI / Swagger
	specURL := router.BaseURL + "/docs/swagger.json"
	swaggerHandler := swagger.New(swagger.Config{
		URL: specURL,
	})

	// Route for Index
	// ---------------------------------------------
	if router.BaseURL == "" {
		app.Get("/", ctlIndex.Index)
	} else {
		app.Get(router.BaseURL, ctlIndex.Index)
		app.Get(router.BaseURL+"/", ctlIndex.Index)
	}

	// Route for OpenAPI / Swagger
	// ---------------------------------------------
	app.Get(router.BaseURL+"/docs/swagger.json", func(c *fiber.Ctx) error {
		return c.SendFile("docs/swagger.json")
	})
	app.Get(router.BaseURL+"/docs/swagger.yaml", func(c *fiber.Ctx) error {
		return c.SendFile("docs/swagger.yaml")
	})
	app.Get(router.BaseURL+"/docs/*", swaggerHandler)

	// ============================================================
	// ADMIN ROUTES (X-Admin-Secret authentication)
	// ============================================================
	adminMiddleware := auth.AdminAuth()
	
	// Admin Dashboard APIs
	app.Get(router.BaseURL+"/admin/stats", adminMiddleware, ctlAdmin.GetStats)
	app.Get(router.BaseURL+"/admin/health", adminMiddleware, ctlAdmin.GetHealth)
	app.Get(router.BaseURL+"/admin/devices", adminMiddleware, ctlAdmin.ListAllDevices)
	app.Get(router.BaseURL+"/admin/devices/status", adminMiddleware, ctlAdmin.GetAllDevicesStatus)
	app.Post(router.BaseURL+"/admin/devices/reconnect", adminMiddleware, ctlAdmin.ReconnectAllDevices)
	app.Get(router.BaseURL+"/admin/webhooks/stats", adminMiddleware, ctlAdmin.GetWebhookStats)
	
	// API Key Management
	app.Post(router.BaseURL+"/admin/api-keys", adminMiddleware, ctlAdmin.CreateAPIKey)
	app.Get(router.BaseURL+"/admin/api-keys", adminMiddleware, ctlAdmin.ListAPIKeys)
	app.Get(router.BaseURL+"/admin/api-keys/:id", adminMiddleware, ctlAdmin.GetAPIKey)
	app.Patch(router.BaseURL+"/admin/api-keys/:id", adminMiddleware, ctlAdmin.UpdateAPIKey)
	app.Delete(router.BaseURL+"/admin/api-keys/:id", adminMiddleware, ctlAdmin.DeleteAPIKey)
	app.Get(router.BaseURL+"/admin/api-keys/:id/devices", adminMiddleware, ctlAdmin.ListDevicesByAPIKey)
	app.Get(router.BaseURL+"/admin/api-keys/:id/devices/status", adminMiddleware, ctlAdmin.GetAllDeviceStatuses)
	app.Delete(router.BaseURL+"/admin/devices/:device_id", adminMiddleware, ctlAdmin.DeleteDevice)

	// ============================================================
	// DEVICE CREATION (X-API-Key authentication)
	// ============================================================
	apiKeyMiddleware := auth.APIKeyAuth()
	app.Post(router.BaseURL+"/devices", apiKeyMiddleware, ctlAuth.CreateDevice)

	// ============================================================
	// TOKEN REGENERATION (No auth - uses device credentials in body)
	// ============================================================
	app.Post(router.BaseURL+"/devices/token", ctlAuth.RegenerateToken)

	// ============================================================
	// DEVICE OPERATIONS (JWT Bearer token authentication)
	// All WhatsApp operations require valid JWT token
	// ============================================================
	deviceAuthMiddleware := auth.DeviceAuth()

	// Device management
	app.Get(router.BaseURL+"/devices/me", deviceAuthMiddleware, ctlDevice.GetDeviceMe)
	app.Get(router.BaseURL+"/devices/me/status", deviceAuthMiddleware, ctlDevice.GetStatus)
	app.Post(router.BaseURL+"/devices/me/login", deviceAuthMiddleware, ctlDevice.Login)
	app.Post(router.BaseURL+"/devices/me/login-code", deviceAuthMiddleware, ctlDevice.LoginWithCode)
	app.Post(router.BaseURL+"/devices/me/reconnect", deviceAuthMiddleware, ctlDevice.Reconnect)
	app.Delete(router.BaseURL+"/devices/me/session", deviceAuthMiddleware, ctlDevice.Logout)
	app.Get(router.BaseURL+"/devices/me/contacts/:phone/registered", deviceAuthMiddleware, ctlDevice.CheckRegistered)

	// User routes
	app.Get(router.BaseURL+"/users/:user_jid", deviceAuthMiddleware, ctlUser.GetInfo)
	app.Get(router.BaseURL+"/users/:user_jid/profile-picture", deviceAuthMiddleware, ctlUser.GetProfilePicture)
	app.Post(router.BaseURL+"/users/:user_jid/block", deviceAuthMiddleware, ctlUser.BlockUser)
	app.Delete(router.BaseURL+"/users/:user_jid/block", deviceAuthMiddleware, ctlUser.UnblockUser)
	app.Get(router.BaseURL+"/users/me/privacy", deviceAuthMiddleware, ctlUser.GetPrivacy)
	app.Patch(router.BaseURL+"/users/me/privacy", deviceAuthMiddleware, ctlUser.UpdatePrivacy)
	app.Get(router.BaseURL+"/users/me/status-privacy", deviceAuthMiddleware, ctlUser.GetStatusPrivacy)
	app.Post(router.BaseURL+"/users/me/status", deviceAuthMiddleware, ctlUser.UpdateStatus)
	app.Get(router.BaseURL+"/users/:jid/devices", deviceAuthMiddleware, ctlUser.GetDevices)
	app.Post(router.BaseURL+"/users/me/profile-photo", deviceAuthMiddleware, ctlUser.SetProfilePhoto)
	app.Get(router.BaseURL+"/users/me/contacts", deviceAuthMiddleware, ctlUser.GetContacts)
	app.Post(router.BaseURL+"/users/me/contacts/sync", deviceAuthMiddleware, ctlUser.ContactSync)
	app.Get(router.BaseURL+"/users/me/blocklist", deviceAuthMiddleware, ctlUser.GetBlocklist)

	// Chat/Messaging routes
	app.Post(router.BaseURL+"/chats/:chat_jid/messages", deviceAuthMiddleware, ctlMessaging.SendText)
	app.Post(router.BaseURL+"/chats/:chat_jid/images", deviceAuthMiddleware, ctlMessaging.SendImage)
	app.Post(router.BaseURL+"/chats/:chat_jid/documents", deviceAuthMiddleware, ctlMessaging.SendDocument)
	app.Get(router.BaseURL+"/chats/:chat_jid/messages", deviceAuthMiddleware, ctlMessaging.GetMessages)
	app.Post(router.BaseURL+"/chats/:chat_jid/archive", deviceAuthMiddleware, ctlMessaging.ArchiveChat)
	app.Post(router.BaseURL+"/chats/:chat_jid/pin", deviceAuthMiddleware, ctlMessaging.PinChat)

	// Message routes
	app.Post(router.BaseURL+"/messages/:message_id/read", deviceAuthMiddleware, ctlMessage.MarkRead)
	app.Post(router.BaseURL+"/messages/:message_id/reaction", deviceAuthMiddleware, ctlMessage.React)
	app.Patch(router.BaseURL+"/messages/:message_id", deviceAuthMiddleware, ctlMessage.Edit)
	app.Delete(router.BaseURL+"/messages/:message_id", deviceAuthMiddleware, ctlMessage.Delete)
	app.Post(router.BaseURL+"/messages/:message_id/reply", deviceAuthMiddleware, ctlMessage.Reply)
	app.Post(router.BaseURL+"/messages/:message_id/forward", deviceAuthMiddleware, ctlMessage.Forward)

	// Group routes
	app.Get(router.BaseURL+"/groups", deviceAuthMiddleware, ctlGroups.List)
	app.Get(router.BaseURL+"/groups/:group_jid", deviceAuthMiddleware, ctlGroups.GetInfo)
	app.Post(router.BaseURL+"/groups", deviceAuthMiddleware, ctlGroups.Create)
	app.Post(router.BaseURL+"/groups/:group_jid/leave", deviceAuthMiddleware, ctlGroups.Leave)
	app.Patch(router.BaseURL+"/groups/:group_jid/name", deviceAuthMiddleware, ctlGroups.UpdateName)
	app.Patch(router.BaseURL+"/groups/:group_jid/description", deviceAuthMiddleware, ctlGroups.UpdateDescription)
	app.Post(router.BaseURL+"/groups/:group_jid/photo", deviceAuthMiddleware, ctlGroups.UpdatePhoto)
	app.Get(router.BaseURL+"/groups/:group_jid/invite-link", deviceAuthMiddleware, ctlGroups.GetInviteLink)
	app.Patch(router.BaseURL+"/groups/:group_jid/settings", deviceAuthMiddleware, ctlGroups.UpdateSettings)
	app.Get(router.BaseURL+"/groups/:group_jid/participant-requests", deviceAuthMiddleware, ctlGroups.GetParticipantRequests)
	app.Post(router.BaseURL+"/groups/:group_jid/join-approval", deviceAuthMiddleware, ctlGroups.SetJoinApproval)
	app.Get(router.BaseURL+"/groups/invite/:invite_code", deviceAuthMiddleware, ctlGroups.GetInfoFromInvite)
	app.Post(router.BaseURL+"/groups/:group_jid/join-invite", deviceAuthMiddleware, ctlGroups.JoinWithInvite)
	app.Patch(router.BaseURL+"/groups/:group_jid/member-add-mode", deviceAuthMiddleware, ctlGroups.SetMemberAddMode)
	app.Patch(router.BaseURL+"/groups/:group_jid/topic", deviceAuthMiddleware, ctlGroups.SetTopic)
	app.Post(router.BaseURL+"/groups/:parent_group_jid/link/:group_jid", deviceAuthMiddleware, ctlGroups.LinkGroup)
	app.Get(router.BaseURL+"/groups/:community_jid/linked-participants", deviceAuthMiddleware, ctlGroups.GetLinkedParticipants)
	app.Get(router.BaseURL+"/groups/:community_jid/subgroups", deviceAuthMiddleware, ctlGroups.GetSubGroups)
	app.Post(router.BaseURL+"/groups/:group_jid/participants", deviceAuthMiddleware, ctlGroups.AddParticipants)
	app.Delete(router.BaseURL+"/groups/:group_jid/participants", deviceAuthMiddleware, ctlGroups.RemoveParticipants)
	app.Post(router.BaseURL+"/groups/:group_jid/requests/approve", deviceAuthMiddleware, ctlGroups.ApproveRequests)
	app.Post(router.BaseURL+"/groups/:group_jid/requests/reject", deviceAuthMiddleware, ctlGroups.RejectRequests)
	app.Post(router.BaseURL+"/groups/:group_jid/admins", deviceAuthMiddleware, ctlGroups.PromoteAdmins)
	app.Delete(router.BaseURL+"/groups/:group_jid/admins", deviceAuthMiddleware, ctlGroups.DemoteAdmins)

	// Presence routes
	app.Post(router.BaseURL+"/chats/:chat_jid/presence", deviceAuthMiddleware, ctlPresence.SendChatPresence)
	app.Post(router.BaseURL+"/presence/status", deviceAuthMiddleware, ctlPresence.UpdateStatus)
	app.Patch(router.BaseURL+"/chats/:chat_jid/disappearing-timer", deviceAuthMiddleware, ctlPresence.SetDisappearingTimer)

	// App state routes
	app.Get(router.BaseURL+"/app-state/:name", deviceAuthMiddleware, ctlAppState.FetchAppState)
	app.Post(router.BaseURL+"/app-state", deviceAuthMiddleware, ctlAppState.SendAppState)
	app.Post(router.BaseURL+"/app-state/mark-clean", deviceAuthMiddleware, ctlAppState.MarkNotDirty)

	// Webhook routes
	app.Get(router.BaseURL+"/webhooks", deviceAuthMiddleware, ctlWebhooks.ListWebhooks)
	app.Post(router.BaseURL+"/webhooks", deviceAuthMiddleware, ctlWebhooks.CreateWebhook)
	app.Get(router.BaseURL+"/webhooks/:webhook_id", deviceAuthMiddleware, ctlWebhooks.GetWebhook)
	app.Patch(router.BaseURL+"/webhooks/:webhook_id", deviceAuthMiddleware, ctlWebhooks.UpdateWebhook)
	app.Delete(router.BaseURL+"/webhooks/:webhook_id", deviceAuthMiddleware, ctlWebhooks.DeleteWebhook)
	app.Get(router.BaseURL+"/webhooks/:webhook_id/logs", deviceAuthMiddleware, ctlWebhooks.GetWebhookLogs)
	app.Post(router.BaseURL+"/webhooks/:webhook_id/test", deviceAuthMiddleware, ctlWebhooks.TestWebhook)
}
