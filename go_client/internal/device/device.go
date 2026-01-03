package device

import (
	"os"
	"os/exec"
	"runtime"
	"strings"
)

type SystemInfo struct {
	OSName    string `json:"os_name"`
	OSVersion string `json:"os_version"`
	Hostname  string `json:"hostname"`
	Arch      string `json:"arch"`
	UUID      string `json:"uuid"`
}

func GetSystemInfo() (SystemInfo, error) {
	info := SystemInfo{
		OSName: runtime.GOOS,
		Arch:   runtime.GOARCH,
	}

	// Hostname
	hostname, err := os.Hostname()
	if err == nil {
		info.Hostname = hostname
	}

	// OS Version (Linux specific for now as per prompt context)
	if runtime.GOOS == "linux" {
		out, err := exec.Command("uname", "-r").Output()
		if err == nil {
			info.OSVersion = strings.TrimSpace(string(out))
		}
	}

	// UUID (Requires Admin/Root)
	// Try /sys/class/dmi/id/product_uuid first (often readable)
	uuidBytes, err := os.ReadFile("/sys/class/dmi/id/product_uuid")
	if err == nil {
		info.UUID = strings.TrimSpace(string(uuidBytes))
	} else {
		// Fallback to dmidecode
		out, err := exec.Command("dmidecode", "-s", "system-uuid").Output()
		if err == nil {
			info.UUID = strings.TrimSpace(string(out))
		} else {
			// Fallback if not root or fails: Generate a random one or use hostname+user
			// The prompt says "if run user uuid query will be different/fail", implies we should try hard.
			// Returing error might be better if strict, but let's return partial.
			info.UUID = "UNKNOWN-UUID-REQUIRES-ROOT"
		}
	}

	return info, nil
}
