package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type userConfig struct {
	Username string
	Password string
	Quato    int
}

type globalConfig struct {
	Server string

	SecureURL  string
	ReverseURL string

	Timeout time.Duration

	TLS struct {
		Certificate    string
		CertificateKey string
	}

	Users []userConfig
}

// GlobalConfig is global configuration
var GlobalConfig globalConfig = globalConfig{Users: make([]userConfig, 0)}

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
	log.Debugln("Read config from", configFile)
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
	log.Debugln("Write config to", configFile)
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
