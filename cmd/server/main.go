package main

import (
	"fmt"
	"go-chat/internal/config"
	"go-chat/internal/db"
	"log"
	"net/http"
)

func main() {
	cfg := config.MustLoad()

	database, err := db.NewPostgres(cfg.DB)
	if err != nil {
		log.Fatalf("db init: %s", err)
	}
	defer database.Close()
	log.Println("connected to postgres")

	addr := fmt.Sprintf(":%s", cfg.App.Port)
	log.Printf("starting server on %s (env: %s)", addr, cfg.App.Env)

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("server error: %s", err)
	}
}
