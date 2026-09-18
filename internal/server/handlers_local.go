package server

import (
	"context"
	"net/http"

	"github.com/konglong87/go-e2e/internal/skills"
)

func localSkillsHandler(opts Options) http.HandlerFunc {
	listSkills := opts.LocalSkillsFunc
	if listSkills == nil {
		listSkills = func(_ context.Context) ([]skills.Skill, error) {
			return skills.List(opts.Workspace)
		}
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if !authorize(w, r, opts.AuthToken) {
			return
		}
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		items, err := listSkills(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if items == nil {
			items = []skills.Skill{}
		}
		writeJSON(w, map[string]any{
			"workspace": opts.Workspace,
			"data":      items,
		})
	}
}
