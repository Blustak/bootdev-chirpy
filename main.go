package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
)

type apiConfig struct {
	fileServerHits atomic.Int32
}

func (cfg *apiConfig) middlewareIncrementHits(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileServerHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func main() {
	apiState := apiConfig{
		fileServerHits: atomic.Int32{},
	}
	serve := http.NewServeMux()

	serve.HandleFunc("GET /api/healthz", readinessHandler)
    serve.HandleFunc("POST /api/validate_chirp", validateChirpHandler)

	serve.HandleFunc("GET /admin/metrics", apiState.hitsHandler)
	serve.HandleFunc("POST /admin/reset", apiState.resetHandler)

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
	w.Header().Add("Content-Type", "text/html")
	w.WriteHeader(200)
	fmt.Fprintf(
		w,
		`<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html> `,
		cfg.fileServerHits.Load(),
	)
}

func (cfg *apiConfig) resetHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	cfg.fileServerHits.Swap(int32(0))
	w.WriteHeader(200)
	w.Write([]byte("Reset count."))
}

func validateChirpHandler(w http.ResponseWriter, r *http.Request) {
    type chirp struct {
        Body string `json:"body"`
    }
    type serverError struct {
        Error string `json:"error"`
    }
    type validChirp struct {
        Valid bool `json:"valid"`
    }

    decoder := json.NewDecoder(r.Body)
    var reqChirp chirp
    if err := decoder.Decode(&reqChirp); err != nil{
        errMsg := fmt.Sprintf("error decoding chirp: %s",err)

        respBody := serverError{
            Error: errMsg,
        }
        data,err := json.Marshal(respBody)
        if err != nil {
            log.Printf("error marshalling JSON: %s",err)
            w.WriteHeader(500)
            return //Can't do much else if the marshalling error is happening
        }

        log.Printf(errMsg)
        w.WriteHeader(500)
        w.Write(data)
        return
    }
    // Check for length of body
    if len(reqChirp.Body) > 140 {
        respBody := serverError{
            Error: "chirp too long",
        }
        data,err := json.Marshal(respBody)
        if err != nil {
            log.Printf("error marshalling JSON: %s",err)
            w.WriteHeader(500)
            return //Can't do much else if the marshalling error is happening
        }
        w.WriteHeader(400)
        w.Write(data)
        return
    }
    respBody := validChirp{
        Valid: true,
    }
    data,err := json.Marshal(respBody)
    if err != nil {
        log.Printf("error marshalling JSON: %s",err)
        w.WriteHeader(500)
        return //Can't do much else if the marshalling error is happening
    }
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(200)
    w.Write(data)

}
