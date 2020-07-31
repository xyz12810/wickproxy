package main

import "net"

func aclPrivateCheck(host string) bool {
	ip := net.ParseIP(host)
	if ip == nil {
		return true
	}

	return ip.IsGlobalUnicast()
}

func aclHostCheck(host string) bool {
	return true
}

func aclPortCheck(port string) bool {
	return true
}
