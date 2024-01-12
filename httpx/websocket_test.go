package httpx_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/nyaruka/gocommon/httpx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newSocketServer(t *testing.T, fn func(httpx.WebSocket)) string {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sock, err := httpx.NewWebSocket(w, r, 4096, 5)
		require.NoError(t, err)

		fn(sock)
	}))

	return "ws:" + strings.TrimPrefix(s.URL, "http:")
}

func newSocketConnection(t *testing.T, url string) *websocket.Conn {
	d := websocket.Dialer{
		Subprotocols:     []string{"p1", "p2"},
		ReadBufferSize:   1024,
		WriteBufferSize:  1024,
		HandshakeTimeout: 30 * time.Second,
	}
	c, _, err := d.Dial(url, nil)
	assert.NoError(t, err)
	return c
}

func TestSocketMessages(t *testing.T) {
	var sock httpx.WebSocket
	var serverReceived [][]byte
	var serverCloseCode int

	serverURL := newSocketServer(t, func(ws httpx.WebSocket) {
		sock = ws
		sock.OnMessage(func(b []byte) {
			serverReceived = append(serverReceived, b)
		})
		sock.OnClose(func(code int) {
			serverCloseCode = code
		})
		sock.Start()
	})

	conn := newSocketConnection(t, serverURL)

	// send a message from the server...
	sock.Send([]byte("from server"))

	// and read it from the client
	msgType, msg, err := conn.ReadMessage()
	assert.NoError(t, err)
	assert.Equal(t, 1, msgType)
	assert.Equal(t, "from server", string(msg))

	// send a message from the client...
	conn.WriteMessage(websocket.TextMessage, []byte("to server"))

	// and check server received it
	time.Sleep(500 * time.Millisecond)
	assert.Equal(t, [][]byte{[]byte("to server")}, serverReceived)

	pongReceived := false
	conn.SetPongHandler(func(appData string) error {
		pongReceived = true
		return nil
	})

	// send a ping message from the client...
	conn.WriteMessage(websocket.PingMessage, []byte{})

	// and give server time to receive it and respond
	time.Sleep(500 * time.Millisecond)

	// give the connection something to read because ReadMessage will block until it gets a non-ping-pong message
	sock.Send([]byte("dummy"))
	conn.ReadMessage()

	assert.True(t, pongReceived)

	var connCloseCode int
	conn.SetCloseHandler(func(code int, text string) error {
		connCloseCode = code
		return nil
	})

	sock.Close(1001)

	conn.ReadMessage() // read the close message

	assert.Equal(t, 1001, serverCloseCode)
	assert.Equal(t, 1001, connCloseCode)
}

func TestSocketClientCloseWithMessage(t *testing.T) {
	var sock httpx.WebSocket
	var serverCloseCode int

	serverURL := newSocketServer(t, func(ws httpx.WebSocket) {
		sock = ws
		sock.OnClose(func(code int) { serverCloseCode = code })
		sock.Start()
	})

	conn := newSocketConnection(t, serverURL)
	conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.ClosePolicyViolation, ""))
	conn.Close()

	time.Sleep(250 * time.Millisecond)

	assert.Equal(t, websocket.ClosePolicyViolation, serverCloseCode)
}

func TestSocketClientCloseWithoutMessage(t *testing.T) {
	var sock httpx.WebSocket
	var serverCloseCode int

	serverURL := newSocketServer(t, func(ws httpx.WebSocket) {
		sock = ws
		sock.OnClose(func(code int) { serverCloseCode = code })
		sock.Start()
	})

	conn := newSocketConnection(t, serverURL)
	conn.Close()

	time.Sleep(250 * time.Millisecond)

	assert.Equal(t, websocket.CloseAbnormalClosure, serverCloseCode)
}
