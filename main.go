package main

import (
	"crypto/tls"
	"errors"
	"flag"
	"io"
	"log"
	"net"
	"sync"
	"syscall"
)

var (
	debugEnabled = false // could use a proper logger but .. eh.
)

func loadTLSConfig(certFile, keyFile string) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}
	return &tls.Config{Certificates: []tls.Certificate{cert}}, nil
}

func proxyConnection(upstream string, down net.Conn) error {
	defer down.Close()

	up, err := net.Dial("tcp", upstream)
	if err != nil {
		return err
	}
	defer up.Close()

	closeBoth := func() { // explicit stop
		down.Close()
		up.Close()
	}

	errors := make(chan error)
	defer close(errors)

	var firstErr error
	go func() {
		closeCalled := false
		for err := range errors {
			if closeCalled {
				continue
			}

			closeBoth()
			closeCalled = true

			if err == nil || isNetworkClosed(err) {
				continue
			}
			firstErr = err
		}
	}()

	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		defer wg.Done()
		errors <- pump(up, down)
	}()
	go func() {
		defer wg.Done()
		errors <- pump(down, up)
	}()

	wg.Wait()
	return firstErr
}

func isNetworkClosed(err error) bool {
	switch {
	case
		errors.Is(err, net.ErrClosed),
		errors.Is(err, io.EOF),
		errors.Is(err, syscall.EPIPE):
		return true
	default:
		return false
	}
}

func pump(from, to net.Conn) error {
	buf := make([]byte, 1024)
	for {
		n, err := from.Read(buf)
		if err != nil {
			return err
		}
		if _, err := to.Write(buf[:n]); err != nil {
			return err
		}
		if debugEnabled {
			log.Println("Sent", n, "bytes from", from.LocalAddr(), "to", to.LocalAddr())
		}
	}
}

func main() {
	port := flag.String("port", "443", "listening port")
	certFile := flag.String("cert", "cert.pem", "certificate PEM file")
	keyFile := flag.String("key", "key.pem", "key PEM file")
	upstream := flag.String("upstream", "localhost:8000", "upstream server")
	routines := flag.Int("routines", 50, "number of concurrent routines handling connections")
	debug := flag.Bool("debug", false, "enable debug logging")

	flag.Parse()
	debugEnabled = *debug

	tlsConfig, err := loadTLSConfig(*certFile, *keyFile)
	if err != nil {
		panic(err)
	}

	listener, err := tls.Listen("tcp", ":"+*port, tlsConfig)
	if err != nil {
		panic(err)
	}
	defer listener.Close()

	errors := make(chan error)
	go func() {
		for err := range errors {
			if err != nil {
				log.Println("Error", err)
			}
		}
	}()

	work := make(chan net.Conn)
	for i := 0; i < *routines; i++ {
		go func() {
			for conn := range work {
				errors <- proxyConnection(*upstream, conn)
			}
		}()
	}

	log.Println("Listening on port", *port)
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("Error accepting connection", err)
			continue
		}
		work <- conn
	}
}
