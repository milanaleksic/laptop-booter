package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"time"
)

const localForwardedPort = 16888

const (
	// CmdStatus will print the power state and check if main SSH port is there
	CmdStatus = "status"
	// CmdUp will do AMT "powerup" the machine and then decrypt the disk
	// FIXME: NYI - activate
	CmdActivate = "activate"
	// CmdShutdown will execute "shutdown -h now" on the remote system
	// FIXME: NYI - shutdown
	CmdShutdown = "shutdown"
)

func main() {
	username := flag.String("username", "", "Username for the AMT interface")
	password := flag.String("password", "", "Password for the AMT interface")
	// FIXME: should be optional following two, meaning direct access available
	bastionHost := flag.String("bastionHost", "", "Bastion hostname")
	bastionPort := flag.Int("bastionPort", 22, "Bastion port")
	amtHost := flag.String("amtHost", "", "AMT computer hostname")
	amtPort := flag.Int("amtPort", 16992, "AMT computer port")

	command := flag.String("command", "", "Command (one of: status, up, down, decrypt)")
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

	for {
		conn, _ := net.DialTimeout("tcp", net.JoinHostPort("localhost", strconv.Itoa(localForwardedPort)), 10*time.Millisecond)
		if conn != nil {
			conn.Close()
			break
		}
	}

	switch *command {
	case CmdStatus:
		log.Println("Command chosen: show status")
		status := getAmtStatus(*username, *password)
		if status.StateHTTP != 200 {
			log.Printf("Wrong response code from server: %v", status.StateHTTP)
			os.Exit(1)
		}
		fmt.Println(legacyPowerstateTextMap[status.StateAMT])
	case CmdActivate:
		status := getAmtStatus(*username, *password)
		if status.StateHTTP != 200 {
			log.Printf("Wrong response code from server: %v", status.StateHTTP)
			os.Exit(1)
		}
		if status.StateAMT == amtStateOn {
			log.Println("System is already active and on")
			os.Exit(1)
		}
		log.Fatal("NYI!")
	case CmdShutdown:
		status := getAmtStatus(*username, *password)
		if status.StateHTTP != 200 {
			log.Printf("Wrong response code from server: %v", status.StateHTTP)
			os.Exit(1)
		}
		if status.StateAMT == amtStateSoftOff {
			log.Println("System is already turned off")
			os.Exit(1)
		}
		log.Fatal("NYI!")
	default:
		log.Fatalf("Unknown command '%s'", *command)
	}
}
