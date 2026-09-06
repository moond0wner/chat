package main

import (
	"context"
	"fmt"
	"os"
	"time"

	core_command "github.com/moond0wner/chat/internal/core/command"
	core_config "github.com/moond0wner/chat/internal/core/config"
	core_logger "github.com/moond0wner/chat/internal/core/logger"
	core_pgx_pool "github.com/moond0wner/chat/internal/core/repository/postgres/pool/pgx"
	history_postgres_repository "github.com/moond0wner/chat/internal/features/repostitory/postgres"
	client_service "github.com/moond0wner/chat/internal/features/services/client"
	room_service "github.com/moond0wner/chat/internal/features/services/room"
	"go.uber.org/zap"
)

type Init struct {
	Cfg               *core_config.Config
	Logger            *core_logger.Logger
	Manager           *core_command.Manager
	PostgresPool      *core_pgx_pool.Pool
	HistoryRepository *history_postgres_repository.HistoryRepository
	RoomService       *room_service.RoomService
	ClientService     *client_service.ClientService
}

func Initialization(ctx context.Context) *Init {
	var err error

	cfg := core_config.NewConfigMust()
	time.Local = cfg.TimeZone

	logger, err := core_logger.NewLogger(core_logger.NewConfigMust())
	if err != nil {
		fmt.Println("Failed to init applitcation logger:", err)
		os.Exit(1)
	}

	logger.Debug("application time zone", zap.Any("zone", time.Local))

	logger.Debug("initializing postgres connection pool")

	postgresPool, err := core_pgx_pool.NewPool(
		ctx,
		core_pgx_pool.NewConfigMust(),
	)

	if err != nil {
		logger.Fatal("failed to init postgres connection pool", zap.Error(err))
	}

	historyRepository := history_postgres_repository.NewHistoryRepository(postgresPool)
	roomService := room_service.NewRoomService(logger, historyRepository)
	clientService := client_service.NewClientService(logger, historyRepository)

	manager := core_command.RegisterCommands(ctx, logger, roomService, clientService)

	if err = roomService.LoadAllRooms(ctx); err != nil {
		logger.Error("Error load room", zap.Error(err))
	}
	if err = clientService.LoadAllUsers(ctx); err != nil {
		logger.Error("Error load users", zap.Error(err))
	}

	if err = roomService.CreateRoom(ctx, "register"); err != nil {
		logger.Warn("Error create room 'register'", zap.Error(err))
	}
	if err = roomService.CreateRoom(ctx, "general"); err != nil {
		logger.Warn("Error create room 'general'", zap.Error(err))
	}

	if err = roomService.InitSystemRooms(ctx); err != nil {
		logger.Warn("Error initialization ID system rooms", zap.Error(err))
	}

	return &Init{
		Cfg:               cfg,
		Logger:            logger,
		Manager:           manager,
		PostgresPool:      postgresPool,
		HistoryRepository: historyRepository,
		RoomService:       roomService,
		ClientService:     clientService,
	}
}
