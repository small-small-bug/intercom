// Package wserver provides building simple websocket server with message push.
package intercom

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
)

const (
	serverDefaultWSPath     = "/ws"
	serverDefaultPushPath   = "/push"
	serverDefaultHealthPath = "/health"
	serverDefaultLookupPath = "/lookup"
)

var defaultUpgrader = &websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(*http.Request) bool {
		return true
	},
}

// Server defines parameters for running websocket server.
type Server struct {
	// Address for server to listen on
	Addr string

	// Path for websocket request, default "/ws".
	WSPath string

	// Path for push message, default "/push".
	PushPath string

	HealthPath string

	LookupPath string

	// Upgrader is for upgrade connection to websocket connection using
	// "github.com/gorilla/websocket".
	//
	// If Upgrader is nil, default upgrader will be used. Default upgrader is
	// set ReadBufferSize and WriteBufferSize to 1024, and CheckOrigin always
	// returns true.
	Upgrader *websocket.Upgrader

	// Check token if it's valid and return userID. If token is valid, userID
	// must be returned and ok should be true. Otherwise ok should be false.
	AuthToken func(token string) (userID string, ok bool)

	// Authorize push request. Message will be sent if it returns true,
	// otherwise the request will be discarded. Default nil and push request
	// will always be accepted.
	PushAuth func(r *http.Request) bool

	wh *websocketHandler
	ph *pushHandler
	hh *healthHandler
	lh *lookupHandler
}

// ListenAndServe listens on the TCP network address and handle websocket
// request.
func (s *Server) ListenAndServe() error {
	cm := &ConnManager{
		userConnMap: make(map[string]*Conn),
	}

	// websocket request handler
	wh := websocketHandler{
		upgrader: defaultUpgrader,
		cm:       cm,
	}
	if s.Upgrader != nil {
		wh.upgrader = s.Upgrader
	}
	if s.AuthToken != nil {
		wh.calcUserIDFunc = s.AuthToken
	}
	s.wh = &wh
	http.Handle(s.WSPath, s.wh)

	lh := lookupHandler{
		cm: cm,
	}
	s.lh = &lh
	http.Handle(s.LookupPath, s.lh)

	// push request handler
	ph := pushHandler{
		cm: cm,
	}
	if s.PushAuth != nil {
		ph.authFunc = s.PushAuth
	}
	s.ph = &ph
	http.Handle(s.PushPath, s.ph)

	hh := healthHandler{}
	s.hh = &hh
	http.Handle(s.HealthPath, s.hh)

	return http.ListenAndServe(s.Addr, nil)
}

// Push filters connections by userID and event, then write message
func (s *Server) Push(userID, event, message string) (*CommObject, error) {
	return s.ph.push(userID, event, message)
}

// Drop find connections by userID and event, then close them. The userID can't
// be empty. The event is ignored if it's empty.
func (s *Server) Drop(userID, event string) (int, error) {
	return s.wh.closeConns(userID, event)
}

// Check parameters of Server, returns error if fail.
func (s Server) check() error {
	if !checkPath(s.WSPath) {
		return fmt.Errorf("WSPath: %s not illegal", s.WSPath)
	}
	if !checkPath(s.PushPath) {
		return fmt.Errorf("PushPath: %s not illegal", s.PushPath)
	}
	if s.WSPath == s.PushPath {
		return errors.New("WSPath is equal to PushPath")
	}

	return nil
}

// NewServer creates a new Server.
func NewServer(addr string) *Server {
	return &Server{
		Addr:       addr,
		WSPath:     serverDefaultWSPath,
		PushPath:   serverDefaultPushPath,
		HealthPath: serverDefaultHealthPath,
		LookupPath: serverDefaultLookupPath,
	}
}

func checkPath(path string) bool {
	if path != "" && !strings.HasPrefix(path, "/") {
		return false
	}
	return true
}
