package main

import (
	"database/sql"
	"log"
	"os"

	"github.com/dhruv15803/internal/storage"
	_ "github.com/lib/pq"
)

func main() {
	db_conn := os.Getenv("DB_CONN")
	port := os.Getenv("PORT")
	db, err := sql.Open("postgres", db_conn)
	if err != nil {
		log.Fatalf("DB CONNECTION OPENING FAILED :- %v", err.Error())
	}

	if err = db.Ping(); err != nil {
		log.Fatalf("DB CONNECTION FAILED:- %v", err.Error())
	}

	log.Println("DB CONNECTION SUCCESSFULL")
	storage := storage.NewStorage(db)
	server := NewAPIServer(port, storage)

	if err = server.Run(); err != nil {
		log.Fatalf("server failed to start :- %v", err.Error())
	}
}
