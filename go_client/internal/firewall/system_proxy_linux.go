//go:build linux

package firewall

import (
	"demo/network/go_client/internal/logger"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// SetOSProxy cấu hình proxy cho hệ điều hành
func (p *ProxyManager) SetOSProxy() {
	user := os.Getenv("SUDO_USER")
	if user == "" {
		user = os.Getenv("USER")
	}

	if user == "" || user == "root" {
		logger.Warn("Could not determine non-root user for gsettings. System proxy might not be set.")
		return
	}

	logger.Infof("Setting system proxy for user: %s", user)

	// Các lệnh gsettings để cấu hình proxy cho GNOME
	p.runGsettings(user, "set", "org.gnome.system.proxy", "mode", "'manual'")
	p.runGsettings(user, "set", "org.gnome.system.proxy.http", "host", "'127.0.0.1'")
	p.runGsettings(user, "set", "org.gnome.system.proxy.http", "port", fmt.Sprintf("%d", p.port))
	p.runGsettings(user, "set", "org.gnome.system.proxy.https", "host", "'127.0.0.1'")
	p.runGsettings(user, "set", "org.gnome.system.proxy.https", "port", fmt.Sprintf("%d", p.port))
}

// UnsetOSProxy gỡ bỏ cấu hình proxy của hệ điều hành
func (p *ProxyManager) UnsetOSProxy() {
	user := os.Getenv("SUDO_USER")
	if user == "" {
		user = os.Getenv("USER")
	}

	if user == "" || user == "root" {
		return
	}

	logger.Infof("Restoring system proxy for user: %s", user)
	p.runGsettings(user, "set", "org.gnome.system.proxy", "mode", "'none'")
}

func (p *ProxyManager) runGsettings(user string, args ...string) {
	// 1. Lấy UID của user
	uidCmd := exec.Command("id", "-u", user)
	uidBytes, err := uidCmd.Output()
	if err != nil {
		logger.Warnf("Failed to get UID for user %s: %v", user, err)
		return
	}
	uid := strings.TrimSpace(string(uidBytes))

	// 2. Chạy gsettings với DBUS_SESSION_BUS_ADDRESS
	// Thông thường session bus nằm ở /run/user/UID/bus
	dbusAddr := fmt.Sprintf("unix:path=/run/user/%s/bus", uid)

	fullArgs := append([]string{"-u", user, "env", "DBUS_SESSION_BUS_ADDRESS=" + dbusAddr, "gsettings"}, args...)
	cmd := exec.Command("sudo", fullArgs...)

	if out, err := cmd.CombinedOutput(); err != nil {
		logger.Warnf("Failed to execute gsettings %v for user %s: %v (Output: %s)", args, user, err, string(out))
	}
}
