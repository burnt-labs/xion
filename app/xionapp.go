package app

import (
	"net/http"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/gorilla/mux"

	"github.com/cosmos/cosmos-sdk/client"

	"github.com/burnt-labs/xion/client/docs"
)

// overrideWasmVariables overrides the wasm variables to:
//   - allow for larger wasm files
func overrideWasmVariables() {
	// Override Wasm size limitation from WASMD.
	wasmtypes.MaxWasmSize = 2 * 1024 * 1024
	wasmtypes.MaxProposalWasmSize = wasmtypes.MaxWasmSize
}

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
