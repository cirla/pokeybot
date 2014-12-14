package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/codegangsta/negroni"
	"github.com/gorilla/mux"
	"github.com/unrolled/secure"

	"pokeybot/pokey"
)

func HomeHandler(w http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(w, "PokeyBot")
}

func main() {
	router := mux.NewRouter()
	router.HandleFunc("/", HomeHandler)

	secureMiddleware := secure.New(secure.Options{
		AllowedHosts:    []string{"thelunchtrain.herokuapp.com"},
		SSLRedirect:     true,
		SSLProxyHeaders: map[string]string{"X-Forwarded-Proto": "https"},
		IsDevelopment:   true,
	})

	pokey := pokey.New()
	pokey.Route(router.PathPrefix("/pokey").Subrouter())

	n := negroni.New(
		negroni.NewRecovery(),
		negroni.NewLogger(),
		negroni.HandlerFunc(secureMiddleware.HandlerFuncWithNext),
	)

	n.UseHandler(router)

	n.Run(":" + os.Getenv("PORT"))
}
