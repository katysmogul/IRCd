package config

import (
	"encoding/json"
	"os"
	"sync"
)

type Oper struct {
	Name     string `json:"name"`
	Password string `json:"password"`
	HostMask string `json:"host_mask"`
	Flags    string `json:"flags"`
}

type Config struct {
	mu sync.RWMutex `json:"-"`
	cfgPath string  `json:"-"`

	ServerName   string   `json:"server_name"`
	NetworkName  string   `json:"network_name"`
	Description  string   `json:"description"`
	Bind         string   `json:"bind"`
	Port         int      `json:"port"`
	TLSPort      int      `json:"tls_port"`
	TLSCert      string   `json:"tls_cert"`
	TLSKey       string   `json:"tls_key"`
	MOTD         []string `json:"motd"`
	MaxClients   int      `json:"max_clients"`
	MaxChannels  int      `json:"max_channels"`
	PingInterval int      `json:"ping_interval"`
	PingTimeout  int      `json:"ping_timeout"`
	FloodLines   int      `json:"flood_lines"`
	FloodBurst   int      `json:"flood_burst"`
	DBPath       string   `json:"db_path"`
	Opers        []Oper   `json:"opers"`
	CloakKey     string   `json:"cloak_key"`
	LogFile      string   `json:"log_file"`
	APIPort      int      `json:"api_port"`
	APIToken     string   `json:"api_token"`
	WebhookURL   string   `json:"webhook_url"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	cfg.cfgPath = path
	cfg.setDefaults()
	return &cfg, nil
}

// Reload lee el archivo de config de nuevo y devuelve una copia nueva.
func (c *Config) Reload() (*Config, error) {
	c.mu.RLock()
	path := c.cfgPath
	c.mu.RUnlock()
	return Load(path)
}

// Apply copia los campos recargables de newCfg sobre c (en caliente).
// No toca ServerName, Bind, Port, DBPath ni CloakKey (requieren reinicio).
func (c *Config) Apply(newCfg *Config) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.MOTD = newCfg.MOTD
	c.Opers = newCfg.Opers
	c.MaxClients = newCfg.MaxClients
	c.MaxChannels = newCfg.MaxChannels
	c.PingInterval = newCfg.PingInterval
	c.PingTimeout = newCfg.PingTimeout
	c.FloodLines = newCfg.FloodLines
	c.FloodBurst = newCfg.FloodBurst
	c.Description = newCfg.Description
	c.TLSCert = newCfg.TLSCert
	c.TLSKey = newCfg.TLSKey
	c.WebhookURL = newCfg.WebhookURL
	c.APIToken = newCfg.APIToken
}

func (c *Config) setDefaults() {
	if c.Port == 0 {
		c.Port = 6667
	}
	if c.TLSPort == 0 {
		c.TLSPort = 6697
	}
	if c.MaxClients == 0 {
		c.MaxClients = 1000
	}
	if c.MaxChannels == 0 {
		c.MaxChannels = 500
	}
	if c.PingInterval == 0 {
		c.PingInterval = 90
	}
	if c.PingTimeout == 0 {
		c.PingTimeout = 120
	}
	if c.FloodLines == 0 {
		c.FloodLines = 5
	}
	if c.FloodBurst == 0 {
		c.FloodBurst = 10
	}
	if c.Bind == "" {
		c.Bind = "0.0.0.0"
	}
}