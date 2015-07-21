package main

import (
	"../../remoton"
	"../../xpra"
	log "github.com/Sirupsen/logrus"
	"io"
	"net"
	"strconv"
	"strings"
	"time"
)

type chatRemoton struct {
	onRecv func(msg string)
	chRecv chan string
	conn   net.Conn
}

func (c *chatRemoton) init() {
	if c.chRecv == nil {
		c.chRecv = make(chan string)
	}
}

func (c *chatRemoton) Start(session *remoton.SessionClient) error {
	chatConn, err := session.Dial("chat")
	if err != nil {
		return err
	}
	c.conn = chatConn
	c.init()
	go c.handle()
	return nil
}

func (c *chatRemoton) handle() {

	for {
		buf := make([]byte, 32*512)
		log.Println("waiting")
		rlen, err := c.conn.Read(buf)
		if err != nil {
			log.Error(err)
			break
		}
		print(buf)
		if c.onRecv != nil {
			c.onRecv(strings.TrimSpace(string(buf[0:rlen])))
		}
	}
}

func (c *chatRemoton) Send(msg string) {
	if c.conn != nil {
		c.conn.Write([]byte(msg))
	}
}

func (c *chatRemoton) OnRecv(f func(msg string)) {
	c.onRecv = f
}

func (c *chatRemoton) Terminate() {
	if c.conn != nil {
		c.conn.Close()
	}
}

type tunnelRemoton struct {
	listener net.Listener
}

func (c *tunnelRemoton) Start(session *remoton.SessionClient) error {
	port := c.findFreePort()
	addrSrv := "localhost:" + port
	log.Println("listen at " + addrSrv)
	listener, err := net.Listen("tcp", addrSrv)
	if err != nil {
		return err
	}

	remote, err := session.Dial("nx")
	if err != nil {
		listener.Close()
		return err
	}

	chConn := make(chan net.Conn)
	errc := make(chan error)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			listener.Close()
			errc <- err
		}
		chConn <- conn
	}()

	err = xpra.Attach(addrSrv)
	if err != nil {
		listener.Close()
		return err
	}
	c.listener = listener
	go c.handle(<-chConn, remote)
	return nil
}

func (c *tunnelRemoton) handle(local, remoto net.Conn) {

	errc := make(chan error, 2)

	go func() {
		_, err := io.Copy(local, remoto)
		errc <- err
	}()
	go func() {
		_, err := remoton.NetCopy(remoto, local, time.Second*5)
		errc <- err
	}()

	log.Error(<-errc)
}

func (c *tunnelRemoton) findFreePort() string {
	startPort := 55123

	for ; startPort < 65534; startPort++ {
		conn, err := net.Dial("tcp", "localhost:"+strconv.Itoa(startPort))
		if err != nil {
			return strconv.Itoa(startPort)
		}
		conn.Close()
	}
	panic("cant find free port")
	return ""
}

func (c *tunnelRemoton) Terminate() {
	if c.listener != nil {
		c.listener.Close()
	}
	xpra.Terminate()
}
