package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os/exec"
	"strconv"

	"golang.org/x/crypto/ssh"
)

type ProfileType int

type Credential struct {
	Alias           string       `toml:"alias"`
	DBCmd           string       `toml:"cmd"`
	DBHost          string       `toml:"host"`
	DBPort          int          `toml:"port"`
	DBUser          string       `toml:"user"`
	DBPass          string       `toml:"pass"`
	DBDefaultSchema string       `toml:"default_schema"`
	TunelCfg        *TunelConfig `toml:"tunel_config"`
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
		Host: c.DBHost,
		Port: c.DBPort,
	}
}

func (c *Credential) LocalEndpoint() *Endpoint {
	return &Endpoint{
		Host: "127.0.0.1",
		Port: 0,
	}
}

func (c *Credential) SSHClientConfig() *ssh.ClientConfig {
	return c.SSHClientConfig()
}

func (c *Credential) MySQLCommand() *exec.Cmd {
	return exec.Command(
		c.DBCmd,
		"-h",
		c.DBHost,
		"-P",
		strconv.Itoa(c.DBPort),
		"-u",
		c.DBUser,
		"-D",
		c.DBDefaultSchema,
		fmt.Sprintf("-p%s", c.DBPass),
	)
}

func (c *Credential) MySQLTunnelCommand(port string) *exec.Cmd {
	return exec.Command(
		c.DBCmd,
		"-h",
		"127.0.0.1",
		"-P",
		port,
		"-u",
		c.DBUser,
		"-D",
		c.DBDefaultSchema,
		fmt.Sprintf("-p%s", c.DBPass),
	)
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
			publicKeyFile(c.Key),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
}

func publicKeyFile(file string) ssh.AuthMethod {
	buffer, err := ioutil.ReadFile(file)
	if err != nil {
		log.Fatalln(fmt.Sprintf("Cannot read SSH public key file %s", file))
		return nil
	}

	key, err := ssh.ParsePrivateKey(buffer)
	if err != nil {
		log.Fatalln(fmt.Sprintf("Cannot parse SSH public key file %s", file))
		return nil
	}
	return ssh.PublicKeys(key)
}
