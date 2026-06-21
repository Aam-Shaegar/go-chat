package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	core_config "go-chat/internal/core/config"
	core_logger "go-chat/internal/core/logger"
	core_pgx_pool "go-chat/internal/core/repository/postgres/pool/pgx"
	core_http_middleware "go-chat/internal/core/transport/http/middleware"
	core_http_server "go-chat/internal/core/transport/http/server"

	dm_repository_postgres "go-chat/internal/features/dm/repository/postgres"
	dm_service "go-chat/internal/features/dm/service"
	dm_transport_http "go-chat/internal/features/dm/transport/http"
	jwt_repository_postgres "go-chat/internal/features/jwt/repository/postgres"
	jwt_service "go-chat/internal/features/jwt/service"
	jwt_transport_http "go-chat/internal/features/jwt/transport/http"
	messages_repository_postgres "go-chat/internal/features/messages/repository/postgres"
	messages_service "go-chat/internal/features/messages/service"
	messages_transport_http "go-chat/internal/features/messages/transport/http"
	reads_repository_postgres "go-chat/internal/features/reads/repository/postgres"
	reads_service "go-chat/internal/features/reads/service"
	reads_transport_http "go-chat/internal/features/reads/transport/http"

	users_repository_postgres "go-chat/internal/features/users/repository/postgres"
	users_service "go-chat/internal/features/users/service"
	users_transport_http "go-chat/internal/features/users/transport/http"

	rooms_repository_postgres "go-chat/internal/features/rooms/repository/postgres"
	rooms_service "go-chat/internal/features/rooms/service"
	rooms_transport_http "go-chat/internal/features/rooms/transport/http"

	ws_hub "go-chat/internal/features/ws/hub"
	ws_repository_postgres "go-chat/internal/features/ws/repository/postgres"
	ws_service "go-chat/internal/features/ws/service"
	ws_transport_http "go-chat/internal/features/ws/transport/http"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

func main() {
	cfg := core_config.NewConfigMust()
	time.Local = cfg.TimeZone

	ctx, cancel := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT, syscall.SIGTERM,
	)
	defer cancel()

	fmt.Println("GoChat app starting")

	logger, err := core_logger.NewLogger(core_logger.NewConfigMust())
	if err != nil {
		fmt.Println("failed to init application logger:", err)
		os.Exit(1)
	}
	defer logger.Close()

	logger.Debug("application time zone", zap.Any("zone", time.Local))

	// Postgres
	logger.Debug("initializing postgres connection pool")
	pool, err := core_pgx_pool.NewConnectionPool(core_pgx_pool.NewConfigMust(), ctx)
	if err != nil {
		logger.Fatal("failed to init postgres connection pool", zap.Error(err))
	}
	defer pool.Close()

	// Redis
	logger.Debug("initializing redis client")
	redisClient := redis.NewClient(&redis.Options{
		Addr: cfg.RedisAddr,
	})
	if err := redisClient.Ping(ctx).Err(); err != nil {
		logger.Fatal("failed to ping redis", zap.Error(err))
	}
	defer redisClient.Close()

	// Repositories
	jwtRepo := jwt_repository_postgres.NewJwtRepository(pool)
	usersRepo := users_repository_postgres.NewUsersRepository(pool)
	roomsRepo := rooms_repository_postgres.NewRoomsRepository(pool)
	wsRepo := ws_repository_postgres.NewWSRepository(pool)
	messagesRepo := messages_repository_postgres.NewMessagesRepository(pool)
	readsRepo := reads_repository_postgres.NewReadsRepository(pool)
	dmRepo := dm_repository_postgres.NewDMRepository(pool)

	// Services
	jwtSvc := jwt_service.NewJwtService(jwtRepo, usersRepo, cfg)
	usersSvc := users_service.NewUsersService(usersRepo, jwtSvc)
	roomsSvc := rooms_service.NewRoomsService(roomsRepo)
	messagesSvc := messages_service.NewMessagesService(messagesRepo, roomsRepo)
	readSvc := reads_service.NewReadsService(readsRepo, roomsRepo)
	dmSvc := dm_service.NewDMService(dmRepo, usersRepo)

	// WebSocket Hub
	hub := ws_hub.NewHub(redisClient, logger)
	go hub.Run(ctx)

	wsSvc := ws_service.NewWSService(wsRepo, hub)

	// HTTP Handlers
	jwtHandler := jwt_transport_http.NewJwtHTTPHandler(jwtSvc, cfg.JwtRefreshTTL)
	usersHandler := users_transport_http.NewUsersHTTPHandler(usersSvc, cfg)
	roomsHandler := rooms_transport_http.NewRoomsHandler(roomsSvc)
	wsHandler := ws_transport_http.NewWSHandler(wsSvc, hub, roomsRepo)
	messagesHandler := messages_transport_http.NewMessagesHandler(messagesSvc)
	readsHandler := reads_transport_http.NewReadsHandler(readSvc)
	dmHandler := dm_transport_http.NewDMHandler(dmSvc)

	// Auth middleware
	authMiddleware := core_http_middleware.Auth(jwtSvc)

	// Router
	apiRouter := core_http_server.NewAPIVersionRouter(core_http_server.ApiVersion1)

	// Публичные роуты
	publicRoutes := jwtHandler.Routes()
	for _, route := range usersHandler.Routes() {
		if route.Path == "/auth/register" || route.Path == "/auth/login" {
			publicRoutes = append(publicRoutes, route)
		}
	}
	apiRouter.RegisterRoutes(publicRoutes...)

	// Защищённые роуты users
	for _, route := range usersHandler.Routes() {
		if route.Path != "/auth/register" && route.Path != "/auth/login" {
			route.Middleware = append(route.Middleware, authMiddleware)
			apiRouter.RegisterRoutes(route)
		}
	}

	// Rooms роуты (все защищённые — middleware внутри Routes())
	apiRouter.RegisterRoutes(roomsHandler.Routes(authMiddleware)...)

	// Message роуты (защищённые middleware)
	apiRouter.RegisterRoutes(messagesHandler.Routes(authMiddleware)...)

	// WS роуты (middleware внутри Routes())
	apiRouter.RegisterRoutes(wsHandler.Routes(authMiddleware)...)

	// Reads роуты (защищённые)
	apiRouter.RegisterRoutes(readsHandler.Routes(authMiddleware)...)

	// dm роуты (защищённые)
	apiRouter.RegisterRoutes(dmHandler.Routes(authMiddleware)...)

	// HTTP Server
	logger.Debug("initializing http server")
	httpServer := core_http_server.NewHTTPServer(
		core_http_server.NewConfigMust(),
		logger,
		core_http_middleware.RequestID(),
		core_http_middleware.Logger(logger),
		core_http_middleware.Trace(),
		core_http_middleware.Panic(),
	)
	httpServer.RegisterAPIRouters(apiRouter)

	if err := httpServer.Run(ctx); err != nil {
		logger.Error("HTTP server run error", zap.Error(err))
	}

	//Фоновая горутина с очисткой старых токенов
	go jwtRepo.StartCleanup(ctx, time.Hour, logger)

	// Graceful shutdown WS
	logger.Debug("shutting down websocket hub")
	hub.Shutdown()
	logger.Debug("shutdown complete")
}
