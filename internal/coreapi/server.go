package coreapi

import (
	"net/http"
	"sync"

	"bitExchange/internal/config"
	icrypto "bitExchange/internal/crypto"
	"bitExchange/internal/pairing"
)

type ServerConfig struct {
	RootDir string
}

type Server struct {
	cfg       ServerConfig
	taskMgr   *TaskManager
	inbox     *SSEBroker
	mu        sync.Mutex
	device    icrypto.Identity
	peerStore *pairing.PeerStore
}

func NewServer(cfg ServerConfig) *Server {
	return &Server{
		cfg:     cfg,
		taskMgr: NewTaskManager(),
		inbox:   NewSSEBroker(),
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/config", s.handleConfig)
	mux.HandleFunc("/api/device", s.handleDevice)
	mux.HandleFunc("/api/history", s.handleHistory)
	mux.HandleFunc("/api/peers", s.handlePeers)
	mux.HandleFunc("/api/send/text", s.handleSendText)
	mux.HandleFunc("/api/send/file", s.handleSendFile)
	mux.HandleFunc("/api/tasks", s.handleTasks)
	mux.HandleFunc("/api/tasks/", s.handleTaskEvents)
	mux.HandleFunc("/api/inbox/events", s.handleInboxEvents)
	return mux
}

func (s *Server) configPath() string {
	return s.cfg.RootDir + "/config.json"
}

func (s *Server) loadConfig() config.ClientConfig {
	cfg, _ := config.LoadClientConfig(s.configPath())
	return cfg
}

