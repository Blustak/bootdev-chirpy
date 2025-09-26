package main

import (
	"fmt"
	"net/http"
	"sync/atomic"
)

type apiConfig struct {
	fileServerHits atomic.Int32
}

func (cfg *apiConfig) middlewareIncrementHits(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        cfg.fileServerHits.Add(1)
        next.ServeHTTP(w,r)
    })
}

func main() {
	apiState := apiConfig{
		fileServerHits: atomic.Int32{},
	}
	serve := http.NewServeMux()

	serve.HandleFunc("/healthz", readinessHandler)
	serve.HandleFunc("/metrics", apiState.hitsHandler)
	serve.HandleFunc("/reset", apiState.resetHandler)
	fileServeHandle := http.StripPrefix(
			"/app", http.FileServer(http.Dir(".")))
	serve.Handle("/app/", apiState.middlewareIncrementHits(fileServeHandle))
	serve.Handle("/assets", http.FileServer(http.Dir("./assets")))

	server := http.Server{
		Handler: serve,
		Addr:    ":8080",
	}

	server.ListenAndServe()
}

func readinessHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}

func (cfg *apiConfig) hitsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
    hits_str := fmt.Appendf(nil, "Hits: %d", cfg.fileServerHits.Load())
    w.Write(hits_str)
}

func (cfg *apiConfig) resetHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	cfg.fileServerHits.Swap(int32(0))
	w.WriteHeader(200)
	w.Write([]byte("Reset count."))
}
