package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"bytes"

	"golang.org/x/crypto/ssh"
)

const localAmtPort = 16888
const localDropbearPort = 16889

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
	dropbearHost := flag.String("dropbearHost", "", "Dropbear (SSH) computer hostname")
	dropbearPort := flag.Int("dropbearPort", 4748, "Dropbear (SSH) computer port")
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
		//status := getAmtStatus(*username, *password)
		//if status.StateHTTP != 200 {
		//	log.Printf("Wrong response code from server: %v", status.StateHTTP)
		//	os.Exit(1)
		//}
		//if status.StateAMT == amtStateOn {
		//	log.Println("System is already on, ignoring poweron instruction")
		//} else {
		//	log.Println("Activating AMT poweron function")
		//	setPowerStateOn(*username, *password)
		//}

		// waiting for the dropbear to connect
		sshConfigForDropbear := *sshConfig
		sshConfigForDropbear.User = "root"
		var serverConn *ssh.Client
		for {
			serverConn, err = ssh.Dial("tcp", "localhost:"+strconv.Itoa(localDropbearPort), &sshConfigForDropbear)
			if err != nil {
				log.Printf("Dropbear still not ready: %s", err)
				time.Sleep(1 * time.Second)
			} else {
				break
			}
		}
		log.Printf("Dropbear active!")
		defer serverConn.Close()
		session, err := serverConn.NewSession()
		if err != nil {
			log.Fatalf("Failed to create new ssh session: %v", err)
		}
		var b bytes.Buffer
		b.WriteString(*diskUnlockPassword)
		session.Stdin = &b
		log.Printf("Sending disk unlock password!")
		output, err := session.Output("cryptroot-unlock")
		if err != nil {
			log.Fatalf("Unlock call failed: %v", err)
		}
		log.Println(string(output))
		// TODO: await real SSH port open and leave program with success
	case CmdShutdown:
		log.Println("Command chosen: shutdown")
		status := getAmtStatus(*username, *password)
		if status.StateHTTP != 200 {
			log.Printf("Wrong response code from server: %v", status.StateHTTP)
			os.Exit(1)
		}
		if status.StateAMT == amtStateSoftOff {
			log.Println("System is already turned off")
			os.Exit(0)
		} else {
			log.Println("Activating AMT poweroff function")
			// FIXME: if dropbear ssh port is active, turn off via amt
			setPowerStateOff(*username, *password)
			// FIXME: if main ssh port is active, turn off via "shutdown -h now" ssh command
		}
		log.Fatal("NYI!")
	default:
		log.Fatalf("Unknown command '%s'", *command)
	}
}
