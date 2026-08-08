package main

// 1. Фикс реализации graceful shutdown (выполнено)
// 2. Добавление команд через мапу, а не switch-case (выполнено)
// 3. Вынести части конфигурации в .env/yaml/json
// . Авторизация (сначала вход/регистрация в register канале, потом попадание в general)
// 4. Приватные сообщения
// . Роли и права
// 5. Сохрание состояния в БД (комнаты)
// 6. Создание команты с собственными настройками (лимит, пароль, приватность)
// 7. Реализация админки (бан по IP/ID), проверка по JWT-аутентификации
// .WebSocket вебчат вместо netcat
// 8. TLS/SSL (зашифрованное соединениие)
// .Форматирование: Markdown, подсветка системных уведомлений разными цветами
// .REST-API
// .Reconnect
// .Docker контейнеризация
import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	core_logger "tcp_srv/internal/core/logger"
	"tcp_srv/internal/features/handlers/tcp"
	"tcp_srv/internal/features/services"

	"go.uber.org/zap"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	logger, err := core_logger.NewLogger(core_logger.NewConfigMust())
	if err != nil {
		fmt.Println("Failed to init applitcation logger:", err)
		os.Exit(1)
	}
	defer logger.Close()

	roomService := services.NewRoomService(logger)
	clientService := services.NewClientService(logger)

	srv := tcp.NewServer(logger, roomService, clientService)
	if err := srv.Start(ctx); err != nil {
		logger.Error("Error starting server:", zap.Error(err))
	}
}
