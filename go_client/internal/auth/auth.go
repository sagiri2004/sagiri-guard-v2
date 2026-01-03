package auth

type Credentials struct {
	Username  string `json:"username"`
	DeviceID  string `json:"device_id"`
	Name      string `json:"name,omitempty"`
	OSName    string `json:"os_name,omitempty"`
	OSVersion string `json:"os_version,omitempty"`
	Hostname  string `json:"hostname,omitempty"`
	Arch      string `json:"arch,omitempty"`
}
