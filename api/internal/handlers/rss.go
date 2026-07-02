package handlers

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/xuroi/xuroi/api/internal/models"
)

type rssFeed struct {
	XMLName xml.Name   `xml:"rss"`
	Version string     `xml:"version,attr"`
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Title       string    `xml:"title"`
	Link        string    `xml:"link"`
	Description string    `xml:"description"`
	Items       []rssItem `xml:"item"`
}

type rssItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
	GUID        string `xml:"guid"`
}

func (a *API) categoryFeed(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	limit := 25
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 50 {
			limit = n
		}
	}
	viewer, err := a.viewerFromRequest(r)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	page, err := a.reader.CategoryBySlug(r.Context(), slug, 1, limit, viewer)
	if err != nil {
		writeError(w, http.StatusNotFound, "category not found")
		return
	}
	siteURL := strings.TrimRight(a.siteCfg.Site.URL, "/")
	feed := rssFeed{
		Version: "2.0",
		Channel: rssChannel{
			Title:       fmt.Sprintf("%s — %s", a.siteCfg.Site.Name, page.Category.Name),
			Link:        siteURL + page.Category.URL,
			Description: page.Category.Name + " latest threads",
		},
	}
	for _, t := range page.Threads {
		feed.Channel.Items = append(feed.Channel.Items, rssItem{
			Title:       t.Title,
			Link:        siteURL + t.URL,
			Description: fmt.Sprintf("%d replies · last activity %s", t.ReplyCount, t.LastActivityAt.Format(time.RFC1123)),
			PubDate:     t.LastActivityAt.UTC().Format(time.RFC1123),
			GUID:        siteURL + models.ThreadURL(t.Slug, t.ID),
		})
	}
	w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	_, _ = w.Write([]byte(xml.Header))
	_ = enc.Encode(feed)
}