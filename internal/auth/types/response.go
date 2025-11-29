package types

// ResponseDeviceCreated is the response for new device creation
type ResponseDeviceCreated struct {
	DeviceID     string `json:"device_id"`
	DeviceSecret string `json:"device_secret"`
	DeviceName   string `json:"device_name"`
	Token        string `json:"token"`
	Message      string `json:"message"`
}

// ResponseTokenRegenerated is the response for JWT regeneration
type ResponseTokenRegenerated struct {
	DeviceID string `json:"device_id"`
	Token    string `json:"token"`
	Message  string `json:"message"`
}

// ResponseAPIKeyCreated is the response for API key creation (admin only)
type ResponseAPIKeyCreated struct {
	ID           int    `json:"id"`
	APIKey       string `json:"api_key"`
	CustomerName string `json:"customer_name"`
	MaxDevices   int    `json:"max_devices"`
	RateLimit    int    `json:"rate_limit_per_hour"`
}
