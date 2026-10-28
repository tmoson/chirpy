package main

import (
	"database/sql"
	"fmt"
	"regexp"

	"encoding/json"
	"net/http"

	"github.com/google/uuid"
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
	Body   string    `json:"body"`
	UserID uuid.UUID `json:"user_id"`
}

type NewUserRequest struct {
	Email string `json:"email"`
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
		body := DefaultRequestBody{}
		decoder := json.NewDecoder(r.Body)
		err := decoder.Decode(&body)
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
			UserID: body.UserID,
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
		newUser, err := server.Api.dbQueries.CreateUser(r.Context(), newUserBody.Email)
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

func defaultChirpyServer(db *sql.DB, platform string) *ChirpyServer {
	chirpy := newChirpyServer("8080")
	chirpy.Api.dbQueries = database.New(db)
	chirpy.Api.platform = platform
	chirpy.registerHandlerFunction("GET /api/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Add("content-type", "text/plain")
		w.Write([]byte("OK"))
	})
	chirpy.registerHandler(
		"/app/",
		chirpy.Api.middlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir(".")))))
	chirpy.registerHandlerFunction("GET /admin/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Add("content-type", "text/html")
		w.Write([]byte(fmt.Sprintf("<html>\n  <body>\n    <h1>Welcome, Chirpy Admin</h1>\n    <p>Chirpy has been visited %d times!</p>\n  </body>\n</html>", chirpy.Api.fileserverHits.Load())))
	})
	chirpy.registerHandler("POST /admin/reset", chirpy.Api.metricsReset())
	chirpy.registerHandler("POST /api/chirps", chirpPostHandler(chirpy))
	chirpy.registerHandler("POST /api/users", handleNewUser(chirpy))
	chirpy.registerHandler("GET /api/chirps", getAllChirps(chirpy))
	chirpy.registerHandler("GET /api/chirps/{id}", getChirp(chirpy))
	return chirpy
}
