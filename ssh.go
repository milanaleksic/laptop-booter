package laptop_booter

// based on http://blog.ralch.com/tutorial/golang-ssh-tunneling/

import (
	"fmt"
	"io"
	"net"
	"strconv"
	"time"

	"log"

	"golang.org/x/crypto/ssh"
)

type Endpoint struct {
	// Server host address
	Host string
	// Server port
	Port int
}

func (endpoint *Endpoint) String() string {
	return fmt.Sprintf("%s:%d", endpoint.Host, endpoint.Port)
}

func (endpoint *Endpoint) IsSet() bool {
	return endpoint.Host != ""
}

type SSHTunnel struct {
	Local    *Endpoint
	Mediator *Endpoint
	Remote   *Endpoint

	Config *ssh.ClientConfig
}

func (tunnel *SSHTunnel) BlockingListen() error {
	listener, err := net.Listen("tcp", tunnel.Local.String())
	if err != nil {
		return err
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			return err
		}
		go tunnel.forward(conn)
	}
}

func (tunnel *SSHTunnel) forward(localConn net.Conn) {
	var remoteConn net.Conn
	var err error
	if tunnel.Mediator.IsSet() {
		serverConn, err := ssh.Dial("tcp", tunnel.Mediator.String(), tunnel.Config)
		if err != nil {
			fmt.Printf("Server dial error to %s:%d, %s\n", tunnel.Remote.Host, tunnel.Remote.Port, err)
			return
		}

		remoteConn, err = serverConn.Dial("tcp", tunnel.Remote.String())
	} else {
		remoteConn, err = net.Dial("tcp", tunnel.Remote.String())
	}

	if err != nil {
		fmt.Printf("Remote dial error to %s:%d, %+v\n", tunnel.Remote.Host, tunnel.Remote.Port, err)
		return
	}

	copyConn := func(writer, reader net.Conn) {
		_, err := io.Copy(writer, reader)
		if err != nil {
			fmt.Printf("io.Copy error to %s:%d, %s", tunnel.Remote.Host, tunnel.Remote.Port, err)
		}
	}

	go copyConn(localConn, remoteConn)
	go copyConn(remoteConn, localConn)
}

func (tunnel *SSHTunnel) Activate() {
	log.Printf("Activating local port %v for tunnel to %v (user: %s)", tunnel.Local.Port, tunnel.Remote, tunnel.Config.User)
	go tunnel.BlockingListen()
	tunnel.waitForLocalHostOpen()
}

func (tunnel *SSHTunnel) waitForLocalHostOpen() {
	for {
		conn, _ := net.DialTimeout("tcp", net.JoinHostPort("localhost", strconv.Itoa(tunnel.Local.Port)), 100*time.Millisecond)
		if conn != nil {
			conn.Close()
			break
		}
	}
}
