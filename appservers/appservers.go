package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

// IndexHanlder returns the landing response of the app server.
func indexHanlder(w http.ResponseWriter, r *http.Request) {
	report := fmt.Sprintf("This is the %s application.", os.Getenv("APP"))
	w.Write([]byte(report))
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Ok"))
}

func main() {
	http.HandleFunc("/", indexHanlder)
	http.HandleFunc("/healthcheck", healthCheck)
	log.Fatal(http.ListenAndServe(":5008", nil))
}
