package main

import (
	"flag"
	"fmt"
	"github.com/bradfitz/go-smtpd/smtpd"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

var smtpAddrs *string = flag.String("smtp", ":25", "SMTP adddresses")
var httpAddrs *string = flag.String("http", ":80", "HTTP addresses")

// Just dump the mail
func onNewMail(c smtpd.Connection, from smtpd.MailAddress) (env smtpd.Envelope, err error) {
	log.Printf("%v %v", c.Addr(), from)
	env = new(smtpd.BasicEnvelope)
	env.AddRecipient(from)
	return
}

func main() {
	flag.Parse()

	addrs := strings.Split(*smtpAddrs, ",")
	wg := sync.WaitGroup{}

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

	http.HandleFunc("/", func(rw http.ResponseWriter, req *http.Request) {
		req.Write(os.Stdout)
		fmt.Fprintf(rw, "OK")
		fmt.Println()
		fmt.Println()
	})

	addrs = strings.Split(*httpAddrs, ",")
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

	wg.Wait()

}
