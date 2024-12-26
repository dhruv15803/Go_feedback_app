package main

import (
	"net/http"
	"time"

	"github.com/dhruv15803/internal/storage"
	"github.com/go-chi/chi/v5"
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

	router.Route("/api/v1", func(r chi.Router) {

		r.Get("/test", s.testHandler)

		r.Route("/user", func(r chi.Router) {
			r.Post("/register", s.registerUserHandler)
			r.Post("/login", s.loginUserHandler)

			r.Group(func(r chi.Router) {
				r.Use(s.AuthMiddleware)
				r.Get("/authenticated", s.getAuthenticatedUser)
			})
		})

		r.Route("/form", func(r chi.Router) {
			r.Use(s.AuthMiddleware)
			r.Post("/", s.createForm)
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
