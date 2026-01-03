//go:build !linux

package firewall

// SetOSProxy stub cho các nền tảng không phải Linux
func (p *ProxyManager) SetOSProxy() {}

// UnsetOSProxy stub cho các nền tảng không phải Linux
func (p *ProxyManager) UnsetOSProxy() {}
