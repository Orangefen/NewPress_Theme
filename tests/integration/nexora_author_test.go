package httpserver

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"golang.org/x/net/html"
	"newpress.local/newpress/internal/config"
	"newpress.local/newpress/internal/content"
	"newpress.local/newpress/internal/plugin"
)

func TestNexoraAuthorSearchRetainsAuthorAndFilterContext(t *testing.T) {
	_, _, identity, repository, _, owner := newPersistentAdminTestServer(t)
	author, _, _ := createAndLoginTestUser(t, identity, owner, "nexora-author", "nexora-author@example.test", "Nexora 作者", "author")
	create := func(title, slug, category string, authorID int64, status string) {
		t.Helper()
		_, err := repository.Create(t.Context(), content.PostInput{Type: content.TypePost, Status: status, Title: title, Slug: slug, Category: category, AuthorID: authorID, AuthorName: author.DisplayName, PublishedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)})
		if err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 13; i++ {
		create(fmt.Sprintf("needle %02d", i), fmt.Sprintf("nexora-author-%02d", i), "Research Lab", author.ID, content.StatusPublish)
	}
	create("other research", "nexora-author-unfiltered", "Research Lab", author.ID, content.StatusPublish)
	create("needle product", "nexora-author-product", "Product News", author.ID, content.StatusPublish)
	create("needle foreign owner", "nexora-foreign-owner", "Research Lab", owner.UserID, content.StatusPublish)
	create("needle private", "nexora-author-private", "Research Lab", author.ID, content.StatusPrivate)
	m := nexoraFixtureTheme(t)
	s := New(config.Config{SiteTitle: "NEXORA", PublicURL: "https://example.test", ActiveTheme: "nexora"}, repository, m, plugin.NewManager(t.TempDir()), identity)
	h := s.routes()
	path := fmt.Sprintf("/authors/%d/", author.ID)
	get := func(route string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "https://example.test"+route, nil))
		return w
	}
	read := func(w *httptest.ResponseRecorder) (map[string]string, []string) {
		t.Helper()
		if w.Code != http.StatusOK {
			t.Fatalf("author HTTP status = %d: %s", w.Code, w.Body)
		}
		if strings.Contains(w.Body.String(), "nexora-foreign-owner") || strings.Contains(w.Body.String(), "nexora-author-private") {
			t.Fatal("author search exposed another owner's or a private post")
		}
		document, err := html.Parse(strings.NewReader(w.Body.String()))
		if err != nil {
			t.Fatal(err)
		}
		inputs, links := map[string]string{}, []string{}
		var visit func(*html.Node)
		visit = func(n *html.Node) {
			if n.Type == html.ElementNode {
				attributes := map[string]string{}
				for _, a := range n.Attr {
					attributes[a.Key] = a.Val
				}
				if n.Data == "input" && attributes["name"] != "" {
					inputs[attributes["name"]] = attributes["value"]
				}
				if n.Data == "a" {
					links = append(links, attributes["href"])
				}
				if n.Data == "form" && attributes["class"] == "inline-search" && attributes["action"] != "" && attributes["action"] != path {
					t.Fatalf("author form escaped context: %s", attributes["action"])
				}
			}
			for child := n.FirstChild; child != nil; child = child.NextSibling {
				visit(child)
			}
		}
		visit(document)
		return inputs, links
	}
	for _, query := range []string{"", "?q=&category=", "?q=%20%20&category=%20%20"} {
		w := get(path + query)
		inputs, _ := read(w)
		if inputs["q"] != "" || inputs["category"] != "" || !strings.Contains(w.Body.String(), "共 15 篇文章") {
			t.Fatalf("empty filters were not normalized: %v", inputs)
		}
	}
	query := url.Values{"q": {" needle "}, "category": {" Research Lab "}, "sort": {"title"}}
	w := get(path + "?" + query.Encode())
	inputs, links := read(w)
	if inputs["q"] != "needle" || inputs["category"] != "research-lab" || !strings.Contains(w.Body.String(), "找到 13 篇文章") {
		t.Fatalf("filter context missing: %v", inputs)
	}
	next, clearQuery, switchCategory := "", false, false
	for _, link := range links {
		parsed, err := url.Parse(link)
		if err != nil {
			t.Fatal(err)
		}
		values := parsed.Query()
		if parsed.Path == path && values.Get("page") == "2" {
			if values.Get("q") != "needle" || values.Get("category") != "Research Lab" {
				t.Fatalf("pagination lost filters: %s", link)
			}
			next = link
		}
		if parsed.Path == "" && values.Get("category") == "research-lab" && values.Get("q") == "" {
			clearQuery = true
		}
		if parsed.Path == "" && values.Get("category") == "product-news" && values.Get("q") == "needle" {
			switchCategory = true
		}
	}
	if next == "" || !clearQuery || !switchCategory {
		t.Fatalf("missing scoped controls: next=%q clear=%v switch=%v", next, clearQuery, switchCategory)
	}
	second := get(next)
	read(second)
	if !strings.Contains(second.Body.String(), "/posts/nexora-author-12") || strings.Contains(second.Body.String(), "/posts/nexora-author-00") {
		t.Fatal("author pagination did not produce the second filtered page")
	}
	for _, suffix := range []string{"?q=a&q=b", "?category=a&category=b", "?page=0", "?q=" + strings.Repeat("x", 81)} {
		if bad := get(path + suffix); bad.Code != http.StatusBadRequest {
			t.Fatalf("invalid author query %s returned %d", suffix, bad.Code)
		}
	}
}
