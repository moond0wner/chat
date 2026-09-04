package main

// Планы на следующее обновление:
// 1) Перелопатить весь код, убрать всё лишнее, ну и линтер добавить еще
// 2) Фикс истории сообщений (логи не должны попадать в бд)
// 3) WebSocket
// 4) HTML+JS

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	core_config "tcp_srv/internal/core/config"
	core_logger "tcp_srv/internal/core/logger"
	core_pgx_pool "tcp_srv/internal/core/repository/postgres/pool/pgx"
	"tcp_srv/internal/features/handlers/tcp"
	history_postgres_repository "tcp_srv/internal/features/repostitory/postgres"
	client_service "tcp_srv/internal/features/services/client"
	room_service "tcp_srv/internal/features/services/room"
	"time"

	"go.uber.org/zap"
)

func main() {
	var err error

	cfg := core_config.NewConfigMust()
	time.Local = cfg.TimeZone

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	logger, err := core_logger.NewLogger(core_logger.NewConfigMust())
	if err != nil {
		fmt.Println("Failed to init applitcation logger:", err)
		os.Exit(1)
	}
	defer logger.Close()

	logger.Debug("application time zone", zap.Any("zone", time.Local))

	logger.Debug("initializing postgres connection pool")

	postgresPool, err := core_pgx_pool.NewPool(
		ctx,
		core_pgx_pool.NewConfigMust(),
	)

	if err != nil {
		logger.Fatal("failed to init postgres connection pool", zap.Error(err))
	}
	defer postgresPool.Close()

	historyRepository := history_postgres_repository.NewHistoryRepository(postgresPool)
	roomService := room_service.NewRoomService(logger, historyRepository)
	clientService := client_service.NewClientService(logger, historyRepository)

	if err = roomService.LoadAllRooms(ctx); err != nil {
		logger.Fatal("Ошибка загрузки комнат", zap.Error(err))
	}
	if err = clientService.LoadAllUsers(ctx); err != nil {
		logger.Fatal("Ошибка загрузки пользователей", zap.Error(err))
	}

	if err = roomService.CreateRoom(ctx, "register"); err != nil {
		logger.Warn("Ошибка создания комнаты register", zap.Error(err))
	}
	if err = roomService.CreateRoom(ctx, "general"); err != nil {
		logger.Warn("Ошибка создания комнаты general", zap.Error(err))
	}

	if err = roomService.InitSystemRooms(ctx); err != nil {
		logger.Warn("Ошибка инициализации ID системных комнат", zap.Error(err))
	}

	srv := tcp.NewServer(logger, roomService, clientService, cfg)
	if err = srv.Start(ctx); err != nil {
		logger.Error("Error starting server:", zap.Error(err))
	}
}
