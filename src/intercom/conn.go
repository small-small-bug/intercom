package intercom

import (
	"encoding/json"
	"errors"
	"github.com/gorilla/websocket"
	"io"
	"log"
	"sync"
)

const (
	RegisterMessageType = 1
	NormalMessageType   = 255
)

type WSMessage struct {
	Kind int    `json:"Kind"`
	Body string `json:"Body"`
}

// Conn wraps websocket.Conn with Conn. It defines to listen and read
// data from Conn.
type Conn struct {
	Conn *websocket.Conn

	AfterReadFunc   func(messageType int, r io.Reader)
	BeforeCloseFunc func()

	// not useful
	once   sync.Once
	id     string
	stopCh chan struct{}

	// if a socket is bound, then the string userId must not be empty
	userId *string

	// if the socket registered or not
	// not used any more
	registered bool

	// the websocket handler
	// must not be empty
	wh *websocketHandler

	// command storage
	commMap map[string]*CommObject
	// lock for the command map
	mu sync.RWMutex
}

// Write write p to the websocket connection. The error returned will always
// be nil if success.
func (c *Conn) Write(p []byte) (n int, err error) {
	select {
	case <-c.stopCh:
		return 0, errors.New("Conn is closed, can't be written")
	default:
		err = c.Conn.WriteMessage(websocket.TextMessage, p)
		if err != nil {
			return 0, err
		}
		return len(p), nil
	}
}

// after a successful receive
// TODO: add error handling. if any error happen, close the connection, so the client can retry.
func (c *Conn) OnMessage(messageType int, r io.Reader) {

	// messageType: either TextMessage or BinaryMessage, other message like PingMessage, PongMessage will be processed
	// by websocket it self.
	// actually, only TextMessage is used.
	var wm WSMessage
	decoder := json.NewDecoder(r)
	if err := decoder.Decode(&wm); err != nil {
		return
	}

	if wm.Kind == RegisterMessageType {
		// TODO close socket in this case
		// should exit goroutine
		c.HandleRegister(wm.Body)
		return
	} else if wm.Kind == NormalMessageType {
		c.HandleCommand(wm.Body)
		return
	}

}

// do not need any more. HTTP header can be used for registration.
func (c *Conn) HandleRegister(body string) error {

	rm := RegisterMessage{}
	err := json.Unmarshal([]byte(body), &rm)

	if err != nil {
		return err
	}

	wh := c.wh

	userID := rm.Token
	if wh.calcUserIDFunc != nil {
		uID, ok := wh.calcUserIDFunc(rm.Token)
		if !ok {
			return errors.New("calcUserIDFunc failed")
		}
		userID = uID
	}

	// bind
	return wh.cm.Bind(userID, c)

}

// Got a Response back from the client.
func (c *Conn) HandleCommand(body string) error {

	cr := CommResponse{}

	err := json.Unmarshal([]byte(body), &cr)

	if err != nil {
		return err
	}

	commandID := cr.Id

	userID := c.userId
	if userID == nil {
		return errors.New("in HandleCommand: this connection is not registered yet")
	}

	c.mu.RLock()
	obj, ok := c.commMap[commandID]
	c.mu.RUnlock()

	if !ok {
		return errors.New("in HandleCommand: cannot find this command, maybe command timeout")
	}

	// notify the pusher to get the response
	obj.response = &cr
	close(obj.waitCH)

	return nil
}

// Listen listens for receive data from websocket connection. It blocks
// until websocket connection is closed.
func (c *Conn) Listen() {

ReadLoop:
	for {
		select {
		case <-c.stopCh:
			break ReadLoop
		default:
			messageType, r, err := c.Conn.NextReader()
			if err != nil {
				// TODO: handle read error maybe
				log.Println(err.Error())
				break ReadLoop
			}
			// TODO handle error
			c.OnMessage(messageType, r)

		}
	}
}

// Close close the connection.
func (c *Conn) Close() error {
	select {
	case <-c.stopCh:
		return errors.New("Conn already been closed")
	default:
		c.Conn.Close()
		close(c.stopCh)
		return nil
	}
}

// NewConn wraps conn.
func NewConn(conn *websocket.Conn, wh *websocketHandler) *Conn {
	return &Conn{
		wh:      wh,
		Conn:    conn,
		stopCh:  make(chan struct{}),
		commMap: make(map[string]*CommObject),
	}
}
