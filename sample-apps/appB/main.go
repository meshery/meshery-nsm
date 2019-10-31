package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Println("received a request")
		fmt.Fprintf(w, "Welcome to NSM!")
		log.Println("request processed successfully")
	})

	http.ListenAndServe(":80", nil)
}
