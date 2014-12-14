package pokey

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/mux"

	"pokeybot/pokey/db"
)

type Pokey struct {
	db   db.Database
	rand *rand.Rand
}

func New() *Pokey {
	db, err := db.Open()
	if err != nil {
		panic(err)
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	return &Pokey{
		db:   db,
		rand: r,
	}
}

type pokeyHandlerFunc func(*Pokey, http.ResponseWriter, *http.Request)

type pokeyHandler struct {
	pokey       *Pokey
	handlerFunc pokeyHandlerFunc
}

func (h *pokeyHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	h.handlerFunc(h.pokey, w, req)
}

func (p *Pokey) makeHandler(h pokeyHandlerFunc) http.Handler {
	return &pokeyHandler{pokey: p, handlerFunc: h}
}

func homeHandler(p *Pokey, w http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(w, "Pokey API")
}

func comicsHandler(p *Pokey, w http.ResponseWriter, req *http.Request) {
	var comics []db.Comic
	p.db.LoadAllComics(&comics, true, true)

	w.Header().Set("Content-Type", "application/json")
	result, _ := json.Marshal(comics)
	fmt.Fprint(w, string(result))
}

func randomHandler(p *Pokey, w http.ResponseWriter, req *http.Request) {
	var comics []db.Comic
	p.db.LoadAllComics(&comics, false, false)
	comic := comics[p.rand.Intn(len(comics))]
	p.db.LoadImages(&comic)
	p.db.LoadTags(&comic)

	w.Header().Set("Content-Type", "application/json")
	result, _ := json.Marshal(comic)
	fmt.Fprint(w, string(result))
}

type slackMessage struct {
	Text        string `json:"text"`
	Channel     string `json:"channel,omitempty"`
	Username    string `json:"username,omitempty"`
	IconEmoji   string `json:"icon_emoji,omitempty"`
	UnfurlMedia bool   `json:"unfurl_media"`
}

type slackUser struct {
	Username string
	Icon     string
}

var SLACK_USERS = []slackUser{
	{"Pokey", ":pokey:"},
	{"Mr. Nutty", ":mrnutty:"},
	{"Little Girl", ":littlegirl:"},
	{"Skeptopotamus", ":skeptopotamus:"},
}

func slackHandler(p *Pokey, w http.ResponseWriter, req *http.Request) {
	slackToken := os.Getenv("SLACK_TOKEN_POKEY")
	slackWebhookUrl := os.Getenv("SLACK_WEBHOOK_URL_POKEY")

	w.Header().Set("Content-Type", "text/plain")

	token := req.FormValue("token")
	if token != slackToken {
		http.Error(w, "Invalid token.", http.StatusUnauthorized)
		return
	}

	//teamId := req.FormValue("team_id")
	//channelId := req.FormValue("channel_id")
	channelName := req.FormValue("channel_name")
	userId := req.FormValue("user_id")
	userName := req.FormValue("user_name")
	command := req.FormValue("command")
	text := req.FormValue("text")

	if command != "/pokey" {
		http.Error(w, "Invalid command.", http.StatusBadRequest)
		return
	}

	var params []string
	for _, token := range strings.Split(text, " ") {
		param := strings.TrimSpace(token)
		if param != "" {
			params = append(params, param)
		}
	}

	bot := SLACK_USERS[p.rand.Intn(len(SLACK_USERS))]
	if len(params) == 0 {
		// display a random comic
		var comics []db.Comic
		p.db.LoadAllComics(&comics, false, false)
		comic := comics[p.rand.Intn(len(comics))]
		p.db.LoadImages(&comic)

		var buffer bytes.Buffer
		buffer.WriteString(fmt.Sprintf("<@%s|%s>: <%s|%s>", userId, userName, comic.Url, comic.Title))
		//for _, img := range comic.Images {
		//    buffer.WriteString(fmt.Sprintf("\n<%s>", img.Url))
		//}

		msg := slackMessage{
			Text:        buffer.String(),
			Channel:     "#" + channelName,
			Username:    bot.Username,
			IconEmoji:   bot.Icon,
			UnfurlMedia: true,
		}
		payload, _ := json.Marshal(msg)

		resp, err := http.Post(slackWebhookUrl, "application/json", bytes.NewReader(payload))
		if err != nil || resp.StatusCode != http.StatusOK {
			defer resp.Body.Close()
			body, _ := ioutil.ReadAll(resp.Body)
			http.Error(w, fmt.Sprintf("Error posting to Slack Webhook URL: %s: %s", resp.Status, body), http.StatusInternalServerError)
			return
		}

		return
	}

	http.Error(w, fmt.Sprintf("Unsupported command: %s", params[0]), http.StatusBadRequest)
}

func initDbHandler(p *Pokey, w http.ResponseWriter, req *http.Request) {
	secretKey := os.Getenv("SECRET_KEY_POKEY")
	if secretKey != req.FormValue("secret_key") {
		http.Error(w, "Invalid secret key.", http.StatusUnauthorized)
		return
	}

	p.db.Init()
}

func clearDbHandler(p *Pokey, w http.ResponseWriter, req *http.Request) {
	secretKey := os.Getenv("SECRET_KEY_POKEY")
	if secretKey != req.FormValue("secret_key") {
		http.Error(w, "Invalid secret key.", http.StatusUnauthorized)
		return
	}

	p.db.Clear()
}

func populateDbHandler(p *Pokey, w http.ResponseWriter, req *http.Request) {
	secretKey := os.Getenv("SECRET_KEY_POKEY")
	if secretKey != req.FormValue("secret_key") {
		http.Error(w, "Invalid secret key.", http.StatusUnauthorized)
		return
	}

	p.db.Populate()
}

func (p *Pokey) Route(router *mux.Router) {
	router.Handle("/", p.makeHandler(homeHandler))
	router.Handle("/comics", p.makeHandler(comicsHandler))
	router.Handle("/random", p.makeHandler(randomHandler))
	router.Handle("/slack", p.makeHandler(slackHandler))

	dbRouter := router.PathPrefix("/db").Subrouter()
	dbRouter.Handle("/init", p.makeHandler(initDbHandler))
	dbRouter.Handle("/clear", p.makeHandler(clearDbHandler))
	dbRouter.Handle("/populate", p.makeHandler(populateDbHandler))
}
