package main

import (
	"fmt"
	"go-chat/internal/config"
	"go-chat/internal/db"
	"go-chat/internal/handler"
	"go-chat/internal/repository"
	"go-chat/internal/service"
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

	userRepo := repository.NewUserRepository(database)
	tokenService := service.NewTokenService(cfg.JWT)
	authService := service.NewAuthService(userRepo, tokenService)

	authHandler := handler.NewAuthHandler(authService)
	userHandler := handler.NewUserHandler()

	authMiddleware := handler.AuthMiddleware(tokenService)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/auth/register", authHandler.Register)
	mux.HandleFunc("POST /api/auth/login", authHandler.Login)

	mux.Handle("GET /api/users/me", authMiddleware(http.HandlerFunc(userHandler.Me)))

	addr := fmt.Sprintf(":%s", cfg.App.Port)
	log.Printf("starting server on %s (env: %s)", addr, cfg.App.Env)

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server error: %s", err)
	}
}
