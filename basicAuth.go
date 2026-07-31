package middleware

import (
	"net/http"
	"github.com/72sevenzy2/http-router/core"
	"github.com/72sevenzy2/json-parser/helpers"
)

// basic auth middleware (this auth includes having a user and password inorder to access the endpoint)

func BasicAuth(user, password string) core.Middleware { // implements the middleware type which returns a handler
	return func(hf core.HandlerFunc) core.HandlerFunc {
		return func(w http.ResponseWriter, r *core.Request) {

			authUser, authPassword, ok := r.BasicAuth() // extracting the user and password and if it exists (ok) from the r.BasicAuth() func, which is a built in method in go to do so, instead of manually parsing it ourselves.

			if !ok || authUser != user || authPassword != password { // run the necessary logic
				helpers.Failed(w)
				return
			}

			hf(w, r) // continue to next handler
		}
	}
}
