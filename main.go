// goredsocks project main.go
package main

import (
	"flag"
	"fmt"
	"golang.org/x/net/proxy"
	"io"
	"log"
	"net"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var (
	proxyDialer proxy.Dialer
	running     bool
	bind        string
	relay       string
	ioHack      bool
	pidfile     string
	debugMode   bool
)

func customCopy(dst io.Writer, src io.Reader, buf []byte) (err error) {
	for {
		nr, er := src.Read(buf)
		if nr > 0 {
			_, ew := dst.Write(buf[0:nr])
			if ew != nil {
				err = ew
				break
			}
		}
		if er != nil {
			if er != io.EOF {
				err = er
			}
			break
		}
	}
	return err
}

func PrintfLog(format string, v ...interface{}) {
	if debugMode {
		fmt.Println(format, v)
	}
}

func handleConnection(conn *net.TCPConn) {
	conn.SetKeepAlive(true)
	conn.SetKeepAlivePeriod(20 * time.Second)
	conn.SetNoDelay(false)

	dstAddr, err := GetOriginalDST(conn)
	if err != nil {
		PrintfLog("[ERROR] GetOriginalDST error: %s\n", err.Error())
		return
	}
	log.Printf("[INFO] Connect to %s.\n", dstAddr.String())
	proxy, err := proxyDialer.Dial("tcp", dstAddr.String())
	if err != nil {
		PrintfLog("[ERROR] Dial proxy error: %s.\n", err.Error())
		return
	}
	proxy.(*net.TCPConn).SetKeepAlive(true)
	proxy.(*net.TCPConn).SetKeepAlivePeriod(20 * time.Second)
	proxy.(*net.TCPConn).SetNoDelay(false)

	copyEnd := false

	go func() {
		buf := leakyBuf.Get()
		var err error
		if ioHack {
			err = customCopy(proxy, conn, buf)
		} else {
			_, err = io.CopyBuffer(proxy, conn, buf)
		}
		if err != nil && !copyEnd {
			PrintfLog("[ERROR] Copy error: %s.\n", err.Error())
		}
		leakyBuf.Put(buf)
		copyEnd = true
		conn.Close()
		proxy.Close()
	}()

	buf := leakyBuf.Get()
	if ioHack {
		err = customCopy(conn, proxy, buf)
	} else {
		_, err = io.CopyBuffer(conn, proxy, buf)
	}
	if err != nil && !copyEnd {
		PrintfLog("[ERROR] Copy error: %s.\n", err.Error())
	}
	leakyBuf.Put(buf)
	copyEnd = true
	conn.Close()
	proxy.Close()
}

func main() {
	flag.StringVar(&bind, "bind", "0.0.0.0:1081", "bind local address")
	flag.StringVar(&relay, "relay", "10.0.0.80:1080", "relay remove address")
	flag.BoolVar(&ioHack, "iohack", false, "is ioHackCopy default false")
	flag.BoolVar(&debugMode, "debug", false, "debug mode")
	flag.Parse()
	listener, err := net.Listen("tcp", bind)
	if err != nil {
		log.Fatalf("[ERROR] Listen tcp error: %s.\n", err.Error())
	}
	defer listener.Close()

	proxyURL, err := url.Parse("socks5://" + relay)
	if err != nil {
		log.Fatalf("[ERROR] Parse proxy address error: %s.\n", err.Error())
	}
	proxyDialer, err = proxy.FromURL(proxyURL, &net.Dialer{})
	if err != nil {
		log.Fatalf("[ERROR] Create proxy error: %s.\n", err.Error())
	}
	log.Println("[INFO] Starting goredsocks...")
	if ioHack {
		log.Println("[INFO] Using io.Copy hack.")
	}
	running = true
	go func() {
		for running {
			conn, err := listener.Accept()
			if err != nil {
				if running {
					log.Printf("[ERROR] TCP accept error: %s.\n", err.Error())
				}
				continue
			}
			go handleConnection(conn.(*net.TCPConn))
		}
	}()
	c := make(chan os.Signal)
	signal.Notify(c, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	<-c
	log.Println("[INFO] Exiting goredsocks...")
	running = false
	listener.Close()
}
