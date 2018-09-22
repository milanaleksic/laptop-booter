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
	serverConn, err := ssh.Dial("tcp", tunnel.Mediator.String(), tunnel.Config)
	if err != nil {
		fmt.Printf("Server dial error: %s\n", err)
		return
	}

	remoteConn, err := serverConn.Dial("tcp", tunnel.Remote.String())
	if err != nil {
		fmt.Printf("Remote dial error: %+v\n", err)
		return
	}

	copyConn := func(writer, reader net.Conn) {
		_, err := io.Copy(writer, reader)
		if err != nil {
			fmt.Printf("io.Copy error: %s", err)
		}
	}

	go copyConn(localConn, remoteConn)
	go copyConn(remoteConn, localConn)
}

func (tunnel *SSHTunnel) Activate() {
	log.Printf("Activating local port %v for tunnel to %+v", tunnel.Local.Port, tunnel.Remote)
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
