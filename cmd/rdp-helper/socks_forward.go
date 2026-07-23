package main

import (
	"fmt"
	"io"
	"net"
	"strconv"
	"sync"
	"time"

	"golang.org/x/net/proxy"
)

// startLocalForwardViaSOCKS listens on 127.0.0.1:localPort and proxies each
// connection through a SOCKS5 server to targetHost:targetPort.
func startLocalForwardViaSOCKS(socksHost string, socksPort, localPort int, targetHost string, targetPort int) (stop func(), err error) {
	dialer, err := proxy.SOCKS5("tcp", net.JoinHostPort(socksHost, strconv.Itoa(socksPort)), nil, proxy.Direct)
	if err != nil {
		return nil, fmt.Errorf("socks5 dialer: %w", err)
	}
	ln, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(localPort)))
	if err != nil {
		return nil, err
	}
	var wg sync.WaitGroup
	done := make(chan struct{})
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				select {
				case <-done:
					return
				default:
					return
				}
			}
			wg.Add(1)
			go func(c net.Conn) {
				defer wg.Done()
				defer c.Close()
				_ = c.SetDeadline(time.Now().Add(2 * time.Hour))
				remote, err := dialer.Dial("tcp", net.JoinHostPort(targetHost, strconv.Itoa(targetPort)))
				if err != nil {
					logf("socks dial %s:%d: %v", targetHost, targetPort, err)
					return
				}
				defer remote.Close()
				pipe(c, remote)
			}(c)
		}
	}()
	return func() {
		close(done)
		_ = ln.Close()
		wg.Wait()
	}, nil
}

func pipe(a, b net.Conn) {
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		_, _ = io.Copy(a, b)
		_ = a.Close()
	}()
	go func() {
		defer wg.Done()
		_, _ = io.Copy(b, a)
		_ = b.Close()
	}()
	wg.Wait()
}
