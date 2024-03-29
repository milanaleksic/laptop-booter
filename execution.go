package laptop_booter

import (
	"bytes"
	"fmt"
	"log"

	"golang.org/x/crypto/ssh"
)

type Configuration struct {
	Username           string
	Password           string
	BastionUsername    string
	BastionHost        string
	BastionPort        int
	AmtHost            string
	AmtPort            int
	DropbearUsername   string
	DropbearHost       string
	DropbearPort       int
	RealSSHUsername    string
	RealSSHHost        string
	RealSSHPort        int
	DiskUnlockPassword string
	Command            string
	LocalRealSSHPort   int
	LocalAmtPort       int
	LocalDropbearPort  int
	AgentConfiguration *ssh.ClientConfig
}

const (
	// CmdStatus will print the power state and check if main SSH port is there
	CmdStatus = "status"
	// CmdUp will do AMT "powerup" the machine and then decrypt the disk
	CmdActivate = "activate"
	// CmdShutdown will execute "shutdown -h now" on the remote system
	CmdShutdown = "shutdown"
)

func Execute(c *Configuration) (output string, err error) {
	if c.AgentConfiguration == nil {
		sshAgentCloser, sshConfig, err := SSHConfigFromAgent(c.BastionUsername)
		if err != nil {
			return "", fmt.Errorf("failed to create ssh configuration: %w", err)
		}
		defer sshAgentCloser()
		c.AgentConfiguration = sshConfig
	}
	bastion := &Endpoint{
		Host: c.BastionHost,
		Port: c.BastionPort,
	}

	amtTunnel := &SSHTunnel{
		Config: c.AgentConfiguration,
		Local: &Endpoint{
			Host: "localhost",
			Port: c.LocalAmtPort,
		},
		Mediator: bastion,
		Remote: &Endpoint{
			Host: c.AmtHost,
			Port: c.AmtPort,
		},
	}
	amtTunnel.Activate()

	dropbearTunnel := &SSHTunnel{
		Config: c.AgentConfiguration,
		Local: &Endpoint{
			Host: "localhost",
			Port: c.LocalDropbearPort,
		},
		Mediator: bastion,
		Remote: &Endpoint{
			Host: c.DropbearHost,
			Port: c.DropbearPort,
		},
	}
	dropbearTunnel.Activate()

	realSSHTunnel := &SSHTunnel{
		Config: c.AgentConfiguration,
		Local: &Endpoint{
			Host: "localhost",
			Port: c.LocalRealSSHPort,
		},
		Mediator: bastion,
		Remote: &Endpoint{
			Host: c.RealSSHHost,
			Port: c.RealSSHPort,
		},
	}
	realSSHTunnel.Activate()

	switch c.Command {
	case CmdStatus:
		log.Println("Command chosen: show status")
		status := getAmtStatus(c.Username, c.Password, c.LocalAmtPort)
		if status.StateHTTP != 200 {
			return "", fmt.Errorf("Wrong response code from server: %w", status.StateHTTP)
		}
		return legacyPowerstateTextMap[status.StateAMT], nil
	case CmdActivate:
		log.Println("Command chosen: activate")
		status := getAmtStatus(c.Username, c.Password, c.LocalAmtPort)
		if status.StateHTTP != 200 {
			return "", fmt.Errorf("Wrong response code from server when fetching status: %v", status.StateHTTP)
		}
		if status.StateAMT == amtStateOn {
			log.Println("System is already on, ignoring poweron instruction")
		} else {
			log.Println("Activating AMT poweron function")
			setPowerStateOn(c.Username, c.Password, c.LocalAmtPort)
		}

		if singleCheckSSHConnectivityViaLocalPort(c.LocalRealSSHPort, c.RealSSHUsername, c.AgentConfiguration) {
			log.Println("System's real SSH is already on, ignoring disk decryption voodoo")
		} else {
			log.Println("System's real SSH is not available, reaching out to dropbear to unlock")
			dropbearConn, err := awaitSSHConnectivityViaLocalPort(c.LocalDropbearPort, "root", c.AgentConfiguration)
			if err != nil {
				return "", fmt.Errorf("Dropbear connection could not be established!: %w", err)
			}
			defer dropbearConn.Close()
			log.Printf("Dropbear connection established!")
			session, err := dropbearConn.NewSession()
			if err != nil {
				return "", fmt.Errorf("Failed to create new ssh session: %w", err)
			}
			err = unlockDisk(c.DiskUnlockPassword, session)
			if err != nil {
				return "", fmt.Errorf("Failed to unlock the disk: %w", err)
			}
			_, err = awaitSSHConnectivityViaLocalPort(c.LocalRealSSHPort, c.RealSSHUsername, c.AgentConfiguration)
			if err != nil {
				return "", fmt.Errorf("Failed to establish error to real SSH: %w", err)
			}
			log.Printf("Real SSH active")
		}
		return "Success", nil
	case CmdShutdown:
		log.Println("Command chosen: shutdown")
		status := getAmtStatus(c.Username, c.Password, c.LocalAmtPort)
		if status.StateHTTP != 200 {
			return "", fmt.Errorf("Wrong response code from server when fetching status: %v", status.StateHTTP)
		}
		if status.StateAMT == amtStateSoftOff {
			log.Println("System is already turned off")
		} else if singleCheckSSHConnectivityViaLocalPort(c.LocalRealSSHPort, c.RealSSHUsername, c.AgentConfiguration) {
			log.Println("System's real SSH is already on, proceeding with SSH-driven turn off")
			realSSHConn, err := awaitSSHConnectivityViaLocalPort(c.LocalRealSSHPort, c.RealSSHUsername, c.AgentConfiguration)
			if err != nil {
				return "", fmt.Errorf("Failed to establish error to real SSH: %w", err)
			}
			defer realSSHConn.Close()
			session, err := realSSHConn.NewSession()
			if err != nil {
				return "", fmt.Errorf("Failed to create new ssh session: %w", err)
			}
			err = session.Start("sudo shutdown -h now")
			if err != nil {
				return "", fmt.Errorf("Shutdown call failed: %w", err)
			}
		} else {
			log.Println("Activating AMT poweroff function")
			setPowerStateOff(c.Username, c.Password, c.LocalAmtPort)
		}
		return "Success", nil
	default:
		return "", fmt.Errorf("Unknown command '%s'", c.Command)
	}
}

func unlockDisk(diskUnlockPassword string, session *ssh.Session) error {
	var b bytes.Buffer
	b.WriteString(diskUnlockPassword)
	session.Stdin = &b
	log.Printf("Sending disk unlock password!")
	output, err := session.CombinedOutput("cryptroot-unlock")
	if err != nil {
		return fmt.Errorf("Unlock call failed: %w", err)
	}
	log.Println(string(output))
	return nil
}
