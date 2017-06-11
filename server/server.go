package server

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/hashicorp/errwrap"
	"github.com/kisielk/sqlstruct"
)

const (
	// UserSessionName is used by the cookie
	UserSessionName = "userSession"
)

// SessionStore for the cookie store
var SessionStore = sessions.NewCookieStore(
	// "sessions/",
	// securecookie.GenerateRandomKey(64),
	[]byte("super-secret-key"),
)

var muxRouter = mux.NewRouter()

// Instance of the server
var Instance = &Server{
	httpServer: &http.Server{
		Handler: muxRouter,
	},
	mux: muxRouter,
}

func init() {
	sqlstruct.NameMapper = sqlstruct.ToSnakeCase
}

// Start server for embroidery
func Start(ctx context.Context, params Params) error {
	return Instance.Serve(context.Background(), params)
}

// Params for configuring the server
type Params struct {
	Address string
	DSN     string
}

// Server for handling requests
type Server struct {
	db             *sql.DB
	mux            *mux.Router
	params         Params
	ctx            context.Context
	ctxCancel      context.CancelFunc
	netListener    net.Listener
	httpServer     *http.Server
	serverAddress  string
	temporaryToken string
	email          string
	err            error
}

// DB is a handle for MySQL
func (s *Server) DB() *sql.DB {
	return s.db
}

// Mux of the server
func (s *Server) Mux() *mux.Router {
	return s.mux
}

func (s *Server) createDBHandle() error {
	db, err := sql.Open("mysql", s.params.DSN)

	if err != nil {
		return errwrap.Wrapf("Database error: {{err}}", err)
	}

	if err := db.PingContext(s.ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Can not ping database: %v\n", err)
	}

	s.db = db
	return nil
}

// Serve handlers
func (s *Server) Serve(ctx context.Context, params Params) error {
	s.ctx = ctx
	s.params = params

	if err := s.createDBHandle(); err != nil {
		return err
	}

	address, err := s.listenHTTP(ctx)

	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stdout, "Starting server on %v\n", address)
	return s.serve()
}

func (s *Server) listenHTTP(ctx context.Context) (address string, err error) {
	s.ctx, s.ctxCancel = context.WithCancel(ctx)
	s.netListener, err = net.Listen("tcp", s.params.Address)

	if err != nil {
		return "", errwrap.Wrapf("Can not start server: {{err}}", err)
	}

	s.serverAddress = fmt.Sprintf("http://localhost:%v",
		strings.TrimPrefix(
			s.netListener.Addr().String(),
			"127.0.0.1:"))

	return s.serverAddress, nil
}

func (s *Server) waitServer(w *sync.WaitGroup) {
	<-s.ctx.Done()
	var err = s.httpServer.Shutdown(s.ctx)
	if err != nil && err != context.Canceled {
		s.err = errwrap.Wrapf("Can not shutdown server properly: {{err}}", err)
	}
	w.Done()
}

// Serve HTTP requests
func (s *Server) serve() error {
	var w sync.WaitGroup
	w.Add(1)
	go s.waitServer(&w)

	var serverErr = s.httpServer.Serve(s.netListener)

	if serverErr != http.ErrServerClosed {
		fmt.Fprintf(os.Stderr, fmt.Sprintf("Error closing authentication server: %v", serverErr))
	}

	w.Wait()
	return s.err
}
