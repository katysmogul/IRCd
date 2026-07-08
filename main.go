package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"git.delaustral.com/Keiko/IRCd/api"
	"git.delaustral.com/Keiko/IRCd/config"
	"git.delaustral.com/Keiko/IRCd/db"
	"git.delaustral.com/Keiko/IRCd/internal/server"
)

func main() {
	cfgPath := flag.String("config", "ircd.json", "Ruta al archivo de configuración")
	flag.Parse()

	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatalf("[config] No se pudo cargar %s: %v", *cfgPath, err)
	}

	if cfg.LogFile != "" {
		dir := filepath.Dir(cfg.LogFile)
		if err := os.MkdirAll(dir, 0750); err == nil {
			if f, err := os.OpenFile(cfg.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0640); err == nil {
				log.SetOutput(f)
			}
		}
	}

	database, err := db.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("[db] No se pudo abrir %s: %v", cfg.DBPath, err)
	}
	defer database.Close()

	log.Printf("[main] Iniciando delaustral-ircd v1.5 — red: %s", cfg.NetworkName)

	// IRCd
	srv := server.New(cfg, database)

	// API REST + Webhook
	apiSrv := api.New(cfg, srv.GetState())
	srv.SetWebhook(apiSrv)
	srv.GetHandler().SetWebhook(apiSrv)
	apiSrv.Start()

	// Señales del sistema
	go handleSignals(cfg, srv)

	if err := srv.Start(); err != nil {
		log.Fatalf("[main] Error: %v", err)
	}
}

func handleSignals(cfg *config.Config, srv *server.Server) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)

	for sig := range ch {
		switch sig {
		case syscall.SIGHUP:
			// REHASH automático
			log.Printf("[signal] SIGHUP recibido — recargando config")
			newCfg, err := cfg.Reload()
			if err != nil {
				log.Printf("[signal] Error recargando config: %v", err)
				continue
			}
			cfg.Apply(newCfg)
			log.Printf("[signal] Config recargada")

		case syscall.SIGTERM, syscall.SIGINT:
			log.Printf("[signal] %s recibido — shutdown limpio", sig)
			srv.Shutdown("Server shutting down")
		}
	}
}