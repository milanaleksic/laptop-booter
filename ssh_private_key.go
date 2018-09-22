package laptop_booter

import (
	"time"

	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"
)

func SSHConfigFromPrivateKey(pemBytes []byte) (clientConfig *ssh.ClientConfig, err error) {
	key, err := ssh.ParsePrivateKey(pemBytes)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to parse private key")
	}
	clientConfig := &ssh.ClientConfig{
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(key)},
		Timeout:         5 * time.Second,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	return clientConfig, nil
}
