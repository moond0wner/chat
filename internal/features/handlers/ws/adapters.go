package ws

import (
	"net"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type WebSocketConnAdapter struct {
	conn *websocket.Conn
	mtx  sync.RWMutex
}

func NewWebSocketConnAdapter(conn *websocket.Conn) *WebSocketConnAdapter {
	return &WebSocketConnAdapter{conn: conn}
}

func (w *WebSocketConnAdapter) Read(b []byte) (int, error) {
	w.mtx.RLock()
	defer w.mtx.RUnlock()
	_, msg, err := w.conn.ReadMessage()
	if err != nil {
		return 0, err
	}
	n := copy(b, msg)
	return n, nil
}

func (w *WebSocketConnAdapter) Write(b []byte) (int, error) {
	w.mtx.Lock()
	defer w.mtx.Unlock()

	err := w.conn.WriteMessage(websocket.TextMessage, b)
	if err != nil {
		return 0, err
	}
	return len(b), nil
}

func (w *WebSocketConnAdapter) Close() error {
	w.mtx.Lock()
	defer w.mtx.Unlock()
	return w.conn.Close()
}

func (w *WebSocketConnAdapter) LocalAddr() net.Addr {
	w.mtx.RLock()
	defer w.mtx.RUnlock()
	return w.conn.LocalAddr()
}

func (w *WebSocketConnAdapter) RemoteAddr() net.Addr {
	w.mtx.Lock()
	defer w.mtx.Unlock()
	return w.conn.RemoteAddr()
}

func (w *WebSocketConnAdapter) SetWriteDeadline(t time.Time) error {
	w.mtx.Lock()
	defer w.mtx.Unlock()
	return nil
}
