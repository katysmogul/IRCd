# delaustral-ircd

IRCd minimalista escrito en Go para la red IRC Del Austral.

Implementación completa del protocolo IRC (RFC 1459) con soporte IRCv3 CAP, TLS nativo, operadores, G-Lines/K-Lines persistentes en SQLite, WATCH/MONITOR, y API REST integrada para integración con servicios externos.

---

## Requisitos

- Go 1.21+
- gcc (para cgo / go-sqlite3)
- libsqlite3-dev

```bash
apt install gcc libsqlite3-dev
```

---

## Instalación

```bash
git clone https://git.delaustral.com/Keiko/IRCd
cd IRCd
go get github.com/mattn/go-sqlite3
go build -o delaustral-ircd .
```

---

## Configuración

Copiar y editar `ircd.json`:

```json
{
  "server_name": "irc.delaustral.com",
  "network_name": "DelAustral",
  "description": "IRC Del Austral",
  "bind": "0.0.0.0",
  "port": 6667,
  "tls_port": 6697,
  "tls_cert": "/etc/ssl/delaustral/cert.pem",
  "tls_key":  "/etc/ssl/delaustral/key.pem",
  "motd": ["Bienvenido a IRC Del Austral.", "Red privada. Respetar las normas."],
  "max_clients": 1000,
  "ping_interval": 90,
  "ping_timeout": 120,
  "flood_burst": 10,
  "db_path": "/var/lib/delaustral-ircd/ircd.db",
  "cloak_key": "CLAVE_ALEATORIA_AQUI",
  "log_file": "/var/log/delaustral-ircd/ircd.log",
  "api_port": 7766,
  "api_token": "TOKEN_SECRETO",
  "webhook_url": "",
  "opers": [
    {
      "name": "Keiko",
      "password": "CONTRASEÑA",
      "host_mask": "*@*",
      "flags": "aAbBcC"
    }
  ]
}
```

### Campos principales

| Campo | Descripción |
|-------|-------------|
| `server_name` | FQDN del servidor |
| `network_name` | Nombre de la red IRC |
| `port` / `tls_port` | Puertos plain (6667) y TLS (6697) |
| `tls_cert` / `tls_key` | Certificados TLS (opcional) |
| `cloak_key` | Clave HMAC para enmascarar IPs |
| `db_path` | Base SQLite para G-Lines y K-Lines (se crea automáticamente) |
| `api_port` | Puerto de la API REST (solo localhost, 0 = desactivada) |
| `api_token` | Token de autenticación para la API |
| `webhook_url` | URL para recibir eventos IRC (vacío = desactivado) |

---

## Ejecución

```bash
# Manual
./delaustral-ircd -config /etc/delaustral-ircd/ircd.json

# Con systemd (ver abajo)
systemctl start delaustral-ircd
```

### Systemd

Guardar en `/etc/systemd/system/delaustral-ircd.service`:

```ini
[Unit]
Description=Del Austral IRCd
After=network.target

[Service]
ExecStart=/usr/local/bin/delaustral-ircd -config /etc/delaustral-ircd/ircd.json
Restart=on-failure
RestartSec=5
User=ircd
Group=ircd
AmbientCapabilities=CAP_NET_BIND_SERVICE

[Install]
WantedBy=multi-user.target
```

```bash
systemctl daemon-reload
systemctl enable --now delaustral-ircd
```

---

## Señales

| Señal | Efecto |
|-------|--------|
| `SIGHUP` | REHASH — recarga `ircd.json` sin bajar el servidor |
| `SIGTERM` | Shutdown limpio — avisa a todos los clientes antes de salir |

```bash
kill -HUP  $(pidof delaustral-ircd)   # rehash
kill -TERM $(pidof delaustral-ircd)   # shutdown limpio
```

---

## Protocolo soportado

### Comandos de registro
`NICK` `USER` `PASS` `QUIT` `CAP`

### Canales
`JOIN` `PART` `KICK` `INVITE` `TOPIC` `MODE` `NAMES` `LIST`

### Mensajería
`PRIVMSG` `NOTICE` `AWAY`

### Consultas
`WHO` `WHOIS` `USERHOST` `ISON` `WATCH` `MONITOR`

### Operadores
`OPER` `KILL` `GLINE` `KLINE` `UNGLINE` `UNKLINE` `REHASH` `RESTART` `WALLOPS`

### Info
`LUSERS` `MOTD` `VERSION` `TIME` `ADMIN` `INFO` `PING` `PONG`

### IRCv3 CAP
`multi-prefix` `away-notify` `server-time`

---

## Modos de canal

| Modo | Descripción |
|------|-------------|
| `+o` | Operador de canal |
| `+v` | Voice |
| `+b` | Ban (user@host con wildcards) |
| `+m` | Moderado — solo +o/+v pueden hablar |
| `+n` | Sin mensajes externos |
| `+s` | Secreto — no aparece en LIST |
| `+i` | Solo por invitación |
| `+t` | Solo ops cambian el topic |
| `+k` | Clave de acceso |
| `+l` | Límite de usuarios |
| `+r` | Solo nicks identificados con services |

---

## Comandos de oper

```irc
/OPER nombre contraseña
/KILL nick :razón
/GLINE user@host :razón
/KLINE user@host :razón
/UNGLINE user@host
/UNKLINE user@host
/REHASH
/RESTART
/WALLOPS :mensaje para opers
```

---

## API REST

Escucha solo en `127.0.0.1`. Autenticación con header `X-IRC-Token` o parámetro `?token=`.

### Endpoints

| Método | Ruta | Descripción |
|--------|------|-------------|
| GET | `/api/status` | Estado general del servidor |
| GET | `/api/clients` | Lista de clientes conectados |
| GET | `/api/clients?nick=X` | Info de un nick específico |
| GET | `/api/channels` | Lista de canales públicos |
| GET | `/api/bans` | G-Lines y K-Lines activos |
| POST | `/api/action` | Enviar acción al servidor |
| POST | `/api/identify` | Marcar un nick como identificado |

### Ejemplos

```bash
# Estado
curl -H "X-IRC-Token: TOKEN" http://127.0.0.1:7766/api/status

# Clientes online
curl -H "X-IRC-Token: TOKEN" http://127.0.0.1:7766/api/clients

# Enviar mensaje a un canal
curl -X POST -H "X-IRC-Token: TOKEN" -H "Content-Type: application/json" \
  -d '{"action":"privmsg","target":"#general","text":"Mantenimiento en 5 minutos"}' \
  http://127.0.0.1:7766/api/action

# Marcar nick como identificado (llamado por NiCK services)
curl -X POST -H "X-IRC-Token: TOKEN" -H "Content-Type: application/json" \
  -d '{"nick":"Keiko","identified":true}' \
  http://127.0.0.1:7766/api/identify

# Aplicar G-Line
curl -X POST -H "X-IRC-Token: TOKEN" -H "Content-Type: application/json" \
  -d '{"action":"gline","mask":"*@1.2.3.4","reason":"spam"}' \
  http://127.0.0.1:7766/api/action
```

### Webhooks

Cuando `webhook_url` está configurado, el IRCd hace `POST` a esa URL en JSON para los siguientes eventos:

| Evento | Descripción |
|--------|-------------|
| `client.connect` | Un usuario completó el registro |
| `client.quit` | Un usuario se desconectó |

Payload de ejemplo:
```json
{
  "event": "client.connect",
  "time": 1720000000,
  "payload": {
    "nick": "Keiko",
    "user": "keiko",
    "host": "1.2.xxx.IP",
    "realname": "Del Austral"
  }
}
```

---

## Integración con NiCK services

El modo `+r` de canal bloquea el acceso a usuarios no identificados. El flujo es:

1. Usuario se conecta y escribe `/msg NiCK IDENTIFY contraseña`
2. NiCK services verifica la contraseña
3. NiCK services llama `POST /api/identify` con `{"nick":"X","identified":true}`
4. El IRCd actualiza el flag en memoria y envía `MODE nick +r` al cliente
5. El usuario ahora puede entrar a canales `+r`

---

## Estructura del proyecto

```
delaustral-ircd/
├── main.go                    # Punto de entrada, señales
├── ircd.json                  # Configuración
├── go.mod
├── config/
│   └── config.go              # Carga y REHASH de config
├── db/
│   └── db.go                  # SQLite: bans, oper log
├── api/
│   └── api.go                 # API REST + webhooks
└── internal/
    ├── irc/
    │   ├── parser.go          # Parser RFC 1459
    │   └── numerics.go        # Constantes RPL_* / ERR_*
    └── server/
        ├── server.go          # Listener TCP/TLS, loop de conexiones
        ├── client.go          # Struct Client, goroutines I/O
        ├── channel.go         # Struct Channel, modos, bans
        ├── state.go           # Estado global: clients, channels, bans, watches
        └── handlers.go        # Todos los comandos IRC
```

---

## Versión

**v1.5** — Julio 2026

Cambios desde v1.0: CAP IRCv3, USERHOST/ISON, ADMIN/INFO, modo +r, RESTART, WALLOPS, SIGTERM/SIGHUP, API REST, webhooks, WATCH/MONITOR, UNGLINE/UNKLINE, REHASH real.