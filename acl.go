package main

import (
	"fmt"
	"net"
	"strings"
)

func aclPrivateCheck(host string) bool {
	ip := net.ParseIP(host)
	if ip == nil {
		return true
	}

	return ip.IsGlobalUnicast()
}

func aclAdd(index int, context, action string) {

	if action != "allow" {
		action = "deny"
	}

	var err error
	var host, port, domain string
	var in net.IPNet

	if strings.Contains(context, "/") {
		_, tin, err := net.ParseCIDR(context)
		in = net.IPNet{IP: tin.IP, Mask: tin.Mask}
		if err != nil {
			log.Fatalln("[ACL] invalid CIDR:", context)
		}
		domain = ""
		port = ""
	} else {
		host, port, err = net.SplitHostPort(context)
		if err != nil {
			host = context
			port = ""
		}
		ip := net.ParseIP(host)
		if ip == nil {
			domain = host
			in = net.IPNet{}
		} else if ip.To4() != nil {
			in = net.IPNet{IP: ip, Mask: net.CIDRMask(32, 32)}
			domain = ""
		} else {
			in = net.IPNet{IP: ip, Mask: net.CIDRMask(128, 128)}
			domain = ""
		}
	}

	var act bool
	if action == "allow" {
		act = true
	} else {
		act = false
	}

	newItem := aclConfig{IsAllow: act, Domain: domain, Addr: in, Port: port}

	if index < 0 || index >= len(GlobalConfig.ACL) {
		GlobalConfig.ACL = append(GlobalConfig.ACL, newItem)
	} else {
		rear := append([]aclConfig{}, GlobalConfig.ACL[index:]...)
		GlobalConfig.ACL = append(append(GlobalConfig.ACL[:index], newItem), rear...)
	}

	return
}

func aclDel(idx int, all bool) {
	if idx == -1 {
		idx = len(GlobalConfig.ACL) - 1
	}

	if idx < 0 || idx >= len(GlobalConfig.ACL) {
		log.Fatalln("[ACL] index out of range.")
		return
	}

	if all {
		GlobalConfig.ACL = make([]aclConfig, 0)
		return
	}

	tmpACL := make([]aclConfig, 0)
	for i, v := range GlobalConfig.ACL {
		if i != idx {
			tmpACL = append(tmpACL, v)
		}
	}
	GlobalConfig.ACL = tmpACL
	return
}

func acllist() {
	err := configReader(*config)
	if err != nil {
		log.Fatalln("[user] read config error:", err)
	}

	for i, v := range GlobalConfig.ACL {
		fmt.Printf("[%v]\t", i)

		if v.IsAllow {
			fmt.Printf("allow\t")
		} else {
			fmt.Printf("deny\t")
		}

		if v.Domain != "" {
			fmt.Printf("%v", v.Domain)
		} else {
			fmt.Printf("%v", v.Addr.String())
		}
		if v.Port != "" {
			fmt.Printf(":%v", v.Port)
		}
		fmt.Printf("\n")
	}
}

func aclCheck(host, port string) bool {
	for _, v := range GlobalConfig.ACL {
		log.Debugln("[ACL] request:",host,port,"rule:", v)
		if aclMatch(host, port, v) {
			if v.IsAllow {
				return true
			}
			return false
		}
	}
	return true
}

func aclMatch(host, port string, rule aclConfig) bool {
	if rule.Port != "" && port != rule.Port {
		return false
	}

	IP := net.ParseIP(host)
	if IP == nil {
		if host == rule.Domain {
			return true
		}
		return false
	}

	if rule.Addr.Contains(IP) {
		return true
	}
	return false
}
