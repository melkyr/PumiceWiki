package handler

import (
	"encoding/xml"
	"fmt"
	"go-wiki-app/internal/service"
	"net/http"
)

// SeoHandler holds dependencies for SEO-related handlers.
type SeoHandler struct {
	pageService service.PageServicer
}

// NewSeoHandler creates a new SeoHandler.
func NewSeoHandler(ps service.PageServicer) *SeoHandler {
	return &SeoHandler{pageService: ps}
}

// robotsHandler serves a static robots.txt file.
func (h *SeoHandler) robotsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintln(w, "User-agent: *")
	fmt.Fprintln(w, "Allow: /")
	fmt.Fprintln(w, "")
	// In a real app, you would get the domain from config.
	fmt.Fprintln(w, "Sitemap: http://localhost:8080/sitemap.xml")
}

const (
	sitemapDateFormat = "2006-01-02"
	baseURL           = "http://localhost:8080/view/" // In a real app, get this from config
)

type sitemapURL struct {
	XMLName xml.Name `xml:"url"`
	Loc     string   `xml:"loc"`
	LastMod string   `xml:"lastmod"`
}

type urlSet struct {
	XMLName xml.Name     `xml:"urlset"`
	Xmlns   string       `xml:"xmlns,attr"`
	URLs    []sitemapURL `xml:"url"`
}

// sitemapHandler generates and serves a dynamic sitemap.xml.
func (h *SeoHandler) sitemapHandler(w http.ResponseWriter, r *http.Request) {
	pages, err := h.pageService.GetAllPages(r.Context())
	if err != nil {
		http.Error(w, "Failed to retrieve pages for sitemap", http.StatusInternalServerError)
		return
	}

	sitemap := urlSet{
		Xmlns: "http://www.sitemaps.org/schemas/sitemap/0.9",
		URLs:  make([]sitemapURL, len(pages)),
	}

	for i, page := range pages {
		sitemap.URLs[i] = sitemapURL{
			Loc:     baseURL + page.Title,
			LastMod: page.UpdatedAt.Format(sitemapDateFormat),
		}
	}

	w.Header().Set("Content-Type", "application/xml")
	w.Write([]byte(xml.Header))
	encoder := xml.NewEncoder(w)
	encoder.Indent("", "  ")
	if err := encoder.Encode(sitemap); err != nil {
		http.Error(w, "Failed to generate sitemap XML", http.StatusInternalServerError)
		return
	}
}
