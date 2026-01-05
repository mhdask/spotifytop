package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/rs/zerolog/log"
)

const (
	SoftwareName = "spotifytop"
)

type ServerConfig struct {
	// ServerHost specifies the host the server listens on
	ServerHost string
	// ServerPort specifies the port the server listens on
	ServerPort int
	// CookieKey specifies the key used for encoding the cookies
	CookieKey string

	// SpotifyState specifies the string spotify uses to generate unique URLs
	SpotifyState string
	// SpotifyRedirectURI specifies the URI that will be redirected
	// to from spotify upon succesful login
	// This needs to include protocol and port, e.g:
	// http://localhost:8080
	SpotifyRedirectURI string
	// SpotifyClientKey is the client key specified on the spotify
	// developer portal
	SpotifyClientKey string
	// SpotifySecretKey is the client key specified on the spotify
	// developer portal
	SpotifySecretKey string
}

func GetServerConfig() (ServerConfig, error) {
	cfg := ServerConfig{
		ServerHost:         getEnv("SPOTIFYTOP_SERVER_HOST", "0.0.0.0"),
		ServerPort:         getEnvInt("SPOTIFYTOP_SERVER_PORT", 8080),
		CookieKey:          getEnv("SPOTIFYTOP_COOKIE_KEY", "cookie"),
		SpotifyState:       getEnv("SPOTIFYTOP_SPOTIFY_STATE", "state"),
		SpotifyRedirectURI: os.Getenv("SPOTIFYTOP_SPOTIFY_REDIRECT_URI"),
		SpotifyClientKey:   os.Getenv("SPOTIFYTOP_SPOTIFY_CLIENT_KEY"),
		SpotifySecretKey:   os.Getenv("SPOTIFYTOP_SPOTIFY_SECRET_KEY"),
	}

	if cfg.SpotifyClientKey == "" || cfg.SpotifySecretKey == "" || cfg.SpotifyRedirectURI == "" {
		return cfg, fmt.Errorf("required spotify configuration missing")
	}

	log.Info().
		Str("host", cfg.ServerHost).
		Int("port", cfg.ServerPort).
		Msg("Loaded spotifytop config from environment")

	return cfg, nil
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		i, err := strconv.Atoi(v)
		if err != nil {
			log.Error().
				Str("key", key).
				Str("value", v).
				Err(err).
				Msg("invalid integer environment variable, using default")
			return def
		}

		return i
	}

	return def
}
