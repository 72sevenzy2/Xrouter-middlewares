package middleware

import (
	"strings"
	"net/http"
	"github.com/72sevenzy2/http-router/core"
	"github.com/72sevenzy2/json-parser/response"
)

// bearer auth middleware (this includes having a bearer token which will then be compared to the authkey )

func BearerAuth(AuthKey string) core.Middleware {
	return func(hf core.HandlerFunc) core.HandlerFunc {
		return func(w http.ResponseWriter, r *core.Request) {
			authLab := r.Header.Get("Authorization") // grabbing the token

			var token string
			if v := strings.Contains(AuthKey, "Bearer "); v {
				token = strings.TrimPrefix(authLab, "Bearer ") // removing the "bearer " part of the token to then compare it to the authkey

				if token != AuthKey {
					response.JSON(w, http.StatusForbidden, nil, http.StatusText(http.StatusForbidden))
					return
				}

				hf(w, r) // next handler
			}

			// continuing if "Bearer " doesnt include in the authkey.
			if AuthKey != authLab { // check if the authkey is matching
				response.JSON(w, http.StatusForbidden, nil, http.StatusText(http.StatusForbidden))
				return            // exit the request
			}

			hf(w, r) // continue to next handler
		}
	}
}
