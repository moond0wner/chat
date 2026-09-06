```markdown
# Chat

Многопользовательский чат с поддержкой TCP и WebSocket.

## Быстрый старт

```bash
cp .env.example .env
make env-up
make env-port-forward
make migrate-up
make app-run
```

## Подключение

```bash
nc localhost 8080                  # TCP
oткрыть web/index.html в браузере  # WebSocket
```

## Команды

```
/reg <ник>            регистрация
/join <комната>       войти в комнату
/leave                вернуться в general
/nick <ник>           сменить ник
/msg <ник> <текст>    личное сообщение
/info                 участники в комнате
/all_info             все комнаты
```

## Структура

```
core/          интерфейсы и модели
features/      бизнес-логика и транспорт
cmd/           точка входа
```

## Стек

Go, PostgreSQL, gorilla/websocket, Docker
```
