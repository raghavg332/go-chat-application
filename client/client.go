package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

var (
	appendNewline bool
	conn          net.Conn
)

func handleSigint() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT)
	go func() {
		<-ch
		fmt.Println("detected exit")
		if conn != nil {
			_ = conn.Close()
		}
		os.Exit(0)
	}()
}

func main() {
	flag.BoolVar(&appendNewline, "append-newline", false, "append a newline when sending (use if server expects line-based input)")
	flag.Parse()

	handleSigint()

	var err error
	// conn, err = net.Dial("tcp", "0.0.0.0:8081")
	conn, err = net.Dial("tcp", "13.200.235.191:8080")
	if err != nil {
		fmt.Println("connect:", err)
		return
	}
	fmt.Println("connected to server")

	// --- Goroutine: read from server like C++ recv() ---
	done := make(chan struct{})
	go func() {
		defer close(done)
		buf := make([]byte, 1024)
		for {
			n, err := conn.Read(buf)
			if err != nil {
				if err == io.EOF {
					fmt.Fprintln(os.Stderr, "connection disconnected")
				} else {
					fmt.Fprintln(os.Stderr, "receive:", err)
				}
				return
			}
			if n == 0 {
				fmt.Fprintln(os.Stderr, "connection disconnected")
				return
			}
			// Clear current line, print server data as-is
			fmt.Print("\x1b[2K\r")
			os.Stdout.Write(buf[:n])
			// optional newline echo is up to server; we don't force here
		}
	}()

	// --- Goroutine: read stdin and send like C++ (no newline by default) ---
	go func() {
		reader := bufio.NewReader(os.Stdin)
		for {
			line, err := reader.ReadString('\n') // blocks until Enter
			if err != nil {
				// stdin closed; close socket and exit sender
				_ = conn.Close()
				return
			}
			// C++ getline strips newline; replicate that
			line = strings.TrimRight(line, "\r\n")
			if appendNewline {
				line += "\n"
			}
			if _, err := conn.Write([]byte(line)); err != nil {
				fmt.Fprintln(os.Stderr, "send:", err)
				_ = conn.Close()
				return
			}
		}
	}()

	<-done // wait for server reader to finish
	if conn != nil {
		_ = conn.Close()
	}
}
