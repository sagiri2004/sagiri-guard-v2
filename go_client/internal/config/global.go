package config

import "sync"

type FirewallState struct {
	Enabled bool
	Domains []string
	Mu      sync.RWMutex
}

var GlobalFirewall FirewallState

func UpdateFirewallConfig(enabled bool, domains []string) {
	GlobalFirewall.Mu.Lock()
	defer GlobalFirewall.Mu.Unlock()
	GlobalFirewall.Enabled = enabled
	GlobalFirewall.Domains = domains
}
