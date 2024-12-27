package main

import (
	"net/http"
	"time"

	"github.com/dhruv15803/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
)

type APIServer struct {
	addr    string
	storage *storage.Storage
}

func NewAPIServer(addr string, storage *storage.Storage) *APIServer {
	return &APIServer{
		addr:    addr,
		storage: storage,
	}
}

func (s *APIServer) Run() error {
	router := chi.NewRouter()

	corsOptions := cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173"}, // Add your frontend URLs here
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true, // Enable cookies or credentials if needed
		MaxAge:           300,  // Cache preflight requests for 5 minutes
	}

	router.Use(cors.Handler(corsOptions))

	router.Route("/api/v1", func(r chi.Router) {

		r.Get("/test", s.testHandler)

		r.Route("/user", func(r chi.Router) {
			r.Post("/register", s.registerUserHandler)
			r.Post("/login", s.loginUserHandler)

			r.Group(func(r chi.Router) {
				r.Use(s.AuthMiddleware)
				r.Get("/authenticated", s.getAuthenticatedUser)
				r.Get("/logout", s.logoutHandler)
			})
		})

		r.Route("/form", func(r chi.Router) {
			r.Use(s.AuthMiddleware)
			r.Post("/", s.createForm)
			r.Get("/", s.getAllForms)
			r.Get("/my-forms", s.myForms)
			r.Get("/{formId}", s.getFormWithFields)
			r.Delete("/{formId}", s.deleteFormHandler)
			r.Route("/fields", func(r chi.Router) {
				r.Use(s.AuthMiddleware)
				r.Post("/", s.createFormField)
				r.Delete("/{fieldId}", s.deleteFormField)
				r.Put("/{fieldId}", s.updateFormField)
			})
		})

		r.Route("/form-responses", func(r chi.Router) {
			r.Use(s.AuthMiddleware)
			r.Post("/", s.createFormResponse)
			r.Get("/{formId}", s.getFormResponses)
			r.Get("/", s.getMyResponses) // get authenticated user's responses to form's he/she has responded to
			r.Get("/response-fields/{formResponseId}", s.getResponseFields)
		})

	})

	server := http.Server{
		Addr:         s.addr,
		Handler:      router,
		ReadTimeout:  time.Second * 15,
		WriteTimeout: time.Second * 15,
	}

	return server.ListenAndServe()
}

func (s *APIServer) testHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("test handler working"))
}
