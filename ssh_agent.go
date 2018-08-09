package main

import (
	"net"
	"os"
	"time"

	"log"

	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

func SSHConfigFromAgent() (client *ssh.ClientConfig, err error) {
	var signers []ssh.Signer
	agentConn, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK"))
	if err != nil {
		return nil, errors.WithMessage(err, "cannot connect to ssh-agent, check if SSH_AUTH_SOCK is set")
	}
	defer agentConn.Close()
	sshAgent := agent.NewClient(agentConn)
	signers, err = sshAgent.Signers()
	log.Printf("Signers available in the SSH Agent: %+v, user: %v", len(signers), os.Getenv("USER"))
	if err != nil {
		return nil, err
	}
	return &ssh.ClientConfig{
		User:            os.Getenv("USER"),
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signers...)},
		Timeout:         30 * time.Second,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}, nil
}
