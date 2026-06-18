package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"

	"bitExchange/internal/coreapi"
)

func main() {
	root := flag.String("root", "", "save root directory (required)")
	listen := flag.String("listen", "127.0.0.1:0", "listen address")
	flag.Parse()

	if *root == "" {
		fmt.Fprintln(os.Stderr, "--root is required")
		os.Exit(2)
	}

	if err := os.MkdirAll(*root, 0o755); err != nil {
		log.Fatalf("mkdir root: %v", err)
	}

	ln, err := net.Listen("tcp", *listen)
	if err != nil {
		log.Fatalf("listen: %v", err)
	}

	fmt.Printf("LISTENING %s\n", ln.Addr().String())
	os.Stdout.Sync()

	srv := coreapi.NewServer(coreapi.ServerConfig{RootDir: *root})
	httpSrv := &http.Server{Handler: srv.Handler()}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	go func() {
		<-sigCh
		httpSrv.Close()
		ln.Close()
	}()

	if err := httpSrv.Serve(ln); err != nil && err != http.ErrServerClosed {
		log.Fatalf("serve: %v", err)
	}
}
