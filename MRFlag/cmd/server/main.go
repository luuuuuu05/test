package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"mrflag/internal/api"
	"mrflag/internal/config"
	"mrflag/internal/room"
	"mrflag/internal/udp"
	"mrflag/internal/ws"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to config.yaml")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	mgr := room.NewManager(room.ManagerConfig{
		DefaultDuration:  cfg.Game.DefaultDuration,
		DefaultMaxFlags:  cfg.Game.DefaultMaxFlags,
		DefaultMinFlags:  cfg.Game.DefaultMinFlags,
		RespawnDelay:     cfg.Game.RespawnDelay,
		GrabDistance:     cfg.Game.GrabDistance,
		DoubleDuration:   cfg.Game.DoubleDuration,
		MaxDoubleItems:   cfg.Game.MaxDoubleItems,
		SceneMapMaxBytes: cfg.Game.SceneMapMaxBytes,
		FlagWeights:      cfg.Game.FlagWeights,
	})
	hub := ws.NewHub(mgr)
	mgr.SetEventSink(hub)

	apiMux := http.NewServeMux()
	api.NewHandler(mgr).Register(apiMux)
	httpServer := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Server.HTTPPort),
		Handler:           withCORS(apiMux),
		ReadHeaderTimeout: 5 * time.Second,
	}

	wsMux := http.NewServeMux()
	wsMux.HandleFunc("/ws", hub.HandleWS)
	wsServer := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Server.WSPort),
		Handler:           wsMux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("http listening on :%d", cfg.Server.HTTPPort)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("http server: %v", err)
		}
	}()

	go func() {
		log.Printf("ws listening on :%d", cfg.Server.WSPort)
		if err := wsServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("ws server: %v", err)
		}
	}()

	go func() {
		addr := fmt.Sprintf(":%d", cfg.Server.UDPPort)
		if err := udp.NewServer(addr, mgr).ListenAndServe(); err != nil {
			log.Fatalf("udp server: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	log.Println("shutting down")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = httpServer.Shutdown(ctx)
	_ = wsServer.Shutdown(ctx)
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Client-Type, X-Room-ID")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
