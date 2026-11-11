package main

import (
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"time"

	"encoding/json"
	"net/http"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/tmoson/chirpy/internal/auth"
	"github.com/tmoson/chirpy/internal/database"
)

type ChirpyServer struct {
	Server http.Server
	Mux    *http.ServeMux
	Api    *apiConfig
}

func (cs *ChirpyServer) registerHandlerFunction(path string, handler http.HandlerFunc) {
	cs.Mux.HandleFunc(path, handler)
}

func (cs *ChirpyServer) registerHandler(path string, handler http.Handler) {
	cs.Mux.Handle(path, handler)
}

func (cs *ChirpyServer) runServer() error {
	return cs.Server.ListenAndServe()
}

type DefaultRequestBody struct {
	Body string `json:"body"`
}

type NewUserRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type UserLoginRequest struct {
	Email            string `json:"email"`
	Password         string `json:"password"`
	ExpiresInSeconds int    `json:"expires_in_seconds,omitempty"`
}

type UserLoginResponse struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
	Token     string    `json:"token"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type ValidChirpResponse struct {
	Valid bool `json:"valid"`
}

type CleanedChirp struct {
	CleanedBody string `json:"cleaned_body"`
}

func newChirpyServer(port string) *ChirpyServer {
	mux := http.NewServeMux()
	return &ChirpyServer{
		Server: http.Server{
			Addr:    ":" + port,
			Handler: mux,
		},
		Mux: mux,
		Api: &apiConfig{},
	}
}

func sensorPost(s string) string {
	// keep a slice of sensored phrases and specify case insensitivity
	sensors := []*regexp.Regexp{
		regexp.MustCompile("(?i)kerfuffle"),
		regexp.MustCompile("(?i)sharbert"),
		regexp.MustCompile("(?i)fornax"),
	}
	sensored := s
	for _, sensor := range sensors {
		sensored = sensor.ReplaceAllString(sensored, "****")
	}
	return sensored
}

func chirpPostHandler(server *ChirpyServer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userToken, err := auth.GetBearerToken(r.Header)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		userID, err := auth.ValidateJWT(userToken, server.Api.jwtSecret)
		if err != nil {
			if errors.Is(err, jwt.ErrTokenExpired) {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		body := DefaultRequestBody{}
		decoder := json.NewDecoder(r.Body)
		err = decoder.Decode(&body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if len(body.Body) > 140 {
			tooLong := ErrorResponse{
				Error: "Chirp is too long",
			}
			rBody, err := json.Marshal(tooLong)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write(rBody)
			return
		}
		post, err := server.Api.dbQueries.CreatePost(r.Context(), database.CreatePostParams{
			Body:   sensorPost(body.Body),
			UserID: userID,
		})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		respBody, err := json.Marshal(post)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write(respBody)
	})
}

func handleNewUser(server *ChirpyServer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		newUserBody := NewUserRequest{}
		decoder := json.NewDecoder(r.Body)
		err := decoder.Decode(&newUserBody)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		passwordHash, err := auth.HashPassword(newUserBody.Password)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		newUser, err := server.Api.dbQueries.CreateUser(r.Context(), database.CreateUserParams{
			Email:          newUserBody.Email,
			HashedPassword: passwordHash,
		})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		response, err := json.Marshal(newUser)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
		w.Header().Set("Content-Type", "application/json")
		w.Write(response)
	})
}

func loginUser(server *ChirpyServer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userBody := UserLoginRequest{}
		decoder := json.NewDecoder(r.Body)
		err := decoder.Decode(&userBody)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		user, err := server.Api.dbQueries.GetUser(r.Context(), userBody.Email)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		passwordMatch, err := auth.CheckPasswordHash(userBody.Password, user.HashedPassword)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if passwordMatch {
			tokenDuration, err := time.ParseDuration("1h")
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			if userBody.ExpiresInSeconds > 0 && userBody.ExpiresInSeconds < 3600 {
				tokenDuration, err = time.ParseDuration(fmt.Sprintf("%ds", userBody.ExpiresInSeconds))
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
			}
			token, err := auth.MakeJWT(user.ID, server.Api.jwtSecret, tokenDuration)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			response, err := json.Marshal(UserLoginResponse{
				ID:        user.ID,
				CreatedAt: user.CreatedAt,
				UpdatedAt: user.UpdatedAt,
				Email:     user.Email,
				Token:     token,
			})
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "application/json")
			w.Write(response)
			return
		}
		response, err := json.Marshal(ErrorResponse{
			Error: "Incorrect email or password",
		})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
		w.Header().Set("Content-Type", "application/json")
		w.Write(response)
	})
}

func getAllChirps(server *ChirpyServer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		posts, err := server.Api.dbQueries.GetAllPosts(r.Context())
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		postResponse, err := json.Marshal(posts)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		w.Write(postResponse)
	})
}

func getChirp(server *ChirpyServer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		chirpID := r.PathValue("id")
		if chirpID == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		chirpUUID, err := uuid.Parse(chirpID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		chirp, err := server.Api.dbQueries.GetPost(r.Context(), chirpUUID)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		chirpJSON, err := json.Marshal(chirp)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		w.Write(chirpJSON)
	})
}

func getAdminMetrics(server *ChirpyServer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Add("content-type", "text/html")
		w.Write([]byte(fmt.Sprintf("<html>\n  <body>\n    <h1>Welcome, Chirpy Admin</h1>\n    <p>Chirpy has been visited %d times!</p>\n  </body>\n</html>", server.Api.fileserverHits.Load())))
	})
}

func defaultChirpyServer(db *sql.DB, platform string, jwtSecret string) *ChirpyServer {
	chirpy := newChirpyServer("8080")
	chirpy.Api.dbQueries = database.New(db)
	chirpy.Api.platform = platform
	chirpy.Api.jwtSecret = jwtSecret
	chirpy.registerHandler("GET /api/healthz", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Add("content-type", "text/plain")
		w.Write([]byte("OK"))
	}))
	chirpy.registerHandler(
		"/app/",
		chirpy.Api.middlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir(".")))))
	chirpy.registerHandler("GET /admin/metrics", getAdminMetrics(chirpy))
	chirpy.registerHandler("POST /admin/reset", chirpy.Api.metricsReset())
	chirpy.registerHandler("POST /api/chirps", chirpPostHandler(chirpy))
	chirpy.registerHandler("POST /api/users", handleNewUser(chirpy))
	chirpy.registerHandler("GET /api/chirps", getAllChirps(chirpy))
	chirpy.registerHandler("GET /api/chirps/{id}", getChirp(chirpy))
	chirpy.registerHandler("POST /api/login", loginUser(chirpy))
	return chirpy
}
