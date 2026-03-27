package main

import (
	"fmt"
	"go-chat/internal/config"
	"go-chat/internal/db"
	"go-chat/internal/handler"
	"go-chat/internal/repository"
	"go-chat/internal/service"
	"go-chat/internal/ws"
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
	messageRepo := repository.NewMessageRepository(database)

	tokenService := service.NewTokenService(cfg.JWT)
	authService := service.NewAuthService(userRepo, tokenService)
	roomService := service.NewRoomService(roomRepo)

	hub := ws.NewHub(messageRepo, userRepo, roomService)
	go hub.Run()

	authHandler := handler.NewAuthHandler(authService)
	userHandler := handler.NewUserHandler()
	roomHandler := handler.NewRoomHandler(roomService, messageRepo)
	wsHandler := handler.NewWSHandler(handler.WSHandlerDeps{
		Hub:         hub,
		RoomService: roomService,
	})

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
	mux.Handle("GET /api/rooms/{id}/messages", authMiddleware(http.HandlerFunc(roomHandler.GetMessages)))

	mux.Handle("/ws/rooms/{id}", authMiddleware(http.HandlerFunc(wsHandler.ServeWS)))

	addr := fmt.Sprintf(":%s", cfg.App.Port)
	log.Printf("starting server on %s (env: %s)", addr, cfg.App.Env)

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server error: %s", err)
	}
}
