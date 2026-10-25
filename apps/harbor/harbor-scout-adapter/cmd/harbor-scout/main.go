package main

import (
	"flag"
	"log"
	"net/http"
	"time"

	"harbor-scout-adapter/internal/api"
	"harbor-scout-adapter/internal/scanner"
)

func main() {
	addr := flag.String(
		"listen",
		"127.0.0.1:8080",
		"HTTP listen address",
	)

	sarif := flag.String(
		"sarif",
		"",
		"SARIF fixture path",
	)

	live := flag.Bool(
		"live",
		false,
		"Run Docker Scout CLI",
	)

	flag.Parse()

	var service *scanner.Service

	if *live {
		service = scanner.NewLive()
		log.Println("Docker Scout live mode enabled")
	} else {
		if *sarif == "" {
			log.Fatal("specify -sarif or -live")
		}
		service = scanner.New(*sarif)
		log.Println("SARIF simulation mode enabled")
	}

	server := &http.Server{
		Addr:              *addr,
		Handler:           api.New(service),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
	}

	log.Printf("Listening on %s", *addr)
	log.Fatal(server.ListenAndServe())
}
