package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
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
	// for hashing response body
	http.HandleFunc("/api/products", productsHandler)

	http.HandleFunc("/api/products-v", versionedProductsHandler)
	http.HandleFunc("/api/products-v/bump", bumpHandler)

	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

type Product struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Price int    `json:"price"`
}

var products = []Product{
	{1, "Keyboard", 4500},
	{2, "Mouse", 250},
}

func productsHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("HIT SERVER: %s %s If-None-Match=%q",
		r.Method, r.URL.Path, r.Header.Get("If-None-Match"))

	body, err := json.Marshal(products)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// sha256 always produces 32 bytes whether the input is 1 byte or 1 gb
	sum := sha256.Sum256(body)
	// just encode the first 8 bytes and sent as eTag
	// the reason why we are only taking the first 8 bytes rather than the whole 32 bytes is for simplicity sake
	// the risk of collision is small enough
	etag := `"` + hex.EncodeToString(sum[:8]) + `"`

	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("ETag", etag)

	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(body)
}

// same as writing 2 separate lines
// var mu sync.RWMutex
// var version = 1
var (
	mu      sync.RWMutex // guards products and version
	version = 1
)

func versionedProductsHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("HIT SERVER: %s %s If-None-Match=%q",
		r.Method, r.URL.Path, r.Header.Get("If-None-Match"))

	mu.RLock()
	etag := fmt.Sprintf(`"v%d"`, version)

	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("ETag", etag)

	// cheap check first: no JSON, no hashing
	if r.Header.Get("If-None-Match") == etag {
		mu.RUnlock()
		w.WriteHeader(http.StatusNotModified)
		return
	}

	log.Println("BUILDING BODY") // must NOT appear on a 304
	body, err := json.Marshal(products)
	mu.RUnlock()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(body)
}

// simulates "someone edited a product"
func bumpHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	mu.Lock()
	products[1].Price += 100
	version++
	v := version
	mu.Unlock()
	fmt.Fprintf(w, "version is now %d\n", v)
}
