//go:build !linux

package firewall

type HostsManager struct {
	enabled bool
}

func GetHostsManager() *HostsManager {
	return &HostsManager{}
}

func (h *HostsManager) GetDomains() []string {
	return []string{}
}

func (h *HostsManager) SetDomains(domains []string) error {
	return nil
}

func (h *HostsManager) IsEnabled() bool {
	return false
}

func (h *HostsManager) SetEnabled(enabled bool) error {
	return nil
}
