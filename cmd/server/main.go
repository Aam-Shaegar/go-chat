package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"go-chat/internal/config"
	"go-chat/internal/db"
	"go-chat/internal/handler"
	"go-chat/internal/repository"
	"go-chat/internal/service"
	"go-chat/internal/ws"
	_ "net/http/pprof"
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
	inviteRepo := repository.NewInviteRepository(database)
	dmRepo := repository.NewDMRepository(database)
	readsRepo := repository.NewReadsRepository(database)

	tokenService := service.NewTokenService(cfg.JWT)
	authService := service.NewAuthService(userRepo, tokenService)
	roomService := service.NewRoomService(roomRepo)
	messageService := service.NewMessageService(messageRepo, roomRepo)
	inviteService := service.NewInviteService(inviteRepo, roomRepo)
	dmService := service.NewDMService(dmRepo, userRepo)

	hub := ws.NewHub(messageRepo, userRepo, roomService)
	go hub.Run()

	authHandler := handler.NewAuthHandler(authService)
	userHandler := handler.NewUserHandler(userRepo)
	roomHandler := handler.NewRoomHandler(roomService, messageRepo)
	messageHandler := handler.NewMessageHandler(messageService, hub)
	inviteHandler := handler.NewInviteHandler(inviteService)
	dmHandler := handler.NewDMHandler(dmService, hub)
	readsHandler := handler.NewReadsHandler(readsRepo)
	wsHandler := handler.NewWSHandler(handler.WSHandlerDeps{
		Hub:          hub,
		RoomService:  roomService,
		TokenService: tokenService,
	})

	authMiddleware := handler.AuthMiddleware(tokenService)

	go func() {
		log.Println("starting pprof on :6060")
		if err := http.ListenAndServe("localhost:6060", nil); err != nil {
			log.Printf("pprof error: %v", err)
		}
	}()

	mux := http.NewServeMux()

	//Auth
	mux.HandleFunc("POST /api/auth/register", authHandler.Register)
	mux.HandleFunc("POST /api/auth/login", authHandler.Login)
	mux.HandleFunc("POST /api/auth/refresh", authHandler.Refresh)

	//users
	mux.Handle("GET /api/users/me", authMiddleware(http.HandlerFunc(userHandler.Me)))
	mux.Handle("GET /api/users", authMiddleware(http.HandlerFunc(userHandler.ListAll)))

	//rooms
	mux.Handle("POST /api/rooms", authMiddleware(http.HandlerFunc(roomHandler.Create)))
	mux.Handle("GET /api/rooms", authMiddleware(http.HandlerFunc(roomHandler.ListPublic)))
	mux.Handle("GET /api/rooms/my", authMiddleware(http.HandlerFunc(roomHandler.ListMy)))
	mux.Handle("GET /api/rooms/{id}", authMiddleware(http.HandlerFunc(roomHandler.GetRoomByID)))
	mux.Handle("DELETE /api/rooms/{id}", authMiddleware(http.HandlerFunc(roomHandler.Delete)))
	mux.Handle("POST /api/rooms/{id}/join", authMiddleware(http.HandlerFunc(roomHandler.Join)))
	mux.Handle("POST /api/rooms/{id}/leave", authMiddleware(http.HandlerFunc(roomHandler.Leave)))
	mux.Handle("GET /api/rooms/{id}/messages", authMiddleware(http.HandlerFunc(roomHandler.GetMessages)))
	mux.Handle("GET /api/rooms/{id}/members", authMiddleware(http.HandlerFunc(roomHandler.ListMembers)))
	mux.Handle("DELETE /api/rooms/{id}/members/{userId}", authMiddleware(http.HandlerFunc(roomHandler.KickMember)))
	mux.Handle("POST /api/rooms/{id}/invites", authMiddleware(http.HandlerFunc(inviteHandler.Create)))
	mux.Handle("GET /api/rooms/{id}/invites", authMiddleware(http.HandlerFunc(inviteHandler.List)))

	//messages
	mux.Handle("DELETE /api/rooms/{id}/messages/{messageId}", authMiddleware(http.HandlerFunc(messageHandler.Delete)))

	//invites
	mux.Handle("GET /api/invites/{token}", authMiddleware(http.HandlerFunc(inviteHandler.GetInfo)))
	mux.Handle("POST /api/invites/{token}/accept", authMiddleware(http.HandlerFunc(inviteHandler.Accept)))
	mux.Handle("DELETE /api/invites/{token}", authMiddleware(http.HandlerFunc(inviteHandler.Delete)))

	//reads
	mux.Handle("POST /api/rooms/{id}/read", authMiddleware(http.HandlerFunc(readsHandler.Upsert)))
	mux.Handle("GET /api/rooms/{id}/read", authMiddleware(http.HandlerFunc(readsHandler.Get)))
	mux.Handle("GET /api/reads/unread", authMiddleware(http.HandlerFunc(readsHandler.GetUnreadCounts)))

	//dm
	mux.Handle("GET /api/dm", authMiddleware(http.HandlerFunc(dmHandler.GetConversations)))
	mux.Handle("GET /api/dm/{userId}/messages", authMiddleware(http.HandlerFunc(dmHandler.GetHistory)))
	mux.Handle("POST /api/dm/{userId}/messages", authMiddleware(http.HandlerFunc(dmHandler.Send)))
	mux.Handle("POST /api/dm/{userId}/read", authMiddleware(http.HandlerFunc(dmHandler.MarkRead)))
	mux.Handle("GET /api/dm/unread", authMiddleware(http.HandlerFunc(dmHandler.GetUnreadCounts)))
	mux.Handle("GET /api/dm/{userId}/read", authMiddleware(http.HandlerFunc(dmHandler.GetLastRead)))

	//websocket
	mux.HandleFunc("/ws/rooms/{id}", wsHandler.ServeWS)

	//static
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			fs := http.FileServer(http.Dir("static"))
			if _, err := os.Stat("static" + r.URL.Path); err == nil {
				fs.ServeHTTP(w, r)
				return
			}
		}
		http.ServeFile(w, r, "static/index.html")
	})

	addr := fmt.Sprintf(":%s", cfg.App.Port)
	log.Printf("starting server on %s (env: %s)", addr, cfg.App.Env)

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server error: %s", err)
	}
}
