# Changelog

## v1.5 — Julio 2026

### Agregado
- CAP IRCv3: negociación `multi-prefix`, `away-notify`, `server-time`
- Registro pausado hasta `CAP END` para clientes modernos
- `USERHOST` / `ISON` — compatibilidad con scripts y bots legacy
- `ADMIN` / `INFO` — información del servidor
- Modo de canal `+r` — solo nicks identificados con services
- `RESTART` — shutdown limpio vía IRC (solo opers)
- `WALLOPS` — mensaje a todos los opers conectados
- `SIGTERM` → shutdown limpio, avisa a todos los clientes antes de salir
- `SIGHUP` → REHASH automático sin bajar el servidor
- API REST: `/api/identify` para integración con NiCK services
- Endpoint `/api/identify` para marcar nicks como identificados desde services externos
- Webhooks: eventos `client.connect` y `client.quit`

### Cambiado
- `REHASH` ahora recarga la config en caliente (MOTD, opers, limits, flood, webhook)
- `ISUPPORT` actualizado con `CHANMODES=b,k,l,imnrst`

## v1.1 — Julio 2026

### Agregado
- `UNGLINE` / `UNKLINE` — remoción de bans sin reiniciar, persiste en SQLite
- `WATCH` / `MONITOR` (IRCv3) — notificaciones de conexión/desconexión
- API REST en `127.0.0.1`: `/api/status`, `/api/clients`, `/api/channels`, `/api/bans`, `/api/action`
- Webhooks configurables para eventos de red
- `RemoveBan` en capa SQLite

### Cambiado
- `NewState` acepta `serverName` para notificaciones de WATCH
- `Config` soporta `Reload()` y `Apply()` para REHASH diferenciado

## v1.0 — Julio 2026

### Inicial
- Registro: `NICK` `USER` `QUIT` `PASS`
- Canales: `JOIN` `PART` `KICK` `INVITE` `TOPIC` `MODE`
- Modos: `+o +v +b +m +n +s +i +t +k +l`
- Mensajería: `PRIVMSG` `NOTICE` `AWAY`
- Consultas: `NAMES` `LIST` `WHO` `WHOIS`
- Operadores: `OPER` `KILL` `GLINE` `KLINE` `REHASH`
- TLS nativo (puerto 6697)
- IP cloaking HMAC-SHA256
- G-Lines y K-Lines persistentes en SQLite
- Flood protection por cliente
- `PING`/`PONG` keepalive con timeout configurable
- `LUSERS` `MOTD` `VERSION` `TIME`