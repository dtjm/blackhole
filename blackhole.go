package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"sendlib/net/smtp"
	"strings"
	"sync"
	"time"
)

var smtpAddrs *string = flag.String("smtp", "", "SMTP adddresses")
var httpAddrs *string = flag.String("http", "", "HTTP addresses")
var tcpAddrs *string = flag.String("tcp", "", "Command server addresses")

func main() {
	flag.Parse()
	wg := sync.WaitGroup{}

	if *smtpAddrs == "" && *httpAddrs == "" && *tcpAddrs == "" {
		flag.PrintDefaults()
		os.Exit(0)
	}

	if *smtpAddrs != "" {
		addrs := strings.Split(*smtpAddrs, ",")
		server := smtp.Server{
			Greeting:     "blackhole ready.",
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			Handler: smtp.HandleFunc(func(e *smtp.Envelope, s *smtp.Session) {
				fmt.Printf("HELO %s\n", e.Helo)
				fmt.Printf("MAIL FROM: %s\n", e.MailFrom)
				for rcpt := range e.Recipients {
					fmt.Printf("RCPT TO: %s\n", rcpt)
				}

				io.Copy(os.Stdout, e.Data)
			})}
		for _, addr := range addrs {
			wg.Add(1)
			log.Printf("Listening on smtp://%s", addr)
			go func() {
				l, err := net.Listen("tcp", addr)
				if err != nil {
					log.Fatalf("Failed to listen on %s: %s", addr, err)
				}
				err = server.Serve(l)
				if err != nil {
					log.Fatalf("Failed to serve smtp://%s: %s", addr, err)
				}
				wg.Done()
			}()
		}
	}

	http.HandleFunc("/", func(rw http.ResponseWriter, req *http.Request) {
		req.Write(os.Stdout)
		fmt.Fprintf(rw, "OK")
		fmt.Println()
		fmt.Println()
	})

	if *httpAddrs != "" {
		addrs := strings.Split(*httpAddrs, ",")
		for _, addr := range addrs {
			wg.Add(1)
			log.Printf("Listening on http://%s", addr)
			go func() {
				err := http.ListenAndServe(addr, nil)
				if err != nil {
					log.Fatalf("Failed to start HTTP server: %s", err)
				}
				wg.Done()
			}()
		}
	}

	if *tcpAddrs != "" {
		addrs := strings.Split(*tcpAddrs, ",")
		for _, addr := range addrs {
			wg.Add(1)
			log.Printf("Listening on tcp://%s", addr)
			go func() {
				l, err := net.Listen("tcp", addr)
				if err != nil {
					log.Fatalf("Failed to start Command server: %s", err)
				}
				for {
					conn, err := l.Accept()
					if err != nil {
						log.Printf("Error accepting on tcp://%s: %s", addr, err)
					}

					go func() {
						scanner := bufio.NewScanner(conn)
						for scanner.Scan() {
							line := scanner.Text()
							fmt.Printf("%s: %s\n", conn.RemoteAddr().String(), line)
							conn.Write([]byte(line))
							conn.Write([]byte{'\n'})
						}
					}()
				}
			}()
		}
	}

	wg.Wait()
}
