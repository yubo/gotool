package util

import (
	"fmt"
	"net"
	"net/http"
	"strings"
)

// GetRemoteIp host:port "192.0.2.1:25", "[2001:db8::1]:80"
func GetRemoteIP(remoteAddr string) (net.IP, error) {
	if host, _, err := net.SplitHostPort(remoteAddr); err == nil {
		remoteAddr = host
	}

	ip := net.ParseIP(remoteAddr)
	if ip == nil {
		return nil, fmt.Errorf("remoteAddr: %q is invaild ip address", remoteAddr)
	}
	return ip, nil
}

func IPContains(ip net.IP, network string) bool {
	_, ipNet, err := net.ParseCIDR(network)
	if err != nil {
		return false
	}
	return ipNet.Contains(ip)
}

// http

// GetIPAdress "X-Forwarded-For", "X-Real-Ip", req.RemoteAddr
func GetIPAdress(r *http.Request) string {
	for _, h := range []string{"X-Forwarded-For", "X-Real-Ip"} {
		for _, ip := range strings.Split(r.Header.Get(h), ",") {
			// header can contain spaces too, strip those out.
			ip = strings.TrimSpace(ip)
			realIP := net.ParseIP(ip)
			if realIP.IsGlobalUnicast() {
				return ip
			}
		}
	}

	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	return ip
}

func SetContentType(header http.Header, contentType string) error {
	if t := header.Get("Content-Type"); t == "" {
		header.Set("Content-Type", contentType)
		return nil
	} else if t == contentType {
		return nil
	} else {
		return fmt.Errorf("Content-Type already set %s befor %s", t, contentType)
	}
}
