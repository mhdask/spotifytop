package web

import (
	"embed"
	"encoding/gob"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"strconv"
	"strings"
	"text/template"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/rs/zerolog/log"
	"github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
)

var (
	defaultTimeLimit      = "medium_term"
	defaultResultLimit    = 20
	cookieKeyFlashMessage = "flash-session"
	//go:embed static/*
	staticFS embed.FS
)

func init() {
	// Need to register FlashMessage struct to
	// later be encoded/decoded by session.AddFlash()
	gob.Register(flashMessage{})
}

type Web struct {
	Router *mux.Router

	CookieKey []byte
	Cookies   *sessions.CookieStore

	ServerHostName string
	ServerPort     int

	Templates map[string]*template.Template

	State string
	Auth  *spotifyauth.Authenticator
	//RedirectHost is used to specify where spotify redirects
	//needs to specify both hostname and port if needed, e.g. "example.org:8000/toptracks"
	RedirectHost string
	Clientkey    string
	Secretkey    string

	Clients map[string]*spotify.Client
}

func (w *Web) New() {
	if w.Router == nil {
		w.Router = mux.NewRouter()
		w.Routes(w.Router)
	}

	if w.CookieKey == nil {
		log.Fatal().Msg("no cookiekey specified")
	}

	if w.Cookies == nil {
		w.Cookies = sessions.NewCookieStore(w.CookieKey)
	}

	if w.ServerHostName == "" {
		w.ServerHostName = "localhost"
		log.Info().Msg("empty hostname, defaulting to localhost")
	}

	if w.ServerPort == 0 {
		w.ServerPort = 8080
		log.Info().Msg("empty serverport, defaulting to 8080")
	}

	if w.Templates == nil {
		w.Templates = make(map[string]*template.Template)

		w.parseTemplate("topartists", "")
		w.parseTemplate("frontpage", "")
		w.parseTemplate("toptracks", "")
	}

	if w.State == "" {
		log.Fatal().Msg("you have to set a state string")
		return
	}

	if w.RedirectHost == "" {
		w.RedirectHost = "localhost"
		log.Info().Msg("empty redirect hostname, defaulting to http://localhost")
	} else if !strings.Contains(w.RedirectHost, "http") {
		log.Fatal().Msg("redirecthost needs to specify either http or https")
	}

	if w.Clientkey == "" {
		if w.Clientkey = os.Getenv("SPOTIFY_ID"); w.Clientkey == "" {
			log.Fatal().Msg("you have to set a client key")
		}
	}

	if w.Secretkey == "" {
		if w.Secretkey = os.Getenv("SPOTIFY_SECRET"); w.Secretkey == "" {
			log.Fatal().Msg("you have to set a secret key")
		}
	}

	if w.Clients == nil {
		w.Clients = make(map[string]*spotify.Client)
	}
}

func (w *Web) Routes(r *mux.Router) {
	staticSub, err := fs.Sub(staticFS, "static")
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create static sub filesystem")
	}

	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.FS(staticSub))))

	r.HandleFunc("/", w.handleFrontPage).Methods("GET")
	// r.HandleFunc("/topartistsauth", w.handleAuthenticateArtists)
	r.HandleFunc("/topartists", w.handleTopArtists)
	// r.HandleFunc("/toptracksauth", w.handleAuthenticateTracks)
	r.HandleFunc("/toptracks", w.handleTopTracks)
	r.HandleFunc("/createplaylist", w.handleCreatePlaylist)
	r.HandleFunc("/form", w.handleForm)
	r.HandleFunc("/login", w.handleAuth)
	r.HandleFunc("/logout", w.handleLogout)
	r.HandleFunc("/authenticated", w.handleAuthenticated)
}

func (w *Web) Run() {
	log.Info().Msgf("Starting server on port %s:%d", w.ServerHostName, w.ServerPort)
	if err := http.ListenAndServe(fmt.Sprintf("%s:%d", w.ServerHostName, w.ServerPort), w.Router); err != nil {
		log.Fatal().Err(err).Msg("failed to start webserver")
	}
}

func (w *Web) handleFrontPage(rw http.ResponseWriter, r *http.Request) {
	settings := w.cookieGetSettings(rw, r)
	state, err := w.cookieGetState(rw, r)
	if err != nil {
		w.templateExec(rw, r, "frontpage", TmplData{Settings: settings, LoggedIn: false})
		return
	}

	client, ok := w.Clients[state]
	if !ok {
		w.templateExec(rw, r, "frontpage", TmplData{Settings: settings, LoggedIn: false})
		return
	}

	user, err := client.CurrentUser(r.Context())
	if err != nil {
		w.addFlash(rw, r, flashMessage{flashLevelDanger, "Could not communicate with Spotify - Try clearing cache and trying again"})
		redirectReferer(rw, r)
		return
	}

	Data := TmplData{
		Settings: settings,
		User:     user.User,
		LoggedIn: true,
	}
	w.templateExec(rw, r, "frontpage", Data)

}

func (w *Web) handleForm(rw http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		log.Error().Err(err).Msg("could not parse settings form")
		return
	}

	timelimit := r.FormValue("timecheck")
	resultlimit := r.FormValue("limit")
	resultlimitint, err := strconv.Atoi(resultlimit)
	if err != nil {
		w.addFlash(rw, r, flashMessage{flashLevelWarning, "You have to select a valid number of results"})
		resultlimitint = defaultResultLimit
	}

	w.cookieSetSettings(rw, r, Opts{timelimit, resultlimitint})
	redirectReferer(rw, r)
}

func (w *Web) handleAuthenticated(rw http.ResponseWriter, r *http.Request) {
	state, err := w.cookieGetState(rw, r)
	if err != nil {
		w.addFlash(rw, r, flashMessage{flashLevelDanger, "Could not authenticate - Clear cache and try again"})
		redirectReferer(rw, r)
		return
	}

	w.createClient(rw, r, state)
	http.Redirect(rw, r, "/", http.StatusFound)
}

func (w *Web) handleLogout(rw http.ResponseWriter, r *http.Request) {
	state, err := w.cookieGetState(rw, r)
	if err != nil {
		w.addFlash(rw, r, flashMessage{flashLevelDanger, "Something went wrong logging you out"})
		http.Redirect(rw, r, "/", http.StatusFound)
	}

	w.Clients[state] = nil

	w.deleteCookie(rw, r, "state")
	w.addFlash(rw, r, flashMessage{flashLevelSuccess, "Successfully logged you out!"})
	http.Redirect(rw, r, "/", http.StatusFound)
}
