package main

import (
	"flag"
	"log"

	"github.com/milanaleksic/amtgo/amt"
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

	closer, sshConfig, err := SSHConfigFromAgent()
	if err != nil {
		log.Fatalf("Failed to create ssh configuration: %v", err)
	}
	defer closer()

	tunnel := &SSHTunnel{
		Config: sshConfig,
		Local:  localEndpoint,
		Server: bastion,
		Remote: remoteEndpoint,
	}

	go tunnel.Start()

	options := amt.Optionset{
		SwSkipcertchk: 1,
		SwUseTLS:      0,
		Username:      *username,
		Password:      *password,
		Port:          localForwardedPort,
	}
	amt.CliCommand(amt.CmdInfo, []string{"localhost"}, options)
}
