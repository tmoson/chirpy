package main

import (
	"fmt"
	"regexp"

	"encoding/json"
	"net/http"
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

func chirpPostHandler() http.Handler {
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
		valid := CleanedChirp{
			CleanedBody: sensorPost(body.Body),
		}
		respBody, err := json.Marshal(valid)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write(respBody)
	})
}

func defaultChirpyServer() *ChirpyServer {
	chirpy := newChirpyServer("8080")
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
	chirpy.registerHandler("POST /api/validate_chirp", chirpPostHandler())
	return chirpy
}
