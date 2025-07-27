package main

import (
	"log"
	"net/http"
)

func main() {
	addr := ":8080"
	log.Printf("file-archiver is listening on %s\n", addr)

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(err)
	}
}
