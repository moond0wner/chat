package main

import (
	"context"
	"os/signal"
	"syscall"

	core_transport "github.com/moond0wner/chat/internal/core/transport"
	"github.com/moond0wner/chat/internal/features/handlers/tcp"
	"github.com/moond0wner/chat/internal/features/handlers/ws"
	"go.uber.org/zap"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	init := Initialization(ctx)
	defer init.Logger.Close()
	defer init.PostgresPool.Close()

	tcp_server := tcp.NewServer(init.Logger, init.RoomService, init.ClientService, init.Cfg, init.Manager)
	websocket_server := ws.NewServer(init.Logger, init.RoomService, init.ClientService, init.Manager)

	manager := core_transport.NewManager(init.Logger, tcp_server, websocket_server)
	if err := manager.StartAll(ctx); err != nil {
		init.Logger.Error("Server error", zap.Error(err))
	}
}
