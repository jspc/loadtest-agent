package main

import (
	"log"
	"net/http"

	"github.com/jspc/loadtest"
)

const (
	// DeadLetterDatabase is the place to send data when a job
	// name hasn't been specified
	DeadLetterDatabase = "dead_letter"
)

func main() {
	jobs := make(chan Job, 32)

	api := API{
		UploadDir: "/tmp/",
		Jobs:      jobs,
		Binaries:  NewBinaries(),
	}

	collector, err := NewCollector("http://localhost:8082", DeadLetterDatabase)
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

			go func() {
				for o := range outputs {
					err := collector.Push(o)
					if err != nil {
						log.Print(err)
					}
				}

			}()

			j.Start(outputs)
			close(outputs)

			log.Printf("ran %d times", j.items)
		}
	}()

	go func() {
	}()

	panic(http.ListenAndServe(":8081", api))
}
