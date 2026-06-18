package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"bitExchange/internal/server"
)

func main() {
	listen := flag.String("listen", ":8080", "server listen address")
	relayEnabled := flag.Bool("relay-enabled", true, "enable relay forwarding")
	relayMaxBytes := flag.Int64("relay-max-bytes", 64<<20, "max relay payload bytes (default 64MB)")
	relayMaxSessions := flag.Int("relay-max-sessions", 100, "max concurrent relay sessions")
	relaySessionTimeout := flag.Duration("relay-session-timeout", 5*time.Minute, "relay session timeout")
	flag.Parse()

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt)

	store := server.NewMemoryStore()
	onlineTable := server.NewOnlineTable()
	relayMgr := server.NewRelayManager(server.RelayConfig{
		Enabled:        *relayEnabled,
		MaxBytes:       *relayMaxBytes,
		MaxSessions:    *relayMaxSessions,
		SessionTimeout: *relaySessionTimeout,
	})

	handler := server.NewSignalingServer(store, onlineTable, relayMgr)

	log.Printf("bitexchange-server listening on %s", *listen)
	log.Printf("  relay: enabled=%v max-bytes=%d max-sessions=%d timeout=%v",
		*relayEnabled, *relayMaxBytes, *relayMaxSessions, *relaySessionTimeout)

	go func() {
		<-signalCh
		log.Println("shutting down...")
		os.Exit(0)
	}()

	log.Fatal(http.ListenAndServe(*listen, handler))
}
