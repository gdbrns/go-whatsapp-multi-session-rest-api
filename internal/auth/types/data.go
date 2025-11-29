package types

// DeviceAuthContext holds device authentication context
type DeviceAuthContext struct {
	DeviceID     string `json:"device_id"`
	DeviceSecret string `json:"-"` // Never expose in responses
	DeviceName   string `json:"device_name"`
	APIKeyID     int    `json:"api_key_id"`
	WhatsmeowJID string `json:"whatsmeow_jid,omitempty"`
	Status       string `json:"status"`
}

// APIKeyAuthContext holds API key authentication context (for admin/device creation)
type APIKeyAuthContext struct {
	ID           int    `json:"id"`
	APIKey       string `json:"-"` // Never expose the full key
	CustomerName string `json:"customer_name"`
	MaxDevices   int    `json:"max_devices"`
	RateLimit    int    `json:"rate_limit_per_hour"`
}
