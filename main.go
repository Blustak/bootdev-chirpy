package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/Blustak/bootdev-chirpy/internal/database"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type Platform string

const (
    platformDev Platform = "dev"
)

type apiConfig struct {
	fileServerHits atomic.Int32
    platform Platform
	dbQueries      *database.Queries
}


func MarshalDatabaseUserJSON(u database.User) ([]byte,error) {
    user := struct {
        ID uuid.UUID `json:"id"`
        CreatedAt time.Time `json:"created_at"`
        UpdatedAt time.Time `json:"updated_at"`
        Email string `json:"email"`
    }{
        ID: u.ID,
        CreatedAt: u.CreatedAt,
        UpdatedAt: u.UpdatedAt,
        Email: u.Email,
    }
    return json.Marshal(user)
}

var restricted_words = [...]string{
	"sharbert",
	"kerfuffle",
	"fornax",
}

func (cfg *apiConfig) middlewareIncrementHits(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileServerHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func main() {
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		panic("Couldn't connect to postgresql database.")
	}
	apiState := apiConfig{
		fileServerHits: atomic.Int32{},
		dbQueries:      database.New(db),
        platform: Platform(os.Getenv("PLATFORM")),
	}
	serve := http.NewServeMux()

	serve.HandleFunc("GET /api/healthz", readinessHandler)
	serve.HandleFunc("POST /api/validate_chirp", validateChirpHandler)
    serve.HandleFunc("POST /api/users", apiState.addUserHandler)

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
    if cfg.platform != platformDev {
        w.WriteHeader(403)
        return
    }
	cfg.fileServerHits.Swap(int32(0))
    cfg.dbQueries.ResetUserTable(r.Context())
	w.WriteHeader(200)
	w.Write([]byte("Reset count."))
}

func validateChirpHandler(w http.ResponseWriter, r *http.Request) {
	type chirp struct {
		Body string `json:"body"`
	}
	type cleanChirp struct {
		CleanedBody string `json:"cleaned_body"`
	}

	decoder := json.NewDecoder(r.Body)
	var reqChirp chirp
	if err := decoder.Decode(&reqChirp); err != nil {
        clientErrorResponse(w,400,err)
		return
	}

	if len(reqChirp.Body) > 140 {
        clientErrorResponse(w,400,errors.New("chirp too long"))
	}

	var cleanOutput cleanChirp
	bodyWords := strings.Split(string(reqChirp.Body), " ")
	for i, w := range bodyWords {
		for _, word := range restricted_words {
			if strings.ToLower(w) == word {
				bodyWords[i] = "****"
			}
		}
	}
	cleanOutput.CleanedBody = strings.Join(bodyWords, " ")
	data, err := json.Marshal(cleanOutput)
	if err != nil {
        serverErrorResponse(w,500,err)
		return 
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(data)

}

func (cfg *apiConfig) addUserHandler(w http.ResponseWriter, r *http.Request) {
	reqStructure := struct {
		Email string `json:"email"`
	}{}
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&reqStructure); err != nil {
		clientErrorResponse(w, 400, err)
        return
	}
    user,err := cfg.dbQueries.CreateUser(r.Context(),reqStructure.Email)
    if err != nil {
        serverErrorResponse(w,500,err)
        return
    }
    data,err := MarshalDatabaseUserJSON(user)
    if err != nil {
        serverErrorResponse(w,500,err)
        return
    }
    w.WriteHeader(201)
    w.Header().Add("Content-Type","application/json")
    w.Write(data)
}

func serverErrorResponse(w http.ResponseWriter, statusCode int, err error) {
    log.Printf("server error: %v",err)
	w.WriteHeader(statusCode)
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintf(w, "server error: %v", err)
}

func clientErrorResponse(w http.ResponseWriter, statusCode int, err error) {
	errPayload := struct {
		Error string `json:"error"`
	}{
		Error: fmt.Sprintf("error: %v", err),
	}
    log.Printf("error: %v",err)
	data, err := json.Marshal(errPayload)
	if err != nil {
        serverErrorResponse(w,500,err)
		return
	}
	w.WriteHeader(statusCode)
	w.Header().Add("Content-Type", "application/json")
	w.Write(data)
}
