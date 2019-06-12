package main

import (
	"fmt"
	"io/ioutil"
	"strings"

	"log"
	"os/exec"
	"os/user"
	"strconv"

	"golang.org/x/crypto/ssh"
)

type ProfileType int

type Credential struct {
	Alias         string       `toml:"alias"`
	Cmd           string       `toml:"cmd"`
	Host          string       `toml:"host"`
	Port          int          `toml:"port"`
	User          string       `toml:"user"`
	Pass          string       `toml:"pass"`
	DefaultSchema string       `toml:"default_schema"`
	TunelCfg      *TunelConfig `toml:"tunel_config"`
}

type Endpoint struct {
	Host string
	Port int
}

func (endpoint *Endpoint) String() string {
	return fmt.Sprintf("%s:%d", endpoint.Host, endpoint.Port)
}

func (c *Credential) SSHEndpoint() *Endpoint {
	return c.TunelCfg.Endpoint()
}

func (c *Credential) DBEndpoint() *Endpoint {
	return &Endpoint{
		Host: c.Host,
		Port: c.Port,
	}
}

func (c *Credential) LocalEndpoint() *Endpoint {
	return &Endpoint{
		Host: "127.0.0.1",
		Port: 0,
	}
}

func (c *Credential) SSHClientConfig() *ssh.ClientConfig {
	return c.TunelCfg.SSHClientConfig()
}

func (c *Credential) MySQLCommand() *exec.Cmd {
	args := []string{
		"-h",
		c.Host,
		"-u",
		c.User,
		fmt.Sprintf("-p%s", c.Pass),
	}
	if c.Port != 0 {
		args = append(args, []string{"-P", strconv.Itoa(c.Port)}...)
	}
	if c.DefaultSchema != "" {
		args = append(args, []string{"-D", c.DefaultSchema}...)
	}
	return exec.Command(c.Cmd, args...)
}

func (c *Credential) MySQLTunnelCommand(port string) *exec.Cmd {
	args := []string{
		"-h",
		"127.0.0.1",
		"-u",
		c.User,
		fmt.Sprintf("-p%s", c.Pass),
	}
	if c.Port != 0 {
		args = append(args, []string{"-P", port}...)
	}
	if c.DefaultSchema != "" {
		args = append(args, []string{"-D", c.DefaultSchema}...)
	}
	return exec.Command(c.Cmd, args...)
}

type TunelConfig struct {
	Host string `toml:"host"`
	Port int    `toml:"port"`
	User string `toml:"user"`
	Pass string `toml:"pass"`
	Key  string `toml:"key"`
}

func (c *TunelConfig) Endpoint() *Endpoint {
	return &Endpoint{
		Host: c.Host,
		Port: c.Port,
	}
}

func (c *TunelConfig) SSHClientConfig() *ssh.ClientConfig {
	return &ssh.ClientConfig{
		User: c.User,
		Auth: []ssh.AuthMethod{
			publicKeyFile(c.Key, c.Pass),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
}

func publicKeyFile(file, passPhrase string) ssh.AuthMethod {
	usr, _ := user.Current()
	file = strings.Replace(file, "~", usr.HomeDir, 1)
	buffer, err := ioutil.ReadFile(file)
	if err != nil {
		log.Fatalln(fmt.Sprintf("Cannot read SSH public key file %s.\n%s", file, err))
		return nil
	}

	var key ssh.Signer
	if passPhrase != "" {
		key, err = ssh.ParsePrivateKeyWithPassphrase(buffer, []byte(passPhrase))
		if err != nil {
			log.Fatalln(fmt.Sprintf("Cannot parse SSH public key file %s.\n%s", file, err))
			return nil
		}
	} else {
		key, err = ssh.ParsePrivateKey(buffer)
		if err != nil {
			log.Fatalln(fmt.Sprintf("Cannot parse SSH public key file %s.\n%s", file, err))
			return nil
		}
	}
	return ssh.PublicKeys(key)
}
