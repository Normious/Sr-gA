package auth

import (
	"context"
	"net/http"

	"github.com/Normious/Sr-gA/internal/db"
)

type contextKey string

const projectContextKey contextKey = "project"

func Middleware(database *db.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			apiKey := r.Header.Get("X-API-Key")
			if apiKey == "" {
				http.Error(w, `{"success":false,"error":"Missing X-API-Key header"}`, http.StatusUnauthorized)
				return
			}

			project, err := database.GetProjectByAPIKey(r.Context(), apiKey)
			if err != nil {
				http.Error(w, `{"success":false,"error":"Invalid or inactive API Key"}`, http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), projectContextKey, project)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func ProjectFromContext(ctx context.Context) (*db.Project, bool) {
	p, ok := ctx.Value(projectContextKey).(*db.Project)
	return p, ok
}
