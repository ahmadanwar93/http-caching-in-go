package main

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

func withCache(cc string) http.HandlerFunc {
	// http.HandlerFunc is like Laravel controller method
	return func(w http.ResponseWriter, r *http.Request) {
		// r is like request, w is like response
		// go lang dont return response, we write into w and function returns nothing
		log.Printf("HIT SERVER: %s %s", r.Method, r.URL.Path)
		// we are only putting a value into the Cache Control header for the response
		w.Header().Set("Cache-Control", cc)
		// put the time log in the response body
		fmt.Fprintln(w, time.Now().Format(time.RFC3339Nano))
	}
}

// in Go, we pass a function and net/http calls it
// we never call the handler, only register it and it will be called with w and r
// HandleFunction is like the router
func main() {
	http.HandleFunc("/nostore", withCache("no-store"))
	http.HandleFunc("/nocache", withCache("no-cache"))
	http.HandleFunc("/fresh", withCache("max-age=30"))
	http.HandleFunc("/immutable", withCache("max-age=31536000, immutable"))

	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
