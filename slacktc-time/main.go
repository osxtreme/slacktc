package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	addr := ":" + os.Getenv("PORT")
	http.HandleFunc("/", handle)
	log.Fatal(http.ListenAndServe(addr, nil))
}

func handle(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Error parsing form.", http.StatusBadRequest)
		return
	}

	url := r.Form.Get("text")

	start := time.Now()
	_, err := http.Get(url)
	if err != nil {
		http.Error(w, "Error requesting url.", http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "%s took %s", url, time.Since(start))
}
