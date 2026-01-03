package firewall

import (
	"demo/network/go_client/internal/logger"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

type ProxyManager struct {
	port    int
	server  *http.Server
	enabled bool
}

var globalProxyManager *ProxyManager

func GetProxyManager() *ProxyManager {
	if globalProxyManager == nil {
		globalProxyManager = &ProxyManager{
			port: 8080,
		}
	}
	return globalProxyManager
}

func (p *ProxyManager) Start() {
	if p.enabled {
		return
	}

	p.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", p.port),
		Handler: http.HandlerFunc(p.handleProxy),
	}

	p.enabled = true
	go func() {
		logger.Infof("Proxy server starting on :%d", p.port)

		// Tự động cấu hình proxy cho OS
		p.SetOSProxy()

		if err := p.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Errorf("Proxy server error: %v", err)
			p.enabled = false
		}
	}()
}

func (p *ProxyManager) Stop() {
	if !p.enabled || p.server == nil {
		return
	}

	logger.Info("Proxy server stopping...")

	// Khôi phục cài đặt proxy cũ cho OS
	p.UnsetOSProxy()

	p.server.Close()
	p.enabled = false
}

func (p *ProxyManager) handleProxy(w http.ResponseWriter, r *http.Request) {
	// 1. Kiểm tra xem domain có trong danh sách bị chặn không
	host := r.Host
	if strings.Contains(host, ":") {
		h, _, err := net.SplitHostPort(host)
		if err == nil {
			host = h
		}
	}

	if p.isBlocked(host) {
		logger.Infof("Proxy blocked request to: %s", host)
		if r.Method == http.MethodConnect {
			// Đối với HTTPS (CONNECT), ta chỉ đơn giản ngắt kết nối
			w.WriteHeader(http.StatusForbidden)
			return
		}
		// Đối với HTTP, ta có thể hiển thị trang "Blocked"
		p.renderBlockedPage(w, host)
		return
	}

	// 2. Nếu không bị chặn, tiến hành Proxy
	if r.Method == http.MethodConnect {
		p.handleHTTPS(w, r)
	} else {
		p.handleHTTP(w, r)
	}
}

func (p *ProxyManager) isBlocked(host string) bool {
	mgr := GetHostsManager()
	if mgr == nil {
		return false
	}

	blockedDomains := mgr.GetDomains()
	host = strings.ToLower(strings.TrimSpace(host))

	for _, d := range blockedDomains {
		d = strings.ToLower(strings.TrimSpace(d))
		if host == d || strings.HasSuffix(host, "."+d) {
			return true
		}
	}
	return false
}

func (p *ProxyManager) handleHTTP(w http.ResponseWriter, r *http.Request) {
	// Xóa Hop-by-hop headers
	r.RequestURI = ""

	resp, err := http.DefaultTransport.RoundTrip(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()

	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func (p *ProxyManager) handleHTTPS(w http.ResponseWriter, r *http.Request) {
	destConn, err := net.DialTimeout("tcp", r.Host, 10*time.Second)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)

	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		return
	}
	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		destConn.Close()
		return
	}

	go p.transfer(destConn, clientConn)
	go p.transfer(clientConn, destConn)
}

func (p *ProxyManager) transfer(destination io.WriteCloser, source io.ReadCloser) {
	defer destination.Close()
	defer source.Close()
	io.Copy(destination, source)
}

func (p *ProxyManager) renderBlockedPage(w http.ResponseWriter, host string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusForbidden)
	fmt.Fprintf(w, `
<html>
<head>
    <title>Truy cập bị chặn - Sagiri Guard</title>
    <style>
        body { font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif; background: #1a1a1a; color: #fff; display: flex; align-items: center; justify-content: center; height: 100vh; margin: 0; }
        .container { background: rgba(255, 255, 255, 0.05); backdrop-filter: blur(10px); padding: 50px; border-radius: 20px; border: 1px solid rgba(255, 255, 255, 0.1); text-align: center; box-shadow: 0 10px 30px rgba(0,0,0,0.5); }
        h1 { color: #ff4d4d; margin-bottom: 20px; }
        p { font-size: 1.2em; color: #ccc; }
        .host { font-weight: bold; color: #00d4ff; border-bottom: 2px solid #00d4ff; padding-bottom: 2px; }
        .icon { font-size: 50px; margin-bottom: 20px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="icon">🚫</div>
        <h1>Truy cập bị chặn!</h1>
        <p>Trang web <span class="host">%s</span> đã bị chặn bởi quản trị viên hệ thống (Sagiri Guard).</p>
        <p>Vui lòng liên hệ IT để được hỗ trợ.</p>
    </div>
</body>
</html>
`, host)
}
