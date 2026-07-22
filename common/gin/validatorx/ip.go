package validatorx

import (
	"net"
)

// IsIPv4 判断字符串是否为有效的 IPv4 地址
func IsIPv4(s string) bool {
	ip := net.ParseIP(s)
	if ip == nil {
		return false
	}
	return ip.To4() != nil
}
