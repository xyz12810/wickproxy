package main

import (
	"encoding/json"
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
)

type globalConfig struct {
	Server string

	HideURL string
	FailURL string

	TLS struct {
		Certificate    string
		CertificateKey string
	}

	Users []struct {
		Username string
		Password string
		Quato    int
	}
}

// GlobalConfig is global configuration
var GlobalConfig globalConfig

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
	log.Debug("Read config from", configFile)
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
	log.Debug("Write config to", configFile)
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
