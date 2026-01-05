package main

import (
	"github.com/mdanie17/spotifytop/config"
	"github.com/mdanie17/spotifytop/web"
	"github.com/rs/zerolog/log"
)

func main() {
	cfg, err := config.GetServerConfig()
	if err != nil {
		log.Error().Err(err).Msg("could not get config")
		return
	}

	server := web.Web{
		ServerHostName: cfg.ServerHost,
		ServerPort:     cfg.ServerPort,
		State:          cfg.SpotifyState,
		RedirectHost:   cfg.SpotifyRedirectURI,
		CookieKey:      []byte(cfg.CookieKey),
		Clientkey:      cfg.SpotifyClientKey,
		Secretkey:      cfg.SpotifySecretKey,
	}

	server.New()
	server.Run()
}
