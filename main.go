package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
	"github.com/henvic/embroidery/server"
)

var params = server.Params{}

func main() {
	flag.Parse()

	s := server.Server{}
	r := mux.NewRouter()

	if err := s.Serve(context.Background(), params, r); err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(1)
	}
}

func init() {
	flag.StringVar(&params.Address, "addr", "127.0.0.1:8080", "Serving address")
	flag.StringVar(&params.DSN, "dsn", "root@/embroidery", "dsn (MySQL)")
}
