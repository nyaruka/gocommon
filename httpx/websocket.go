package httpx

import (
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// max time for between reading a message before socket is considered closed
	maxReadWait = 60 * time.Second

	// maximum time to wait for message to be written
	maxWriteWait = 15 * time.Second

	// how often to send a ping message
	pingPeriod = 30 * time.Second

	// maximum time to wait for writer to drain when closing
	drainPeriod = 3 * time.Second
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,

	// responsibility of caller to enforce origin rules
	CheckOrigin: func(r *http.Request) bool { return true },
}

// WebSocket provides a websocket interface similar to that of Javascript.
type WebSocket interface {
	// Start begins reading and writing of messages on this socket
	Start()

	// Send sends the given message over the socket
	Send([]byte)

	// Close closes the socket connection
	Close(int)

	// OnMessage is called when the socket receives a message
	OnMessage(func([]byte))

	// OnClose is called when the socket is closed (even if we initiate the close)
	OnClose(func(int))
}

type message struct {
	type_ int
	data  []byte
}

// WebSocket implemention using gorilla library
type socket struct {
	conn   *websocket.Conn
	outbox chan message

	readError       chan error
	writeError      chan error
	shutdown        chan bool
	stopWriter      chan bool
	closingWithCode int

	readerWaitGroup  sync.WaitGroup
	writerWaitGroup  sync.WaitGroup
	monitorWaitGroup sync.WaitGroup

	onMessage func([]byte)
	onClose   func(int)
}

// NewWebSocket creates a new web socket from a regular HTTP request
func NewWebSocket(w http.ResponseWriter, r *http.Request, maxReadBytes int64, sendBuffer int) (WebSocket, error) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return nil, err
	}

	conn.SetReadLimit(maxReadBytes)

	return &socket{
		conn:   conn,
		outbox: make(chan message, sendBuffer),

		readError:  make(chan error, 1),
		writeError: make(chan error, 1),
		shutdown:   make(chan bool, 1),
		stopWriter: make(chan bool),

		onMessage: defaultOnMessage,
		onClose:   defaultOnClose,
	}, nil
}

func (s *socket) OnMessage(fn func([]byte)) { s.onMessage = fn }
func (s *socket) OnClose(fn func(int))      { s.onClose = fn }

func (s *socket) Start() {
	s.conn.SetReadDeadline(time.Now().Add(maxReadWait))
	s.conn.SetPongHandler(s.pong)

	go s.monitor()
	go s.reader()
	go s.writer()
}

func (s *socket) Send(msg []byte) {
	if s.closingWithCode != 0 {
		panic("can't send to socket which is closed or closing")
	}

	s.outbox <- message{type_: websocket.TextMessage, data: msg}
}

func (s *socket) Close(code int) {
	s.closingWithCode = code

	s.outbox <- message{type_: websocket.CloseMessage, data: websocket.FormatCloseMessage(code, "")}

	s.shutdown <- true

	s.monitorWaitGroup.Wait()
}

func (s *socket) pong(m string) error {
	s.conn.SetReadDeadline(time.Now().Add(maxReadWait))

	return nil
}

func (s *socket) monitor() {
	s.monitorWaitGroup.Add(1)
	defer s.monitorWaitGroup.Done()

	// shutdown starts via read error, write error, or Close()
out:
	for {
		select {
		case err := <-s.readError:
			if e, ok := err.(*websocket.CloseError); ok && s.closingWithCode == 0 {
				s.closingWithCode = e.Code
			}
			break out
		case err := <-s.writeError:
			if e, ok := err.(*websocket.CloseError); ok && s.closingWithCode == 0 {
				s.closingWithCode = e.Code
			}
			break out
		case <-s.shutdown:
			break out
		}
	}

	// stop writer if not already finished...
	s.stopWriter <- true
	s.writerWaitGroup.Wait()

	// stop reader if not already finished...
	s.conn.Close()
	s.readerWaitGroup.Wait()

	s.onClose(s.closingWithCode)
}

func (s *socket) reader() {
	s.readerWaitGroup.Add(1)
	defer s.readerWaitGroup.Done()

	for {
		_, message, err := s.conn.ReadMessage()
		if err != nil {
			s.readError <- err
			return
		}

		s.onMessage(message)
	}
}

func (s *socket) writer() {
	s.writerWaitGroup.Add(1)
	defer s.writerWaitGroup.Done()

	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()

out:
	for {
		select {
		case msg := <-s.outbox:
			s.conn.SetWriteDeadline(time.Now().Add(maxWriteWait))

			if err := s.conn.WriteMessage(msg.type_, msg.data); err != nil {
				s.writeError <- err
			}
		case <-ticker.C:
			s.conn.SetWriteDeadline(time.Now().Add(maxWriteWait))

			if err := s.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				s.writeError <- err
			}
		case <-s.stopWriter:
			break out
		}
	}

	// try to drain the outbox with a time limit
	if len(s.outbox) > 0 {
		s.conn.SetWriteDeadline(time.Now().Add(drainPeriod))
		for {
			select {
			case msg := <-s.outbox:
				err := s.conn.WriteMessage(msg.type_, msg.data)
				if err != nil || len(s.outbox) == 0 {
					return
				}
			case <-time.After(drainPeriod):
				return
			}
		}
	}
}

func defaultOnMessage([]byte) {}

func defaultOnClose(int) {}
