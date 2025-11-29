package types

// RequestCreateDevice is the request for creating a new device
type RequestCreateDevice struct {
	DeviceName string `json:"device_name" form:"device_name"`
}

// RequestRegenerateToken is the request for regenerating a device JWT
type RequestRegenerateToken struct {
	DeviceID     string `json:"device_id"`
	DeviceSecret string `json:"device_secret"`
}
