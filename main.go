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

	"github.com/Blustak/bootdev-chirpy/internal/auth"
	"github.com/Blustak/bootdev-chirpy/internal/database"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type Chirp struct {
	database.Chirp
}

func (c Chirp) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{
		"id":         c.ID,
		"created_at": c.CreatedAt,
		"updated_at": c.UpdatedAt,
		"body":       c.Body,
		"user_id":    c.UserID,
	})
}


type createUserRow database.CreateUserRow
type updateUserRow database.UpdateUserRow

type getUserByEmailRow database.GetUserByEmailRow

func (u createUserRow) User() User {
	return User{
		ID:        u.ID,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
		Email:     u.Email,
        IsChirpyRed: u.IsChirpyRed,
	}
}

func (u updateUserRow) User() User {
    return User{
        ID: u.ID,
        CreatedAt: u.CreatedAt,
        UpdatedAt: u.UpdatedAt,
		Email:     u.Email,
        IsChirpyRed: u.IsChirpyRed,
    }
}

func (u getUserByEmailRow) User() User {
	return User{
		ID:        u.ID,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
		Email:     u.Email,
        IsChirpyRed: u.IsChirpyRed,
	}
}

type User struct {
	ID           uuid.UUID `json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Email        string    `json:"email"`
	Token        string    `json:"token"`
	RefreshToken string    `json:"refresh_token"`
    IsChirpyRed bool `json:"is_chirpy_red"`
}

type userLoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type Platform string

const (
	platformDev Platform = "dev"
)

type apiConfig struct {
	fileServerHits atomic.Int32
	platform       Platform
	dbQueries      *database.Queries
	tokenSecret    string
    polkaAPIKey string
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

func lookupKeyOrPanic(envVariable string) string {
    v,ok := os.LookupEnv(envVariable)
    if !ok {
        panic(fmt.Sprintf("Couldn't find environment variable %s", envVariable))
    }
    if v == "" {
        panic(fmt.Sprintf("Empty environment variable %s", envVariable))
    }
    return v
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
		platform:       Platform(os.Getenv("PLATFORM")),
		tokenSecret: func() string {
            s := lookupKeyOrPanic("TOKEN_STRING")
			if len(s) < 64 {
				panic("bad secrets token")
			}
			return s
		}(),
        polkaAPIKey: lookupKeyOrPanic("POLKA_KEY"),
	}
	serve := http.NewServeMux()

	serve.HandleFunc("GET /api/healthz", readinessHandler)

	serve.HandleFunc("POST /api/users", apiState.addUserHandler)
	serve.HandleFunc("PUT /api/users", apiState.updateUserHandler)

	serve.HandleFunc("POST /api/login", apiState.userLoginHandler)

	serve.HandleFunc("POST /api/refresh", apiState.refreshTokenHandler)
	serve.HandleFunc("POST /api/revoke", apiState.revokeHandler)

	serve.HandleFunc("POST /api/chirps", apiState.chirpsHandler)
	serve.HandleFunc("GET /api/chirps", apiState.getAllChirpsHandler)
	serve.HandleFunc("GET /api/chirps/{chirpID}", apiState.getChirpByIdHandler)
    serve.HandleFunc("DELETE /api/chirps/{chirpID}", apiState.deleteChirpByIDHandler)

    serve.HandleFunc("POST /api/polka/webhooks", apiState.polkaWebhooksHandler)

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
	w.Write([]byte("Reset\n"))
}

func (cfg *apiConfig) chirpsHandler(w http.ResponseWriter, r *http.Request) {

	decoder := json.NewDecoder(r.Body)
	var requestChirp struct {
		ChirpBody string `json:"body"`
	}
	if err := decoder.Decode(&requestChirp); err != nil {
		clientErrorResponse(w, 400, err)
		return
	}
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		clientErrorResponse(w, 400, err)
	}
	id, err := auth.ValidateJWT(token, cfg.tokenSecret)
	if err != nil {
		clientErrorResponse(w, 401, err)
	}

	if len(requestChirp.ChirpBody) > 140 {
		clientErrorResponse(w, 400, errors.New("chirp too long"))
		return
	}

	bodyWords := strings.Split(string(requestChirp.ChirpBody), " ")
	for i, w := range bodyWords {
		for _, word := range restricted_words {
			if strings.ToLower(w) == word {
				bodyWords[i] = "****"
			}
		}
	}
	requestChirp.ChirpBody = strings.Join(bodyWords, " ")
	var res Chirp
	res.Chirp, err = cfg.dbQueries.AddChirp(r.Context(), database.AddChirpParams{
		ChirpBody: requestChirp.ChirpBody,
		ID:        id,
	})
	if err != nil {
		serverErrorResponse(w, 500, err)
		return
	}
	data, err := json.Marshal(Chirp(res))
	if err != nil {
		serverErrorResponse(w, 500, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(201)
	w.Write(data)
}

func (cfg *apiConfig) addUserHandler(w http.ResponseWriter, r *http.Request) {
	reqStructure := userLoginRequest{}
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&reqStructure); err != nil {
		clientErrorResponse(w, 400, err)
		return
	}
	hashedPassword, err := auth.HashPassword(reqStructure.Password)
	if err != nil {
		log.Printf("error hashing password: %v\n", err)
		serverErrorResponse(w, 400, err)
		return
	}
	q, err := cfg.dbQueries.CreateUser(r.Context(),
		database.CreateUserParams{
			Email:          reqStructure.Email,
			HashedPassword: hashedPassword,
		})
	if err != nil {
		log.Printf("error adding user : %v\n", err)
		serverErrorResponse(w, 500, err)
		return
	}
	user := createUserRow(q).User()
	data, err := json.Marshal(user)
	if err != nil {
		log.Printf("error marshaling response : %v\n", err)
		serverErrorResponse(w, 500, err)
		return
	}
	w.WriteHeader(201)
	w.Header().Add("Content-Type", "application/json")
	w.Write(data)
}

func (cfg *apiConfig) updateUserHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, err := auth.GetBearerToken(r.Header)
	if err != nil || accessToken == "" {
		clientErrorResponse(w, 401, err)
		return
	}
	userID, err := auth.ValidateJWT(accessToken, cfg.tokenSecret)
	if err != nil {
		clientErrorResponse(w, 401, err)
		return
	}
	decoder := json.NewDecoder(r.Body)
	putData := struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}{}
    if err = decoder.Decode(&putData); err != nil {
        clientErrorResponse(w,401,err)
        return
    }
    hashedPass, err := auth.HashPassword(putData.Password)
    if err != nil {
        serverErrorResponse(w,500,err)
        return
    }
    userQuery, err := cfg.dbQueries.UpdateUser(
        r.Context(),
        database.UpdateUserParams{
            Email: putData.Email,
            HashedPassword: hashedPass,
            UserID: userID,
        },
    )
    if err != nil {
        serverErrorResponse(w,500,err)
        return
    }
    w.WriteHeader(200)
    encoder := json.NewEncoder(w)
    if err = encoder.Encode(
        updateUserRow(userQuery).User(),
    ); err != nil {
        serverErrorResponse(w,500,err)
        return
    }
}

func (cfg *apiConfig) userLoginHandler(w http.ResponseWriter, r *http.Request) {
	var req userLoginRequest
	var err error

	decoder := json.NewDecoder(r.Body)
	if err = decoder.Decode(&req); err != nil {
		serverErrorResponse(w, 500, err)
		return
	}
	row, err := cfg.dbQueries.GetUserByEmail(r.Context(), req.Email)
	if err != nil || row == (database.GetUserByEmailRow{}) {
		log.Printf("error getting user by email: %v", err)
		w.WriteHeader(401)
		w.Header().Add("Content-Type", "text/plain; charset=utf-8")
		w.Write([]byte("incorrect email or password"))
		return
	}

	hashedPass, err := cfg.dbQueries.GetHashedPasswordByID(r.Context(), row.ID)
	if err != nil {
		log.Printf("error getting hashed password:%v", err)
		serverErrorResponse(w, 500, err)
		return
	}
	ok, err := auth.CheckPasswordHash(req.Password, hashedPass)
	if err != nil || !ok {
		log.Printf("error checking hashed password:%v", err)
		w.WriteHeader(401)
		w.Header().Add("Content-Type", "text/plain; charset=utf-8")
		w.Write([]byte("incorrect email or password"))
		return
	}
	user := getUserByEmailRow(row).User()
	user.Token, err = auth.MakeJWT(user.ID, cfg.tokenSecret, time.Hour*1)
	if err != nil {
		serverErrorResponse(w, 500, err)
		return
	}

	user.RefreshToken, err = auth.MakeRefreshToken()
	if err != nil {
		serverErrorResponse(w, 500, err)
		return
	}
	res, err := cfg.dbQueries.AddRefreshToken(r.Context(), database.AddRefreshTokenParams{
		Token:  user.RefreshToken,
		UserID: user.ID,
	})
	if err != nil {
		serverErrorResponse(w, 500, err)
		return
	}
	log.Printf("Added refresh token %v", res)
	data, err := json.Marshal(user)
	if err != nil {
		log.Printf("error marshalling response: %v", err)
		serverErrorResponse(w, 500, err)
		return
	}
	w.WriteHeader(200)
	w.Header().Add("Content-Type", "application/json")
	w.Write(data)

}

func (cfg *apiConfig) refreshTokenHandler(w http.ResponseWriter, r *http.Request) {
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		clientErrorResponse(w, 400, err)
		return
	}
	refreshTokenQuery, err := cfg.dbQueries.GetUserByRefreshToken(r.Context(), token)
	log.Printf("Got query: %v", refreshTokenQuery)
	if err != nil || (refreshTokenQuery == database.GetUserByRefreshTokenRow{}) {
		if errors.Is(err, sql.ErrNoRows) {
			clientErrorResponse(w, 401, err)
			return
		}
	}
	if refreshTokenQuery.ExpiresAt.Valid {
		if time.Now().After(refreshTokenQuery.ExpiresAt.Time) {
			clientErrorResponse(w, 401, errors.New("refresh token has expired"))
			return
		}
	} else {
		panic("this should be unreachable; refresh token's expires at should never be null.")
	}
	if refreshTokenQuery.RevokedAt.Valid {
		clientErrorResponse(w, 401, errors.New("token has been revoked"))
		return
	}
	accessToken, err := auth.MakeJWT(refreshTokenQuery.ID, cfg.tokenSecret, time.Hour*1)
	if err != nil {
		serverErrorResponse(w, 500, errors.New("failed to create jwt token"))
		return
	}
	data, err := json.Marshal(&struct {
		Token string `json:"token"`
	}{
		Token: accessToken,
	})
	if err != nil {
		serverErrorResponse(w, 500, err)
		return
	}
	w.WriteHeader(200)
	w.Header().Add("Content-Type", "application/json")
	w.Write(data)
}

func (cfg *apiConfig) revokeHandler(w http.ResponseWriter, r *http.Request) {
	token, err := auth.GetBearerToken(r.Header)
	if err != nil || token == "" {
		clientErrorResponse(w, 400, errors.New("couldn't find refresh token"))
		return
	}
	if err := cfg.dbQueries.RevokeRefreshToken(r.Context(), token); err != nil {
		clientErrorResponse(w, 401, err)
		return
	}
	w.WriteHeader(204)

}

func (cfg *apiConfig) polkaWebhooksHandler(w http.ResponseWriter, r *http.Request) {
    apiKey, err := auth.GetApiKeyToken(r.Header)
    if err != nil || apiKey != cfg.polkaAPIKey {
        clientErrorResponse(w,401,errors.New("unaothorized"))
        return
    }
    var polkaWebhookEvent struct {
        Event string `json:"event"`
        Data struct{
            UserID uuid.UUID `json:"user_id"`
        } `json:"data"`
    }
    decoder := json.NewDecoder(r.Body)
    if err := decoder.Decode(&polkaWebhookEvent); err != nil {
        clientErrorResponse(w,400,err)
        return
    }
    if polkaWebhookEvent.Event != "user.upgraded" {
        w.WriteHeader(204)
        return
    }
    if err := cfg.dbQueries.UpgradeUserToChirpyRed(r.Context(),polkaWebhookEvent.Data.UserID); err != nil {
        w.WriteHeader(404)
        return
    }
    w.WriteHeader(204)
}

func (cfg *apiConfig) getAllChirpsHandler(w http.ResponseWriter, r *http.Request) {
	var chirps []Chirp
	query, err := cfg.dbQueries.GetAllChirps(r.Context())
	if err != nil {
		serverErrorResponse(w, 500, err)
		return
	}

	for _, q := range query {
		chirps = append(chirps, Chirp{Chirp: q})
	}
	data, err := json.Marshal(chirps)
	if err != nil {
		serverErrorResponse(w, 500, err)
		return
	}
	w.WriteHeader(200)
	w.Header().Add("Content-Type", "application/json")
	w.Write(data)

}

func (cfg *apiConfig) getChirpByIdHandler(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("chirpID"))
	if err != nil {
		clientErrorResponse(w, 404, err)
		return
	}
	var query Chirp
	query.Chirp, err = cfg.dbQueries.GetChirpByID(r.Context(), id)
	if err != nil {
		clientErrorResponse(w, 404, err)
		return
	}
	data, err := json.Marshal(query)
	if err != nil {
		serverErrorResponse(w, 500, err)
		return
	}
	w.WriteHeader(200)
	w.Header().Add("Content-Type", "application/json")
	w.Write(data)

}

func (cfg *apiConfig) deleteChirpByIDHandler(w http.ResponseWriter, r *http.Request) {
    accessToken,err := auth.GetBearerToken(r.Header)
    if err != nil || accessToken == "" {
        clientErrorResponse(w, 401, errors.New("not authorized"))
        return
    }

    userID, err := auth.ValidateJWT(accessToken,cfg.tokenSecret)
    if err != nil {
        clientErrorResponse(w,401, err)
        return
    }
    chirpID, err := uuid.Parse(r.PathValue("chirpID"))
    if err != nil {
        clientErrorResponse(w,404,err)
        return
    }
    chirpQuery, err := cfg.dbQueries.GetChirpByID(
        r.Context(),
        chirpID,
    )
    if err != nil {
        clientErrorResponse(w, 404, err)
        return
    }
    if chirpQuery.UserID != userID {
        clientErrorResponse(w,403,errors.New("user mismatch"))
        return
    }
    if err = cfg.dbQueries.DeleteChirpByID(r.Context(),chirpID); err != nil {
        serverErrorResponse(w,500,err)
        return
    }
    w.WriteHeader(204)
}

func serverErrorResponse(w http.ResponseWriter, statusCode int, err error) {
	log.Printf("server error: %v", err)
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
	log.Printf("error: %v", err)
	data, err := json.Marshal(errPayload)
	if err != nil {
		serverErrorResponse(w, 500, err)
		return
	}
	w.WriteHeader(statusCode)
	w.Header().Add("Content-Type", "application/json")
	w.Write(data)
}
