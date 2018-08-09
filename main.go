package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/milanaleksic/amtgo/amt"
	"golang.org/x/crypto/ssh"
)

const localForwardedPort = 16888

func main() {
	username := flag.String("username", "", "Username for the AMT interface")
	password := flag.String("password", "", "Password for the AMT interface")
	// FIXME: should be optional following two, meaning direct access available
	bastionHost := flag.String("bastionHost", "", "Bastion hostname")
	bastionPort := flag.Int("bastionPort", 22, "Bastion port")
	amtHost := flag.String("amtHost", "", "AMT computer hostname")
	amtPort := flag.Int("amtPort", 16992, "AMT computer port")
	flag.Parse()

	localEndpoint := &Endpoint{
		Host: "localhost",
		Port: localForwardedPort,
	}

	bastion := &Endpoint{
		Host: *bastionHost,
		Port: *bastionPort,
	}

	remoteEndpoint := &Endpoint{
		Host: *amtHost,
		Port: *amtPort,
	}

	sshConfig, err := SSHConfigFromAgent()
	if err != nil {
		log.Fatalf("Failed to create ssh configuration: %v", err)
	}

	tunnel := &SSHTunnel{
		Config: sshConfig,
		Local:  localEndpoint,
		Server: bastion,
		Remote: remoteEndpoint,
	}

	_, err = ssh.Dial("tcp", bastion.String(), sshConfig)
	if err != nil {
		fmt.Printf("Server dial error: %s\n", err)
		return
	}

	tunnel.Start()

	// FIXME: correct waiting handshake algo, not this hardcoded one
	time.Sleep(3 * time.Second)

	// FIXME: graceful termination!

	options := amt.Optionset{
		SwSkipcertchk: 1,
		SwUseTLS:      0,
		Username:      *username,
		Password:      *password,
		Port:          localForwardedPort,
	}
	amt.CliCommand(amt.CmdInfo, []string{"localhost"}, options)
}
