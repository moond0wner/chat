# tcp_server

Сырая версия TCP-чата на Go.

сервер: `make run`
клиент:`nc localhost 8080`

функционал:
- Подключение к general и создание своих комнат (`/join`)
- Выход из комнаты в general (`/leave`)
- Смена ника (`/nick`)