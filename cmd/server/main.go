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
	roomRepo := repository.NewRoomRepository(database)

	tokenService := service.NewTokenService(cfg.JWT)
	authService := service.NewAuthService(userRepo, tokenService)
	roomService := service.NewRoomService(roomRepo)

	authHandler := handler.NewAuthHandler(authService)
	userHandler := handler.NewUserHandler()
	roomHandler := handler.NewRoomHandler(roomService)

	authMiddleware := handler.AuthMiddleware(tokenService)

	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/auth/register", authHandler.Register)
	mux.HandleFunc("POST /api/auth/login", authHandler.Login)

	mux.Handle("GET /api/users/me", authMiddleware(http.HandlerFunc(userHandler.Me)))

	mux.Handle("POST /api/rooms", authMiddleware(http.HandlerFunc(roomHandler.Create)))
	mux.Handle("GET /api/rooms", authMiddleware(http.HandlerFunc(roomHandler.ListPublic)))
	mux.Handle("GET /api/rooms/my", authMiddleware(http.HandlerFunc(roomHandler.ListMy)))
	mux.Handle("GET /api/rooms/{id}", authMiddleware(http.HandlerFunc(roomHandler.GetRoomByID)))
	mux.Handle("POST /api/rooms/{id}/join", authMiddleware(http.HandlerFunc(roomHandler.Join)))

	addr := fmt.Sprintf(":%s", cfg.App.Port)
	log.Printf("starting server on %s (env: %s)", addr, cfg.App.Env)

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server error: %s", err)
	}
}
