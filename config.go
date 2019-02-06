package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/BurntSushi/toml"
)

type Credentials struct {
	Credentials []*Credential
}

func LoadCredentials() (*Credentials, error) {
	// Find config file path
	cfgPath, err := FindCredentialPath()
	if err != nil {
		return nil, err
	}

	cfg := &Credentials{}
	if _, err := os.Stat(cfgPath); err == nil {
		// Load exist config file
		_, err := toml.DecodeFile(cfgPath, cfg)
		if err != nil {
			return nil, err
		}
		return cfg, nil
	}

	// Create new config file
	if !os.IsNotExist(err) {
		return nil, err
	}
	f, err := os.Create(cfgPath)
	if err != nil {
		return nil, err
	}
	if err := toml.NewEncoder(f).Encode(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func FindCredentialPath() (string, error) {
	var dir string
	if runtime.GOOS == "windows" {
		dir = os.Getenv("APPDATA")
		if dir == "" {
			dir = filepath.Join(os.Getenv("USERPROFILE"), "Application Data", "mcm")
		}
		dir = filepath.Join(dir, "mcm")
	} else {
		dir = filepath.Join(os.Getenv("HOME"), ".config", "mcm")
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("cannot create directory, %v", err)
	}
	cfgPath := filepath.Join(dir, "credentials.toml")
	return cfgPath, nil
}
