package laptop_booter

import (
	"bytes"
	"log"

	"github.com/getlantern/errors"
	"golang.org/x/crypto/ssh"
)

type Configuration struct {
	Username           string
	Password           string
	BastionHost        string
	BastionPort        int
	AmtHost            string
	AmtPort            int
	DropbearHost       string
	DropbearPort       int
	RealSSHHost        string
	RealSSHPort        int
	DiskUnlockPassword string
	Command            string
	LocalRealSSHPort   int
	LocalAmtPort       int
	LocalDropbearPort  int
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
	bastion := &Endpoint{
		Host: c.BastionHost,
		Port: c.BastionPort,
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
		Config: sshConfig,
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
		Config: sshConfig,
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
			return "", errors.New("Wrong response code from server: %v", status.StateHTTP)
		}
		return legacyPowerstateTextMap[status.StateAMT], nil
	case CmdActivate:
		log.Println("Command chosen: activate")
		status := getAmtStatus(c.Username, c.Password, c.LocalAmtPort)
		if status.StateHTTP != 200 {
			return "", errors.New("Wrong response code from server when fetching status: %v", status.StateHTTP)
		}
		if status.StateAMT == amtStateOn {
			log.Println("System is already on, ignoring poweron instruction")
		} else {
			log.Println("Activating AMT poweron function")
			setPowerStateOn(c.Username, c.Password, c.LocalAmtPort)
		}

		if singleCheckSSHConnectivityViaLocalPort(c.LocalRealSSHPort, getCurrentUser(), sshConfig) {
			log.Println("System's real SSH is already on, ignoring disk decryption voodoo")
		} else {
			log.Println("System's real SSH is not available, reaching out to dropbear to unlock")
			dropbearConn := awaitSSHConnectivityViaLocalPort(c.LocalDropbearPort, "root", sshConfig)
			log.Printf("Dropbear active!")
			defer dropbearConn.Close()
			session, err := dropbearConn.NewSession()
			if err != nil {
				log.Fatalf("Failed to create new ssh session: %v", err)
			}
			unlockDisk(c.DiskUnlockPassword, session)
			_ = awaitSSHConnectivityViaLocalPort(c.LocalRealSSHPort, getCurrentUser(), sshConfig)
			log.Printf("Real SSH active")
		}
		return "Success", nil
	case CmdShutdown:
		log.Println("Command chosen: shutdown")
		status := getAmtStatus(c.Username, c.Password, c.LocalAmtPort)
		if status.StateHTTP != 200 {
			return "", errors.New("Wrong response code from server when fetching status: %v", status.StateHTTP)
		}
		if status.StateAMT == amtStateSoftOff {
			log.Println("System is already turned off")
		} else if singleCheckSSHConnectivityViaLocalPort(c.LocalRealSSHPort, getCurrentUser(), sshConfig) {
			log.Println("System's real SSH is already on, proceeding with SSH-driven turn off")
			realSSHConn := awaitSSHConnectivityViaLocalPort(c.LocalRealSSHPort, getCurrentUser(), sshConfig)
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
			setPowerStateOff(c.Username, c.Password, c.LocalAmtPort)
		}
		return "Success", nil
	default:
		return "", errors.New("Unknown command '%s'", c.Command)
	}
}

func unlockDisk(diskUnlockPassword string, session *ssh.Session) {
	var b bytes.Buffer
	b.WriteString(diskUnlockPassword)
	session.Stdin = &b
	log.Printf("Sending disk unlock password!")
	output, err := session.CombinedOutput("cryptroot-unlock")
	if err != nil {
		log.Fatalf("Unlock call failed: %v", err)
	}
	log.Println(string(output))
}
