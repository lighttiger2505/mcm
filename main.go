package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Age        int
	Cats       []string
	Pi         float64
	Perfection []int
	DOB        time.Time // requires `import time`
}

func main() {
	tomlData := `Age = 25
Cats = [ "Cauchy", "Plato" ]
Pi = 3.14
Perfection = [ 6, 28, 496, 8128 ]
DOB = 1987-07-05T05:45:00Z`
	var conf Config
	res, err := toml.Decode(tomlData, &conf)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(res)

	// 	res, err := exec.Command("mysql").CombinedOutput()
	// 	if err != nil {
	// 		fmt.Println(err)
	// 	}
	// 	fmt.Println(string(res))
}

func LoadConfig() (*Config, error) {
	// Find config file path
	cfgPath, err := FindConfigPath()
	if err != nil {
		return nil, err
	}

	var cfg Config
	if _, err := os.Stat(cfgPath); err == nil {
		// Load exist config file
		_, err := toml.DecodeFile(cfgPath, cfg)
		if err != nil {
			return nil, err
		}
		return &cfg, nil
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
	return &cfg, nil
}

func FindConfigPath() (string, error) {
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
	cfgPath := filepath.Join(dir, "config.toml")
	return cfgPath, nil
}
