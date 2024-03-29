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
	Socket        string       `toml:"socket"`
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

func (c *Credential) MySQLCommandArgs() []string {
	args := []string{
		"-u",
		c.User,
	}
	if c.Pass != "" {
		args = append(args, fmt.Sprintf("-p%s", c.Pass))
	}
	if c.Host != "" {
		args = append(args, []string{"-h", c.Host}...)
	}
	if c.Socket != "" {
		args = append(args, []string{"-S", c.Socket}...)
	}
	if c.Port != 0 {
		args = append(args, []string{"-P", strconv.Itoa(c.Port)}...)
	}
	if c.DefaultSchema != "" {
		args = append(args, []string{"-D", c.DefaultSchema}...)
	}
	return args
}

func (c *Credential) MySQLCommand() *exec.Cmd {
	return exec.Command(c.Cmd, c.MySQLCommandArgs()...)
}

func (c *Credential) MySQLCommandString() string {
	args := []string{c.Cmd}
	args = append(args, c.MySQLCommandArgs()...)
	return strings.Join(args, " ")
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

func (c *Credential) PostgreSQLCommandArgs() string {
	host := "127.0.0.1"
	if c.Host != "" {
		host = c.Host
	}
	port := 15432
	if c.Port != 0 {
		port = c.Port
	}
	arg := fmt.Sprintf(
		"postgresql://%s:%d/%s?user=%s&password=%s",
		host,
		port,
		c.DefaultSchema,
		c.User,
		c.Pass,
	)
	return arg
}

func (c *Credential) PostgreSQLCommand() *exec.Cmd {
	return exec.Command(c.Cmd, c.PostgreSQLCommandArgs())
}

func (c *Credential) PostgreSQLCommandString() string {
	args := []string{c.Cmd}
	args = append(args, c.PostgreSQLCommandArgs())
	return strings.Join(args, " ")
}

func (c *Credential) PostgreSQLTunnelCommand(port string) *exec.Cmd {
	host := "127.0.0.1"
	arg := fmt.Sprintf(
		"postgresql://%s:%s/%s?user=%s&password=%s",
		host,
		port,
		c.DefaultSchema,
		c.User,
		c.Pass,
	)
	return exec.Command(c.Cmd, arg)
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
		log.Fatalln(fmt.Sprintf("cannot read SSH private key file %s, %s", file, err))
		return nil
	}

	var key ssh.Signer
	if passPhrase != "" {
		key, err = ssh.ParsePrivateKeyWithPassphrase(buffer, []byte(passPhrase))
		if err != nil {
			log.Fatalln(fmt.Sprintf("cannot parse SSH private key file with passphrase, %s, %s", file, err))
			return nil
		}
	} else {
		key, err = ssh.ParsePrivateKey(buffer)
		if err != nil {
			log.Fatalln(fmt.Sprintf("cannot parse SSH private key file, %s, %s", file, err))
			return nil
		}
	}
	return ssh.PublicKeys(key)
}
