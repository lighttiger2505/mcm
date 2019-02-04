package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"golang.org/x/crypto/ssh"
)

func main() {
	cfg, err := LoadConfig()
	if err != nil {
		log.Fatalln("failed load config, ", err)
	}

	if err := TunnelConnect(cfg.Profiles[0]); err != nil {
		log.Fatalln("failed connect, ", err)
		os.Exit(1)
	}
	os.Exit(0)
}

type ProfileType int

type Profile struct {
	Alias           string      `toml:"alis"`
	Type            ProfileType `toml:"type"`
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

func (c *Profile) SSHEndpoint() *Endpoint {
	return &Endpoint{
		Host: c.SSHHost,
		Port: c.SSHPort,
	}
}

func (c *Profile) DBEndpoint() *Endpoint {
	return &Endpoint{
		Host: c.DBHost,
		Port: c.DBPort,
	}
}

func (c *Profile) LocalEndpoint() *Endpoint {
	return &Endpoint{
		Host: "127.0.0.1",
		Port: 0,
	}
}

func (c *Profile) SSHClientConfig() *ssh.ClientConfig {
	return &ssh.ClientConfig{
		User: c.SSHUser,
		Auth: []ssh.AuthMethod{
			publicKeyFile(c.SSHKey),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
}

func (c *Profile) MySQLCommand() *exec.Cmd {
	return exec.Command(
		"mysql",
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

func (c *Profile) MySQLTunnelCommand(port string) *exec.Cmd {
	return exec.Command(
		"mysql",
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

func StanderdConnect(cmd *exec.Cmd) error {
	c := cmd
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	if err := c.Run(); err != nil {
		return fmt.Errorf("command error, %s\n", err)
	}
	return nil
}

func TunnelConnect(profile *Profile) error {
	var serverConn *ssh.Client
	listener, err := net.Listen("tcp", profile.LocalEndpoint().String())
	if err != nil {
		log.Fatalln("Local listen error, ", err)
	}
	defer listener.Close()

	type Q struct{}
	var Ready = make(chan Q, 1)
	var Done = make(chan error)
	go func() {
		for {
			Ready <- Q{}
			localConn, err := listener.Accept()
			if err != nil {
				Done <- fmt.Errorf("Listener Accept error, %s\n", err)
				return
			}

			if serverConn == nil {
				serverConn, err = ssh.Dial("tcp", profile.SSHEndpoint().String(), profile.SSHClientConfig())
				if err != nil {
					Done <- fmt.Errorf("Server dial error, %s\n", err)
					return
				}
			}

			if err := forward(localConn, serverConn, profile.DBEndpoint()); err != nil {
				Done <- err
				return
			}
		}
	}()

	select {
	case <-Ready:
		lstr := strings.Split(listener.Addr().String(), ":")
		c := profile.MySQLTunnelCommand(lstr[1])
		c.Stdin = os.Stdin
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		if err := c.Run(); err != nil {
			return fmt.Errorf("command error, %s\n", err)
		}
	case err = <-Done:
		if serverConn != nil {
			serverConn.Conn.Close()
		}
		return err
	}
	return nil
}

func forward(localConn net.Conn, serverConn *ssh.Client, dbserver *Endpoint) error {
	remoteConn, err := serverConn.Dial("tcp", dbserver.String())
	if err != nil {
		return fmt.Errorf("Remote dial error: %s\n", err)
	}

	copyConn := func(writer, reader net.Conn) {
		_, err := io.Copy(writer, reader)
		if err != nil {
			log.Printf("io.Copy error, %s\n", err)
		}
	}

	go copyConn(localConn, remoteConn)
	go copyConn(remoteConn, localConn)
	return nil
}
