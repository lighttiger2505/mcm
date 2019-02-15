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
	Alias           string      `toml:"alis"`
	Type            ProfileType `toml:"type"`
	DBCmd           string      `toml:"db_cmd"`
	DBHost          string      `toml:"db_host"`
	DBPort          int         `toml:"db_port"`
	DBUser          string      `toml:"db_user"`
	DBPass          string      `toml:"db_pass"`
	DBDefaultSchema string      `toml:"db_default_schema"`
	SSHHost         string      `toml:"ssh_host"`
	SSHPort         int         `toml:"ssh_port"`
	SSHUser         string      `toml:"ssh_user"`
	SSHPass         string      `toml:"ssh_pass"`
	SSHKey          string      `toml:"ssh_key"`
}

type Endpoint struct {
	Host string
	Port int
}

func (endpoint *Endpoint) String() string {
	return fmt.Sprintf("%s:%d", endpoint.Host, endpoint.Port)
}

func (c *Credential) SSHEndpoint() *Endpoint {
	return &Endpoint{
		Host: c.SSHHost,
		Port: c.SSHPort,
	}
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
	return &ssh.ClientConfig{
		User: c.SSHUser,
		Auth: []ssh.AuthMethod{
			publicKeyFile(c.SSHKey),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
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
