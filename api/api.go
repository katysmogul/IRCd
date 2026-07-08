package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"git.delaustral.com/Keiko/IRCd/config"
	"git.delaustral.com/Keiko/IRCd/internal/server"
)

// Server expone una API REST y dispara webhooks.
type Server struct {
	cfg   *config.Config
	state *server.State
}

func New(cfg *config.Config, state *server.State) *Server {
	return &Server{cfg: cfg, state: state}
}

// Start arranca el HTTP listener. No bloquea (corre en goroutine).
func (s *Server) Start() {
	if s.cfg.APIPort == 0 {
		return
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/status",    s.auth(s.handleStatus))
	mux.HandleFunc("/api/clients",   s.auth(s.handleClients))
	mux.HandleFunc("/api/channels",  s.auth(s.handleChannels))
	mux.HandleFunc("/api/bans",      s.auth(s.handleBans))
	mux.HandleFunc("/api/action",    s.auth(s.handleAction))
	mux.HandleFunc("/api/identify",  s.auth(s.handleIdentify))

	addr := fmt.Sprintf("127.0.0.1:%d", s.cfg.APIPort)
	log.Printf("[api] Escuchando en %s", addr)
	go func() {
		if err := http.ListenAndServe(addr, mux); err != nil {
			log.Printf("[api] Error: %v", err)
		}
	}()
}

// --- Middleware de autenticación por token ---

func (s *Server) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("X-IRC-Token")
		if token == "" {
			token = r.URL.Query().Get("token")
		}
		if s.cfg.APIToken != "" && token != s.cfg.APIToken {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		next(w, r)
	}
}

// --- Endpoints ---

// GET /api/status
func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]interface{}{
		"server":   s.cfg.ServerName,
		"network":  s.cfg.NetworkName,
		"clients":  s.state.ClientCount(),
		"channels": s.state.ChannelCount(),
		"uptime":   int64(time.Since(s.state.StartTime()).Seconds()),
	})
}

// GET /api/clients[?nick=X]
func (s *Server) handleClients(w http.ResponseWriter, r *http.Request) {
	filterNick := r.URL.Query().Get("nick")
	type clientInfo struct {
		Nick      string   `json:"nick"`
		User      string   `json:"user"`
		Host      string   `json:"host"`
		RealName  string   `json:"realname"`
		IsOper    bool     `json:"oper"`
		Away      string   `json:"away,omitempty"`
		Channels  []string `json:"channels"`
		Connected int64    `json:"connected_at"`
		Idle      int64    `json:"idle_seconds"`
	}
	var list []clientInfo
	for _, c := range s.state.AllClients() {
		if filterNick != "" && !strings.EqualFold(c.Nick, filterNick) {
			continue
		}
		var chans []string
		for _, ch := range c.Channels() {
			chans = append(chans, ch.Name)
		}
		c.mu.Lock()
		info := clientInfo{
			Nick:      c.Nick,
			User:      c.User,
			Host:      c.CloakHost,
			RealName:  c.RealName,
			IsOper:    c.IsOper,
			Away:      c.Away,
			Channels:  chans,
			Connected: c.ConnectedAt.Unix(),
			Idle:      int64(time.Since(c.LastActive).Seconds()),
		}
		c.mu.Unlock()
		list = append(list, info)
	}
	if list == nil {
		list = []clientInfo{}
	}
	json.NewEncoder(w).Encode(list)
}

// GET /api/channels[?name=#X]
func (s *Server) handleChannels(w http.ResponseWriter, r *http.Request) {
	filterName := r.URL.Query().Get("name")
	type chanInfo struct {
		Name    string   `json:"name"`
		Topic   string   `json:"topic"`
		Members int      `json:"members"`
		Modes   string   `json:"modes"`
		Nicks   []string `json:"nicks"`
	}
	var list []chanInfo
	for _, ch := range s.state.AllChannels() {
		if filterName != "" && !strings.EqualFold(ch.Name, filterName) {
			continue
		}
		ch.mu.RLock()
		secret := ch.Secret
		ch.mu.RUnlock()
		if secret {
			continue // no exponer canales secretos por API
		}
		var nicks []string
		for _, m := range ch.MemberList() {
			nicks = append(nicks, ch.NickPrefix(m)+m.Nick)
		}
		ch.mu.RLock()
		info := chanInfo{
			Name:    ch.Name,
			Topic:   ch.Topic,
			Members: ch.MemberCount(),
			Modes:   ch.ModeString(),
			Nicks:   nicks,
		}
		ch.mu.RUnlock()
		list = append(list, info)
	}
	if list == nil {
		list = []chanInfo{}
	}
	json.NewEncoder(w).Encode(list)
}

// GET /api/bans
func (s *Server) handleBans(w http.ResponseWriter, r *http.Request) {
	type banInfo struct {
		Type    string `json:"type"`
		Mask    string `json:"mask"`
		Reason  string `json:"reason"`
		SetBy   string `json:"set_by"`
		SetAt   int64  `json:"set_at"`
		Expires int64  `json:"expires"`
	}
	var list []banInfo
	for _, b := range s.state.GetGLines() {
		list = append(list, banInfo{"gline", b.Mask, b.Reason, b.SetBy, b.SetAt.Unix(), b.Expires.Unix()})
	}
	for _, b := range s.state.GetKLines() {
		list = append(list, banInfo{"kline", b.Mask, b.Reason, b.SetBy, b.SetAt.Unix(), b.Expires.Unix()})
	}
	if list == nil {
		list = []banInfo{}
	}
	json.NewEncoder(w).Encode(list)
}

// POST /api/action
// Body JSON: {"action": "privmsg", "target": "#canal", "text": "Hola"}
//            {"action": "notice",  "target": "nick",   "text": "Aviso"}
//            {"action": "kill",    "target": "nick",   "reason": "spam"}
//            {"action": "gline",   "mask": "*@1.2.3.4","reason": "bot"}
func (s *Server) handleAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"POST only"}`, http.StatusMethodNotAllowed)
		return
	}
	var req map[string]string
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
		return
	}

	action := strings.ToLower(req["action"])
	switch action {
	case "privmsg", "notice":
		target := req["target"]
		text := req["text"]
		if target == "" || text == "" {
			http.Error(w, `{"error":"target y text requeridos"}`, http.StatusBadRequest)
			return
		}
		cmd := strings.ToUpper(action)
		line := fmt.Sprintf(":%s %s %s :%s", s.cfg.ServerName, cmd, target, text)
		if strings.HasPrefix(target, "#") || strings.HasPrefix(target, "&") {
			if ch, ok := s.state.GetChannel(target); ok {
				ch.BroadcastAll(line)
			}
		} else {
			if c, ok := s.state.GetClient(target); ok {
				c.Send(line)
			}
		}
		json.NewEncoder(w).Encode(map[string]string{"ok": "sent"})

	case "kill":
		target := req["target"]
		reason := req["reason"]
		if target == "" {
			http.Error(w, `{"error":"target requerido"}`, http.StatusBadRequest)
			return
		}
		if c, ok := s.state.GetClient(target); ok {
			c.Send(fmt.Sprintf("ERROR :Closing Link: %s (Killed by services: %s)", c.Host, reason))
			c.mu.Lock()
			c.State = server.StateQuit
			c.mu.Unlock()
			json.NewEncoder(w).Encode(map[string]string{"ok": "killed"})
		} else {
			http.Error(w, `{"error":"nick no encontrado"}`, http.StatusNotFound)
		}

	case "gline":
		mask := req["mask"]
		reason := req["reason"]
		if mask == "" {
			http.Error(w, `{"error":"mask requerido"}`, http.StatusBadRequest)
			return
		}
		ban := server.Ban{Mask: mask, Reason: reason, SetBy: "API", SetAt: time.Now()}
		s.state.AddGLine(ban)
		json.NewEncoder(w).Encode(map[string]string{"ok": "gline added"})

	default:
		http.Error(w, `{"error":"acción desconocida"}`, http.StatusBadRequest)
	}
}

// --- Webhook ---

// WebhookEvent es el payload que se envía al endpoint configurado.
type WebhookEvent struct {
	Event   string                 `json:"event"`
	Time    int64                  `json:"time"`
	Payload map[string]interface{} `json:"payload"`
}

// POST /api/identify
// Body: {"nick": "Keiko", "identified": true}
// Llamado por NiCK services después de que el usuario se identifica con /IDENTIFY
func (s *Server) handleIdentify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"POST only"}`, http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Nick       string `json:"nick"`
		Identified bool   `json:"identified"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Nick == "" {
		http.Error(w, `{"error":"nick requerido"}`, http.StatusBadRequest)
		return
	}
	c, ok := s.state.GetClient(req.Nick)
	if !ok {
		http.Error(w, `{"error":"nick no conectado"}`, http.StatusNotFound)
		return
	}
	c.SetIdentified(req.Identified)

	// Notificar al cliente con un mode +r en su nick (convención de services)
	if req.Identified {
		c.Send(fmt.Sprintf(":%s MODE %s :+r", c.Nick, c.Nick))
	} else {
		c.Send(fmt.Sprintf(":%s MODE %s :-r", c.Nick, c.Nick))
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"ok": true, "identified": req.Identified})
}

// FireWebhook envía un evento al webhook configurado (no bloquea).
func (s *Server) FireWebhook(event string, payload map[string]interface{}) {
	if s.cfg.WebhookURL == "" {
		return
	}
	go func() {
		ev := WebhookEvent{
			Event:   event,
			Time:    time.Now().Unix(),
			Payload: payload,
		}
		data, err := json.Marshal(ev)
		if err != nil {
			return
		}
		resp, err := http.Post(s.cfg.WebhookURL, "application/json", bytes.NewReader(data))
		if err != nil {
			log.Printf("[webhook] Error enviando %s: %v", event, err)
			return
		}
		resp.Body.Close()
	}()
}
