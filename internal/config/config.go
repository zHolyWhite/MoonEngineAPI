package config

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	ListenAddr      string
	APIKeys         map[string]string
	STTURL          string
	TranslateURL    string
	MaxBodyBytes    int64
	MaxFrameBytes   int
	ShutdownTimeout time.Duration
	LogLevel        string
}

func Load() (*Config, error) {
	if err := loadDotEnv(".env"); err != nil {
		return nil, err
	}

	cfg := &Config{
		ListenAddr:   getString("MOONENGINE_LISTEN", "127.0.0.1:8080"),
		STTURL:       getString("MOONENGINE_STT_URL", ""),
		TranslateURL: getString("MOONENGINE_TRANSLATE_URL", ""),
		LogLevel:     strings.ToLower(getString("MOONENGINE_LOG_LEVEL", "info")),
	}

	var err error
	if cfg.MaxBodyBytes, err = getInt64("MOONENGINE_MAX_BODY_BYTES", 8*1024*1024); err != nil {
		return nil, err
	}
	maxFrame, err := getInt64("MOONENGINE_MAX_FRAME_BYTES", 64*1024)
	if err != nil {
		return nil, err
	}
	cfg.MaxFrameBytes = int(maxFrame)
	shutdownMs, err := getInt64("MOONENGINE_SHUTDOWN_TIMEOUT_MS", 10000)
	if err != nil {
		return nil, err
	}
	cfg.ShutdownTimeout = time.Duration(shutdownMs) * time.Millisecond

	keys, err := parseKeys(os.Getenv("MOONENGINE_API_KEYS"))
	if err != nil {
		return nil, err
	}
	cfg.APIKeys = keys

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func loadDotEnv(path string) error {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		value = strings.Trim(value, `"'`)
		if key == "" {
			continue
		}
		if _, exists := os.LookupEnv(key); !exists {
			_ = os.Setenv(key, value)
		}
	}
	return scanner.Err()
}

func (c *Config) validate() error {
	if c.ListenAddr == "" {
		return fmt.Errorf("MOONENGINE_LISTEN vacío")
	}
	if c.MaxBodyBytes <= 0 {
		return fmt.Errorf("MOONENGINE_MAX_BODY_BYTES debe ser mayor que 0")
	}
	if c.MaxFrameBytes <= 0 {
		return fmt.Errorf("MOONENGINE_MAX_FRAME_BYTES debe ser mayor que 0")
	}
	if c.ShutdownTimeout <= 0 {
		return fmt.Errorf("MOONENGINE_SHUTDOWN_TIMEOUT_MS debe ser mayor que 0")
	}
	switch c.LogLevel {
	case "debug", "info", "warn", "error":
	default:
		return fmt.Errorf("MOONENGINE_LOG_LEVEL inválido: %q", c.LogLevel)
	}
	if err := validateURL("MOONENGINE_STT_URL", c.STTURL); err != nil {
		return err
	}
	return validateURL("MOONENGINE_TRANSLATE_URL", c.TranslateURL)
}

func validateURL(name, value string) error {
	if value == "" {
		return nil
	}
	u, err := url.Parse(value)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return fmt.Errorf("%s inválida: %q", name, value)
	}
	return nil
}

func parseKeys(raw string) (map[string]string, error) {
	keys := make(map[string]string)
	if strings.TrimSpace(raw) == "" {
		return nil, fmt.Errorf("MOONENGINE_API_KEYS vacío: formato nombre:clave,nombre2:clave2")
	}
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		name, key, ok := strings.Cut(part, ":")
		name = strings.TrimSpace(name)
		key = strings.TrimSpace(key)
		if !ok || name == "" || key == "" {
			return nil, fmt.Errorf("entrada inválida en MOONENGINE_API_KEYS: %q", part)
		}
		if len(key) < 16 {
			return nil, fmt.Errorf("la clave de %q debe tener al menos 16 caracteres", name)
		}
		if _, exists := keys[name]; exists {
			return nil, fmt.Errorf("nombre repetido en MOONENGINE_API_KEYS: %q", part)
		}
		keys[name] = key
	}
	if len(keys) == 0 {
		return nil, fmt.Errorf("MOONENGINE_API_KEYS sin entradas válidas")
	}
	return keys, nil
}

func getString(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func getInt64(key string, fallback int64) (int64, error) {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback, nil
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%s no es un entero: %q", key, v)
	}
	return n, nil
}
