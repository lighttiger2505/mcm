package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"strings"

	"github.com/urfave/cli"
	"golang.org/x/crypto/ssh"
)

func main() {
	err := newApp().Run(os.Args)
	var exitCode = 0
	if err != nil {
		fmt.Fprint(os.Stderr, err.Error())
		exitCode = 255
	}
	os.Exit(exitCode)
}

func newApp() *cli.App {
	app := cli.NewApp()
	app.Name = "mcm"
	app.HelpName = "mcm"
	app.Usage = "cli tool for mysql connection management."
	// app.UsageText = "liary [options] [write content for diary]"
	app.Version = "0.0.1"
	app.Author = "lighttiger2505"
	app.Email = "lighttiger2505@gmail.com"
	app.Commands = []cli.Command{
		{
			Name:    "connect",
			Aliases: []string{"n"},
			Usage:   "connect DB",
			Action:  connect,
		},
		{
			Name:    "list",
			Aliases: []string{"l"},
			Usage:   "list DB credentials",
			Action:  list,
		},
		{
			Name:    "cred",
			Aliases: []string{"r"},
			Usage:   "modifi DB credential",
			Action:  cred,
		},
	}
	return app
}

func cred(c *cli.Context) error {
	credPath, err := FindCredentialPath()
	if err != nil {
		return err
	}

	editorEnv := os.Getenv("EDITOR")
	if editorEnv == "" {
		editorEnv = "vim"
	}

	cmd := exec.Command(editorEnv, credPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func list(c *cli.Context) error {
	creds, err := LoadCredentials()
	if err != nil {
		return fmt.Errorf("failed load credentials, %s", err)
	}

	for _, cred := range creds.Credentials {
		fmt.Fprintln(os.Stdout, cred.Alias)
	}

	return nil
}

func connect(c *cli.Context) error {
	args := c.Args()
	if len(args) == 0 {
		return errors.New("required arguments were not provided: <alias>")
	}

	creds, err := LoadCredentials()
	if err != nil {
		return err
	}

	cred, err := creds.GetCredential(args[0])
	if err != nil {
		return err
	}

	if cred.TunelCfg != nil {
		if err := TunnelConnect(cred); err != nil {
			return err
		}
	} else {
		if err := StanderdConnect(cred); err != nil {
			return err
		}
	}

	return nil
}

func StanderdConnect(profile *Credential) error {
	var c *exec.Cmd
	switch profile.Cmd {
	case "mysql", "mycli":
		c = profile.MySQLCommand()
	case "psql", "pgcli":
		c = profile.PostgreSQLCommand()
	default:
		return fmt.Errorf("invalid command, %s", profile.Cmd)
	}
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	if err := c.Run(); err != nil {
		return fmt.Errorf("command error, %s", err)
	}
	return nil
}

func TunnelConnect(profile *Credential) error {
	var serverConn *ssh.Client
	listener, err := net.Listen("tcp", profile.LocalEndpoint().String())
	if err != nil {
		return fmt.Errorf("cannot listen local connection, %s", err)
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
				Done <- fmt.Errorf("cannot accept local listener, %s", err)
				return
			}

			if serverConn == nil {
				serverConn, err = ssh.Dial("tcp", profile.SSHEndpoint().String(), profile.SSHClientConfig())
				if err != nil {
					Done <- fmt.Errorf("cannot dial database server, %s", err)
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
		port := lstr[1]

		var c *exec.Cmd
		switch profile.Cmd {
		case "mysql", "mycli":
			c = profile.MySQLTunnelCommand(port)
		case "psql", "pgcli":
			c = profile.PostgreSQLTunnelCommand(port)
		default:
			return fmt.Errorf("invalid command, %s", profile.Cmd)
		}
		c.Stdin = os.Stdin
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		if err := c.Run(); err != nil {
			return fmt.Errorf("failed execute command, %s", err)
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
		return fmt.Errorf("cannot dial remote connection, %s", err)
	}

	copyConn := func(writer, reader net.Conn) {
		_, err := io.Copy(writer, reader)
		if err != nil {
			log.Printf("cannot copy io stream on the ssh tunnel, %s", err)
		}
	}

	go copyConn(localConn, remoteConn)
	go copyConn(remoteConn, localConn)
	return nil
}
