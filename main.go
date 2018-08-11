package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"

	"golang.org/x/crypto/ssh"
)

const localRealSSHPort = 16887
const localAmtPort = 16888
const localDropbearPort = 16889

const (
	// CmdStatus will print the power state and check if main SSH port is there
	CmdStatus = "status"
	// CmdUp will do AMT "powerup" the machine and then decrypt the disk
	CmdActivate = "activate"
	// CmdShutdown will execute "shutdown -h now" on the remote system
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
	dropbearHost := flag.String("dropbearHost", "", "Dropbear (SSH) computer hostname")
	dropbearPort := flag.Int("dropbearPort", 4748, "Dropbear (SSH) computer port")
	realSSHHost := flag.String("realSSHHost", "", "Real SSH computer hostname")
	realSSHPort := flag.Int("realSSHPort", 22, "Real SSH computer port")
	diskUnlockPassword := flag.String("diskUnlockPassword", "", "Disk unlock password")
	command := flag.String("command", "", "Command (one of: status, up, down, decrypt)")
	flag.Parse()

	bastion := &Endpoint{
		Host: *bastionHost,
		Port: *bastionPort,
	}

	// My use case: always demand ssh agent, no private local files allowed (I use Yubikey)
	sshAgentCloser, sshConfig, err := SSHConfigFromAgent()
	if err != nil {
		log.Fatalf("Failed to create ssh configuration: %v", err)
	}
	defer sshAgentCloser()

	amtTunnel := &SSHTunnel{
		Config: sshConfig,
		Local: &Endpoint{
			Host: "localhost",
			Port: localAmtPort,
		},
		Mediator: bastion,
		Remote: &Endpoint{
			Host: *amtHost,
			Port: *amtPort,
		},
	}
	amtTunnel.Activate()

	dropbearTunnel := &SSHTunnel{
		Config: sshConfig,
		Local: &Endpoint{
			Host: "localhost",
			Port: localDropbearPort,
		},
		Mediator: bastion,
		Remote: &Endpoint{
			Host: *dropbearHost,
			Port: *dropbearPort,
		},
	}
	dropbearTunnel.Activate()

	realSSHTunnel := &SSHTunnel{
		Config: sshConfig,
		Local: &Endpoint{
			Host: "localhost",
			Port: localRealSSHPort,
		},
		Mediator: bastion,
		Remote: &Endpoint{
			Host: *realSSHHost,
			Port: *realSSHPort,
		},
	}
	realSSHTunnel.Activate()

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
		log.Println("Command chosen: activate")
		status := getAmtStatus(*username, *password)
		if status.StateHTTP != 200 {
			log.Printf("Wrong response code from server when fetching status: %v", status.StateHTTP)
			os.Exit(1)
		}
		if status.StateAMT == amtStateOn {
			log.Println("System is already on, ignoring poweron instruction")
		} else {
			log.Println("Activating AMT poweron function")
			setPowerStateOn(*username, *password)
		}

		if singleCheckSSHConnectivityViaLocalPort(localRealSSHPort, getCurrentUser(), sshConfig) {
			log.Println("System's real SSH is already on, ignoring disk decryption voodoo")
		} else {
			log.Println("System's real SSH is not available, reaching out to dropbear to unlock")
			dropbearConn := awaitSSHConnectivityViaLocalPort(localDropbearPort, "root", sshConfig)
			log.Printf("Dropbear active!")
			defer dropbearConn.Close()
			session, err := dropbearConn.NewSession()
			if err != nil {
				log.Fatalf("Failed to create new ssh session: %v", err)
			}
			unlockDisk(diskUnlockPassword, session)
			_ = awaitSSHConnectivityViaLocalPort(localRealSSHPort, getCurrentUser(), sshConfig)
			log.Printf("Real SSH active! Leaving the program")
		}
		os.Exit(0)
	case CmdShutdown:
		log.Println("Command chosen: shutdown")
		status := getAmtStatus(*username, *password)
		if status.StateHTTP != 200 {
			log.Printf("Wrong response code from server when fetching status: %v", status.StateHTTP)
			os.Exit(1)
		}
		if status.StateAMT == amtStateSoftOff {
			log.Println("System is already turned off")
		} else if singleCheckSSHConnectivityViaLocalPort(localRealSSHPort, getCurrentUser(), sshConfig) {
			log.Println("System's real SSH is already on, proceeding with SSH-driven turn off")
			realSSHConn := awaitSSHConnectivityViaLocalPort(localRealSSHPort, getCurrentUser(), sshConfig)
			defer realSSHConn.Close()
			session, err := realSSHConn.NewSession()
			if err != nil {
				log.Fatalf("Failed to create new ssh session: %v", err)
			}
			err = session.Start("sudo shutdown -h now")
			if err != nil {
				log.Fatalf("Shutdown call failed: %v", err)
			}
		} else {
			log.Println("Activating AMT poweroff function")
			setPowerStateOff(*username, *password)
		}
		os.Exit(0)
	default:
		log.Fatalf("Unknown command '%s'", *command)
	}
}

func unlockDisk(diskUnlockPassword *string, session *ssh.Session) {
	var b bytes.Buffer
	b.WriteString(*diskUnlockPassword)
	session.Stdin = &b
	log.Printf("Sending disk unlock password!")
	output, err := session.CombinedOutput("cryptroot-unlock")
	if err != nil {
		log.Fatalf("Unlock call failed: %v", err)
	}
	log.Println(string(output))
}
