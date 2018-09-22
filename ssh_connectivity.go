package laptop_booter

import (
	"log"
	"strconv"
	"time"

	"golang.org/x/crypto/ssh"
)

// singleCheckSSHConnectivityViaLocalPort is a function that will verify that a port is open. We can't depend on the
// timeout setting of the ssh client configuration since that is used just to initiate SSH tunnel communication,
// and doesn't verify someone is listening on the other side, thus we need to decide to kill it after the timeout
func singleCheckSSHConnectivityViaLocalPort(port int, user string, sshConfig *ssh.ClientConfig) bool {
	response := make(chan bool)
	go func() {
		var err error
		sshConfigAdapted := *sshConfig
		sshConfigAdapted.User = user
		var serverConn *ssh.Client
		serverConn, err = ssh.Dial("tcp", "localhost:"+strconv.Itoa(port), &sshConfigAdapted)
		if err != nil {
			log.Println("Unavailable SSH connectivity: ", port)
			response <- false
		}
		defer serverConn.Close()
		response <- true
	}()

	select {
	case res := <-response:
		return res
	case <-time.After(sshConfig.Timeout):
		return false
	}
}

func awaitSSHConnectivityViaLocalPort(port int, user string, sshConfig *ssh.ClientConfig) *ssh.Client {
	for {
		response := make(chan *ssh.Client)
		go func() {
			var err error
			sshConfigAdapted := *sshConfig
			sshConfigAdapted.User = user
			var serverConn *ssh.Client
			serverConn, err = ssh.Dial("tcp", "localhost:"+strconv.Itoa(port), &sshConfigAdapted)
			if err != nil {
				log.Println("Unavailable SSH connectivity: ", port)
				response <- nil
			}
			response <- serverConn
		}()

		select {
		case res := <-response:
			return res
		case <-time.After(sshConfig.Timeout):
			log.Printf("Failed to establish the SSH connection to %d in reasonable time", port)
			continue
		}
	}
}
