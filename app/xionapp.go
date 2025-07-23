package app

import (
	"github.com/burnt-labs/xion/client/docs"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/gorilla/mux"
	"net/http"
)

// RegisterSwaggerAPI registers swagger route with API Server
func RegisterSwaggerAPI(_ client.Context, rtr *mux.Router, swaggerEnabled bool) error {
	if swaggerEnabled {
		docsServer := http.FileServer(http.FS(docs.Docs))
		rtr.Handle("/static", docsServer)
		rtr.Handle("/static/", docsServer)
		rtr.Handle("/static/swagger.json", docsServer)
		rtr.Handle("/static/openapi.json", docsServer)

		rtr.PathPrefix("/static").Handler(http.StripPrefix("/static/", docsServer))
		rtr.PathPrefix("/static/").Handler(http.StripPrefix("/static/", docsServer))

		rtr.Handle("/", http.RedirectHandler("/static/", http.StatusMovedPermanently))
		rtr.Handle("/swagger", http.RedirectHandler("/static/", http.StatusMovedPermanently))
		rtr.Handle("/swagger/", http.RedirectHandler("/static/", http.StatusMovedPermanently))
	}
	return nil
}
