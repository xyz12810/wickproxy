package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"time"
)

type userConfig struct {
	Username string
	Password string
	Quato    int
}

type aclConfig struct {
	IsAllow bool
	Domain  string
	Addr    net.IPNet
	Port    string
}

type globalConfig struct {
	Server string

	SecureURL   string
	FallbackURL string

	Timeout time.Duration
	Logging string
	PID     int

	TLS struct {
		Certificate    string
		CertificateKey string
	}

	Users []userConfig
	ACL   []aclConfig
}

// GlobalConfig is global configuration
var GlobalConfig globalConfig = globalConfig{
	Server:  "0.0.0.0:7890",
	Timeout: 10,
	Users:   make([]userConfig, 0),
	ACL:     make([]aclConfig, 0)}

func configReader(configFile string) error {
	file, err := os.Open(configFile)
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&GlobalConfig)
	if err != nil {
		return err
	}
	log.Debugln("[config] read config from", configFile)
	return nil
}

func configWriter(configFile string) error {
	file, err := os.Create(configFile)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	err = encoder.Encode(GlobalConfig)
	if err != nil {
		return err
	}
	log.Debugln("[config] write config to", configFile)
	return nil
}

func configPrint() error {
	body, err := json.MarshalIndent(GlobalConfig, "", "  ")
	if err != nil {
		return err
	}
	fmt.Printf("%s\n", body)
	return nil
}

func configExists(configFile string) bool {
	_, err := os.Stat(configFile)
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		return false
	}
	return true
}
