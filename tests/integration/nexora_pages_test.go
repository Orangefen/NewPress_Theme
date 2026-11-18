package httpserver

import (
	"bytes"
	"fmt"
	"html"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"newpress.local/newpress/internal/comment"
	"newpress.local/newpress/internal/config"
	"newpress.local/newpress/internal/content"
	"newpress.local/newpress/internal/plugin"
	"newpress.local/newpress/internal/presentation"
)

func nexoraPageContent(t *testing.T) (*content.MemoryRepository, []content.Post) {
	t.Helper()
	repo := content.NewMemoryRepository(false)
	posts := []content.Post{}
	for i := 0; i < 21; i++ {
		post, err := repo.Create(t.Context(), content.PostInput{
			Type: content.TypePost, Status: content.StatusPublish, Title: fmt.Sprintf("研究 & 发展 %d", i), Slug: fmt.Sprintf("research-%d", i),
			Category: "Engineering", TagNames: []string{"innovation"}, AuthorName: "NewPress Team", CommentStatus: content.CommentOpen,
			PublishedAt: time.Now().Add(-time.Duration(i+1) * time.Hour), ContentSource: "<h2>背景</h2><p>研究正文。</p><h3>细节</h3><p>阅读与讨论。</p>",
		})
		if err != nil {
			t.Fatal(err)
		}
		posts = append(posts, post)
	}
	return repo, posts
}

func nexoraPageGET(t *testing.T, h http.Handler, path string) string {
	t.Helper()
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "https://example.test"+path, nil)
	r.RemoteAddr = "198.51.100.55:4000"
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK || strings.Contains(w.Body.String(), "#ZgotmplZ") || strings.Contains(w.Body.String(), "<no value>") {
		t.Fatalf("%s: status=%d body=%s", path, w.Code, w.Body.String())
	}
	return w.Body.String()
}

func TestNexoraListFilterAndArchiveState(t *testing.T) {
	m := nexoraFixtureTheme(t)
	repo, _ := nexoraPageContent(t)
	s := New(config.Config{SiteTitle: "Nexora", PublicURL: "https://example.test", ActiveTheme: "nexora"}, repo, m, plugin.NewManager(t.TempDir()))
	snapshot := s.siteSettings.Snapshot()
	snapshot.Settings.Reading.ExplorePosts = 10
	if _, fields, err := s.siteSettings.Save(snapshot.Revision, snapshot.Settings); err != nil || len(fields) > 0 {
		t.Fatal(fields, err)
	}
	h := s.routes()
	query := url.QueryEscape("研究 & 发展")
	body := html.UnescapeString(nexoraPageGET(t, h, "/explore/?q="+query+"&category=engineering"))
	for _, wanted := range []string{`name="category" value="engineering"`, `value="研究 & 发展"`, `href="?category=engineering&q=` + query + `" aria-current="page"`, `href="?category=engineering"`, `href="?q=` + query + `"`, `aria-disabled="true">上一页`, `rel="next" href="/explore/?category=engineering&page=2&q=` + query + `"`} {
		if !strings.Contains(body, wanted) {
			t.Fatalf("combined search lost selected filter, query or pagination: %s; toolbar=%s", wanted, regexp.MustCompile(`(?s)<div class="listing-toolbar">.*?<div class="news-grid">`).FindString(body))
		}
	}
	last := nexoraPageGET(t, h, "/explore/?q="+query+"&category=engineering&page=3")
	if !strings.Contains(last, `aria-disabled="true">下一页`) || strings.Contains(last, `rel="next"`) {
		t.Fatal("last page exposes a next-page link")
	}
	categoryNav := regexp.MustCompile(`(?s)<nav class="category-links".*?</nav>`)
	for _, archive := range []string{"/categories/engineering/", "/tags/innovation/"} {
		body = nexoraPageGET(t, h, archive)
		nav := categoryNav.FindString(body)
		if !strings.Contains(nav, `href="`+archive+`" aria-current="page"`) || strings.Contains(nav, `href="/explore/" aria-current="page"`) {
			t.Fatalf("archive %s has incorrect selected category: %s", archive, nav)
		}
		if !strings.Contains(body, "搜索全站新闻与文章") || !strings.Contains(body, `rel="next" href="`+archive+`?page=2"`) {
			t.Fatal("archive search scope or pagination is incorrect")
		}
	}
}

func TestNexoraSeriesPagesAndReadingOrder(t *testing.T) {
	m := nexoraFixtureTheme(t)
	repo, posts := nexoraPageContent(t)
	for i := 0; i < 25; i++ {
		_, err := repo.SaveSeries(t.Context(), 0, content.Mutation{}, content.SeriesInput{Kind: content.SeriesKindTopic, Status: content.StatusPublish, Title: fmt.Sprintf("研究专题 %d", i), Slug: fmt.Sprintf("topic-%d", i), PostIDs: []int64{posts[0].ID, posts[1].ID}})
		if err != nil {
			t.Fatal(err)
		}
	}
	_, err := repo.SaveSeries(t.Context(), 0, content.Mutation{}, content.SeriesInput{Kind: content.SeriesKindSequence, Status: content.StatusPublish, Title: "研究系列", Slug: "research-sequence", PostIDs: []int64{posts[0].ID, posts[1].ID, posts[2].ID}})
	if err != nil {
		t.Fatal(err)
	}
	s := New(config.Config{SiteTitle: "Nexora", PublicURL: "https://example.test", ActiveTheme: "nexora"}, repo, m, plugin.NewManager(t.TempDir()))
	h := s.routes()
	query := url.QueryEscape("研究")
	first := nexoraPageGET(t, h, "/series/?kind=topic&q="+query)
	if !strings.Contains(first, `data-theme="nexora"`) || !strings.Contains(first, `name="kind" value="topic"`) || !strings.Contains(first, `aria-disabled="true">上一页`) || !strings.Contains(first, `href="/series/?kind=topic&amp;page=2&amp;q=`+query+`" rel="next"`) {
		t.Fatal("series index did not preserve filters or first-page boundary")
	}
	last := nexoraPageGET(t, h, "/series/?kind=topic&q="+query+"&page=2")
	if strings.Contains(last, `rel="next"`) || !strings.Contains(last, `aria-disabled="true">下一页`) {
		t.Fatal("series index exposed a next link on its final page")
	}
	detail := nexoraPageGET(t, h, "/series/research-sequence/")
	if !strings.Contains(detail, `class="chapter-list"`) || !(strings.Index(detail, `href="/posts/research-0"`) < strings.Index(detail, `href="/posts/research-1"`) && strings.Index(detail, `href="/posts/research-1"`) < strings.Index(detail, `href="/posts/research-2"`)) {
		t.Fatal("series chapters were not rendered in editorial order")
	}
	article := nexoraPageGET(t, h, "/posts/research-1")
	if !strings.Contains(article, `href="/posts/research-0" rel="prev"`) || !strings.Contains(article, `href="/posts/research-2" rel="next"`) || !strings.Contains(article, `data-reading-toc hidden`) || !strings.Contains(article, `data-article-body`) {
		t.Fatal("article reading controls or ordered series navigation are missing")
	}
}

func TestNexoraCommentReplyValidationPreservesDraft(t *testing.T) {
	m := nexoraFixtureTheme(t)
	repo, posts := nexoraPageContent(t)
	s := New(config.Config{SiteTitle: "Nexora", PublicURL: "https://example.test", ActiveTheme: "nexora"}, repo, m, plugin.NewManager(t.TempDir()))
	snapshot := s.siteSettings.Snapshot()
	snapshot.Settings.Discussion.RequireLogin = false
	if _, fields, err := s.siteSettings.Save(snapshot.Revision, snapshot.Settings); err != nil || len(fields) > 0 {
		t.Fatal(fields, err)
	}
	parent, err := s.comments.Create(t.Context(), comment.CreateInput{PostID: posts[0].ID, AuthorName: "读者", AuthorEmail: "reader@example.test", ContentSource: "已有评论", Status: comment.StatusApproved})
	if err != nil {
		t.Fatal(err)
	}
	h := s.routes()
	body := nexoraPageGET(t, h, "/posts/research-0")
	form := url.Values{"form_token": {extractCommentToken(t, body)}, "parent_id": {fmt.Sprint(parent.ID)}, "website": {""}, "author_name": {"回复者"}, "email": {"invalid-email"}, "content": {"应当保留的回复正文"}}
	w := servePublicCommentForm(h, "research-0", form, "https://example.test", "198.51.100.55:4000")
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("invalid comment: %d %s", w.Code, w.Body.String())
	}
	for _, wanted := range []string{`name="parent_id" value="` + fmt.Sprint(parent.ID) + `"`, "应当保留的回复正文", `aria-invalid="true" aria-describedby="comment-error-email"`, "data-reply-cancel"} {
		if !strings.Contains(w.Body.String(), wanted) {
			t.Fatalf("failed comment lost its reply context or field error: %s", wanted)
		}
	}
}

func TestNexoraAccountCommentAndPreviewBoundaries(t *testing.T) {
	m := nexoraFixtureTheme(t)
	p := presentation.Fixtures()["article"]
	p.Settings, p.Now = m.SettingsSnapshot().Values, time.Now()
	p.Modules = map[string]bool{"comments": true, "series": true}
	p.Comments.AccountName = "已登录读者"
	var out bytes.Buffer
	if err := m.Render(&out, "single", p); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "已登录读者") || strings.Contains(out.String(), `name="author_name"`) || strings.Contains(out.String(), `name="email"`) || !strings.Contains(out.String(), `data-comment-form`) {
		t.Fatal("signed-in users must post with their account identity without mandatory guest fields")
	}
	p.IsPreview = true
	out.Reset()
	if err := m.Render(&out, "single", p); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), `data-comment-form`) || strings.Contains(out.String(), `data-copy-link`) || strings.Contains(out.String(), `href="#comments"`) {
		t.Fatal("draft preview exposed public commenting or sharing controls")
	}
}
