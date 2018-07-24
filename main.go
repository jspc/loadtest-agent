package main

import (
	"crypto/tls"
	"flag"
	"log"
	"net/http"
	"time"

	"github.com/jspc/loadtest"
)

const (
	// DeadLetterDatabase is the place to send data when a job
	// name hasn't been specified
	DeadLetterDatabase = "dead_letter"
)

var (
	collector = flag.String("collector", "http://localhost:8082", "Collector endpoint")
	insecure  = flag.Bool("insecure", false, "Allow access to https endpoints with shit certs")
	logDir    = flag.String("logs", "/var/log/loadtest-agent", "Dir to log to")
)

func main() {
	flag.Parse()

	if *insecure {
		http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	jobs := make(chan Job, 32)

	api := API{
		UploadDir: "/tmp/",
		Jobs:      jobs,
		Binaries:  NewBinaries(),
	}

	collector, err := NewCollector(*collector, DeadLetterDatabase)
	if err != nil {
		panic(err)
	}

	go func() {
		for j := range jobs {
			if j.Name == "" {
				j.Name = DeadLetterDatabase
			}
			collector.Database = j.Name

			outputs := make(chan loadtest.Output)

			var lastRead time.Time
			go func() {
				for o := range outputs {
					lastRead = time.Now()

					err := collector.Push(o)
					if err != nil {
						log.Print(err)
					}
				}
			}()

			err = j.Start(outputs)
			if err != nil {
				log.Print(err)
			}

			log.Print("Entering cooldown period")

			// Wait until we've received nothing for a minute
			// in the hopes that this is enough for the final
			// requests to end
			for {
				if time.Now().Sub(lastRead).Seconds() > 60.0 {
					break
				}

				time.Sleep(500 * time.Millisecond)
			}

			close(outputs)
			log.Printf("ran %d times", j.items)
		}
	}()

	panic(http.ListenAndServe(":8081", api))
}
