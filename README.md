# MoonEngineAPI

Gateway/orquestador central en Go para el sistema de traducción de voz en tiempo real de LusoTranslate (NeoForge 1.21.1).

## Arquitectura

```
NeoForge Mod (Java/WebSocket)
  ↓ binary frame [uuid(16b) + srcLang(2b) + dstLang(2b) + PCM chunks]
MoonEngineAPI (Go)
  ├── SessionManager  → una goroutine worker por jugador
  ├── Silence-timeout VAD → acumula chunks, dispara tras 600ms de silencio
  ├── PCM → WAV in-memory
  ├── Flask /transcribe → STT (faster-whisper)
  ├── Filtro de confianza → avg_logprob / no_speech_prob
  ├── Flask /translate  → NLLB-200
  └── WebSocket push → HUD del jugador
```

## Variables de entorno

| Variable | Default | Descripción |
|---|---|---|
| `LISTEN_ADDR` | `:8080` | Dirección de escucha |
| `FLASK_BASE_URL` | `http://localhost:5001` | URL base de Flask backend |
| `SILENCE_TIMEOUT_MS` | `600` | Timeout de silencio en ms |
| `RATE_BURST` | `3` | Burst del rate limiter por jugador |
| `RATE_EVERY_MS` | `2000` | Ventana del rate limiter en ms |
| `AUDIO_CHAN_CAP` | `50` | Capacidad del canal de audio por sesión |
| `AVG_LOGPROB_MIN` | `-0.8` | Umbral mínimo de confianza STT |
| `NO_SPEECH_PROB_MAX` | `0.4` | Umbral máximo de no-habla STT |
| `MIN_TEXT_LEN` | `3` | Longitud mínima de texto transcrito |

## Protocolo WebSocket (NeoForge → Go)

### Frame binario (audio)
```
[0..15]  playerUUID  (16 bytes, UTF-8)
[16..17] srcLang     (2 bytes, ej: "es", "en")
[18..19] dstLang     (2 bytes)
[20..]   PCM data    (raw 16-bit LE, 48kHz mono)
```

### Frame de texto (actualizar idioma)
```
[0..1]   0x0001      (uint16 BE, tipo = update_lang)
[2..17]  playerUUID  (16 bytes)
[18..19] nuevo srcLang
[20..21] nuevo dstLang
```

### Frame de texto (respuesta del servidor)
```
"<playerUUID>:<texto traducido>"
```

## Endpoints HTTP

- `GET /health` — estado del servidor
- `POST /session/lang` — actualizar idiomas de una sesión activa
  ```json
  { "player_id": "uuid", "src_lang": "es", "dst_lang": "en" }
  ```

## Construcción

```bash
go build ./cmd/server
./server
```

## Modificación requerida en Flask

El endpoint `/transcribe` debe retornar `avg_logprob` y `no_speech_prob`:

```python
segments_list = list(segments)
avg_logprob = sum(s.avg_logprob for s in segments_list) / max(len(segments_list), 1)
no_speech = sum(s.no_speech_prob for s in segments_list) / max(len(segments_list), 1)
return jsonify({
    "text": text,
    "lang": info.language,
    "avg_logprob": avg_logprob,
    "no_speech_prob": no_speech
})
```
