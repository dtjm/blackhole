package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/bradfitz/go-smtpd/smtpd"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

var smtpAddrs *string = flag.String("smtp", "", "SMTP adddresses")
var httpAddrs *string = flag.String("http", "", "HTTP addresses")
var tcpAddrs *string = flag.String("tcp", "", "Command server addresses")

// Just dump the mail
func onNewMail(c smtpd.Connection, from smtpd.MailAddress) (env smtpd.Envelope, err error) {
	log.Printf("%v %v", c.Addr(), from)
	env = new(smtpd.BasicEnvelope)
	env.AddRecipient(from)
	return
}

func main() {
	flag.Parse()
	wg := sync.WaitGroup{}

	if *smtpAddrs == "" && *httpAddrs == "" && *tcpAddrs == "" {
		flag.PrintDefaults()
		os.Exit(0)
	}

	if *smtpAddrs != "" {
		addrs := strings.Split(*smtpAddrs, ",")
		for _, addr := range addrs {
			wg.Add(1)
			log.Printf("Listening on smtp://%s", addr)
			go func() {
				server := smtpd.Server{
					Addr:         addr,
					Hostname:     "", // use system hostname
					ReadTimeout:  60 * time.Second,
					WriteTimeout: 60 * time.Second,
					PlainAuth:    false,
					OnNewMail:    onNewMail,
				}
				err := server.ListenAndServe()
				if err != nil {
					log.Fatalf("Failed to start server: %s", err)
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
