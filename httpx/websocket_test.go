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

func TestSocket(t *testing.T) {
	var sock httpx.WebSocket
	var err error

	var serverReceived [][]byte
	var serverCloseCode int

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sock, err = httpx.NewWebSocket(w, r, 4096, 5)

		sock.OnMessage(func(b []byte) {
			serverReceived = append(serverReceived, b)
		})
		sock.OnClose(func(code int) {
			serverCloseCode = code
		})

		sock.Start()

		require.NoError(t, err)
	}))

	wsURL := "ws:" + strings.TrimPrefix(server.URL, "http:")

	d := websocket.Dialer{
		Subprotocols:     []string{"p1", "p2"},
		ReadBufferSize:   1024,
		WriteBufferSize:  1024,
		HandshakeTimeout: 30 * time.Second,
	}
	conn, _, err := d.Dial(wsURL, nil)
	assert.NoError(t, err)
	assert.NotNil(t, conn)

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
	time.Sleep(time.Second)
	assert.Equal(t, [][]byte{[]byte("to server")}, serverReceived)

	sock.Close()

	assert.Equal(t, 1000, serverCloseCode)
}
