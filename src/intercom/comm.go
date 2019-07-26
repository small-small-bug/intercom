package intercom

import (
	"errors"
	"sync"
	"time"
)

// one command, has several properties
// 1. async or sync
// 2. expire
// 3. need response or not

type CommObject struct {
	id string

	async bool

	start   time.Time
	timeout time.Duration

	request  *CommRequest
	response *CommResponse
	waitCH   chan struct{}

	// a command is associated with with one connection
	conn *Conn
}

type CommRequest struct {
	Id  string `json:"id"`
	Msg string `json:"msg"`
}

type CommResponse struct {
	Id  string `json:"id"`
	Msg string `json:"msg"`
}

// one connection may have many commands
type CommConn struct {
	conn    *Conn
	commMap map[string]*CommObject
}

// user-connection-command management
type ConnManager struct {
	mu          sync.RWMutex
	userConnMap map[string]*Conn
}

func (m *ConnManager) Bind(userID string, conn *Conn) error {

	if userID == "" {
		return errors.New("in Bind: userID can't be empty")
	}

	if conn == nil {
		return errors.New("in Bind: conn can't be nil")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.userConnMap[userID]; ok {
		return errors.New("in Bind: already registered")
	}

	var user = userID

	// save user id in the connection, in this way to do the cleanup easily.
	conn.userId = &user
	m.userConnMap[userID] = conn

	return nil
}

// remove all the commands
// leave others to close the connection

// need a way to unbind without user ID
// you cannot get userid in close context

func (m *ConnManager) Unbind(conn *Conn) error {

	if conn == nil {
		return errors.New("in Unbind: conn can't be nil")
	}

	var userID *string
	userID = conn.userId

	// the connection is not registered yet.
	if conn.userId == nil {
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if cc, ok := m.userConnMap[*userID]; ok {
		if cc == conn {
			delete(m.userConnMap, *userID)
		} else {
			return errors.New("in Unbind: cannot unbind it. it is not yours")
		}
	} else {
		return errors.New("in Unbind: cannot find registered user")
	}

	return nil
}

func (m *ConnManager) hasUser(userId string) (bool, error) {

	if userId == "" {
		return false, errors.New("userID can't be empty")
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.userConnMap[userId]; ok {
		return true, nil
	}

	return false, nil
}
func (m *ConnManager) getConnByUser(userID string) (*Conn, error) {

	if userID == "" {
		return nil, errors.New("in getConnByUser: userID can't be empty")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	c, ok := m.userConnMap[userID]
	if !ok {
		return nil, errors.New("in getConnByUser: no such user")
	}

	return c, nil
}

// called when pushing a command to client.
func (m *ConnManager) newCommand(userID, commID string) (*CommObject, error) {

	if userID == "" {
		return nil, errors.New("in newCommand: userID can't be empty")
	}

	if commID == "" {
		return nil, errors.New("in newCommand: userID can't be empty")
	}

	// do we really need this lock for conn lookup?
	m.mu.Lock()
	c, ok := m.userConnMap[userID]
	m.mu.Unlock()

	if !ok {
		return nil, errors.New("in newCommand: no such user")
	}

	// use connection lock for command operation
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, ok := c.commMap[commID]; ok {
		return nil, errors.New("in newCommand: command already existed")
	} else {

		// create a new command
		comm := CommObject{
			conn:   c,
			waitCH: make(chan struct{}),
		}

		c.commMap[commID] = &comm
		return &comm, nil
	}
}

func (m *ConnManager) lookupCommand(userID, commID string) (*CommObject, error) {

	if userID == "" {
		return nil, errors.New("userID can't be empty")
	}

	if commID == "" {
		return nil, errors.New("userID can't be empty")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if cc, ok := m.userConnMap[userID]; !ok {
		return nil, errors.New("no such user")
	} else if c, ok := cc.commMap[commID]; ok {
		return c, nil
	} else {
		return nil, nil
	}
}

func (m *ConnManager) removeCommand(userID, commID string) error {

	if userID == "" {
		return errors.New("userID can't be empty")
	}

	if commID == "" {
		return errors.New("userID can't be empty")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// if I cannot find the command, just return OK
	if cc, ok := m.userConnMap[userID]; !ok {
		return errors.New("no such user")
	} else if _, ok := cc.commMap[commID]; ok {
		delete(cc.commMap, commID)
		return nil
	} else {
		return errors.New("no such command")
	}
}
