package handlers

import "github.com/xuroi/xuroi/api/internal/markdown"

func (a *API) renderPostHTML(source string) string {
	return markdown.RenderUGC(source, a.siteCfg.Moderation.WordFilter)
}