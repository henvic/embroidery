package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/henvic/embroidery/modules"
	"github.com/henvic/embroidery/server"
)

var params = server.Params{}

func main() {
	flag.Parse()

	if err := server.Start(context.Background(), params); err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(1)
	}
}

func init() {
	flag.StringVar(&params.Address, "addr", "127.0.0.1:8080", "Serving address")
	flag.StringVar(&params.DSN, "dsn", "root@/embroidery", "dsn (MySQL)")
}
