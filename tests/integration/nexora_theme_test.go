package httpserver

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"newpress.local/newpress/internal/auth"
	"newpress.local/newpress/internal/comment"
	"newpress.local/newpress/internal/config"
	"newpress.local/newpress/internal/content"
	"newpress.local/newpress/internal/extension"
	"newpress.local/newpress/internal/plugin"
	"newpress.local/newpress/internal/theme"
	"newpress.local/newpress/internal/verification"
)

func nexoraFixtureTheme(t *testing.T) *theme.Manager {
	t.Helper()
	root := t.TempDir()
	source := os.Getenv("NEWPRESS_NEXORA_SOURCE")
	if source == "" { t.Fatal("set NEWPRESS_NEXORA_SOURCE via scripts/check_core.py") }
	if err := extension.CopyTree(t.Context(), source, filepath.Join(root, "nexora")); err != nil {
		t.Fatal(err)
	}
	m := theme.NewManager(root, "nexora", theme.WithSettingsDir(t.TempDir()))
	if err := m.Load(); err != nil {
		t.Fatal(err)
	}
	return m
}

func TestNexoraPublicTheme(t *testing.T) {
	m := nexoraFixtureTheme(t)
	repository := content.NewMemoryRepository(true)
	_, err := repository.Create(t.Context(), content.PostInput{Type: content.TypePage, Status: content.StatusPublish, Title: "企业介绍", Slug: "company", ContentSource: "<p>我们的企业介绍正文。</p>"})
	if err != nil {
		t.Fatal(err)
	}
	s := New(config.Config{SiteTitle: "NEXORA", PublicURL: "https://example.test", ActiveTheme: "nexora"}, repository, m, plugin.NewManager(t.TempDir()))
	h := s.routes()
	for _, route := range []string{"/", "/explore/", "/explore/?q=not-found-query", "/posts/welcome-to-newpress", "/pages/company", "/account/", "/admin/"} {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", route, nil)
		r.RemoteAddr = "127.0.0.1:1234"
		h.ServeHTTP(w, r)
		if w.Code != http.StatusOK || strings.Contains(w.Body.String(), "#ZgotmplZ") || strings.Contains(w.Body.String(), "<no value>") {
			t.Fatalf("%s: status=%d body=%s", route, w.Code, w.Body.String())
		}
		if route == "/pages/company" && (!strings.Contains(w.Body.String(), "我们的企业介绍正文") || strings.Contains(w.Body.String(), "comment-form")) {
			t.Fatal("corporate page body or comment ownership is incorrect")
		}
		if route == "/admin/" && strings.Contains(w.Body.String(), "/theme-assets/nexora/") {
			t.Fatal("theme assets leaked into the core admin")
		}
	}
	for _, asset := range []string{"css/nexora.css", "js/nexora.js", "fonts/nexora-serif.woff2", "images/headquarters.webp", "images/about-architecture.webp", "images/innovation-mountains.webp"} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", m.AssetURL(asset), nil))
		if w.Code != 200 || !strings.Contains(w.Header().Get("Cache-Control"), "immutable") {
			t.Fatalf("theme asset %s: %d", asset, w.Code)
		}
	}
	snapshot := s.siteSettings.Snapshot()
	settings := snapshot.Settings
	settings.Discussion.RequireLogin = false
	if _, fields, err := s.siteSettings.Save(snapshot.Revision, settings); err != nil || len(fields) != 0 {
		t.Fatal(fields, err)
	}
	article := serveCommentRequest(h, nil, "GET", "https://example.test/posts/welcome-to-newpress", "", "", "198.51.100.8:4000")
	token := extractCommentToken(t, article.Body.String())
	form := url.Values{"form_token": {token}, "parent_id": {"0"}, "website": {""}, "author_name": {"企业访客"}, "email": {"reader@example.test"}, "content": {"期待更多技术分享。"}}
	created := servePublicCommentForm(h, "welcome-to-newpress", form, "https://example.test", "198.51.100.8:4000")
	if created.Code != http.StatusSeeOther {
		t.Fatalf("comment submission: %d %s", created.Code, created.Body.String())
	}
	pending, err := s.comments.ListAdmin(t.Context(), comment.Query{Status: comment.StatusPending, Limit: 10})
	if err != nil || pending.Total != 1 {
		t.Fatalf("comment was not persisted: %v %v", pending, err)
	}
	store, err := verification.NewStore("")
	if err != nil {
		t.Fatal(err)
	}
	s.SetVerification(store)
	enableTestVerification(t, s)
	verifiedPage := serveCommentRequest(h, nil, "GET", "https://example.test/posts/welcome-to-newpress", "", "", "198.51.100.8:4000")
	if !strings.Contains(verifiedPage.Body.String(), "nexora-verification.js") || !strings.Contains(verifiedPage.Body.String(), "data-comment-form") || strings.Contains(verifiedPage.Body.String(), "fixture-private-secret") {
		t.Fatal("verified comments do not expose an independent safe adapter")
	}
	form.Set("content", "需要验证的第二条评论。")
	form.Set("form_token", extractCommentToken(t, verifiedPage.Body.String()))
	denied := servePublicCommentForm(h, "welcome-to-newpress", form, "https://example.test", "198.51.100.8:4000")
	if denied.Code != http.StatusUnprocessableEntity {
		t.Fatalf("missing verification was not rejected: %d", denied.Code)
	}
	proofRequest := httptest.NewRequest("POST", "https://example.test/api/v1/verification/verify", strings.NewReader(`{"scene":"comment","proof":{"token":"comment:nexora"}}`))
	proofRequest.RemoteAddr = "198.51.100.8:4000"
	proofRequest.Header.Set("Content-Type", "application/json")
	proofRequest.Header.Set("Origin", "https://example.test")
	proofResponse := httptest.NewRecorder()
	h.ServeHTTP(proofResponse, proofRequest)
	var proof struct{ Token string }
	if proofResponse.Code != 200 || json.Unmarshal(proofResponse.Body.Bytes(), &proof) != nil || proof.Token == "" {
		t.Fatalf("verification receipt failed: %d", proofResponse.Code)
	}
	form.Set("verification_token", proof.Token)
	accepted := servePublicCommentForm(h, "welcome-to-newpress", form, "https://example.test", "198.51.100.8:4000")
	if accepted.Code != http.StatusSeeOther {
		t.Fatalf("verified comment rejected: %d %s", accepted.Code, accepted.Body.String())
	}
}

// Opt-in browser fixture, with a separate theme installation, temporary
// settings, and in-memory demonstration content. It never opens production var.
func TestNexoraInteractiveUI(t *testing.T) {
	if os.Getenv("NEWPRESS_NEXORA_UI") != "1" {
		t.Skip("interactive browser fixture only")
	}
	m := nexoraFixtureTheme(t)
	users, err := auth.NewFileUserStore("")
	if err != nil {
		t.Fatal(err)
	}
	hash, err := auth.HashPassword("Nexora fixture 2026!")
	if err != nil {
		t.Fatal(err)
	}
	identity, err := auth.NewWithContext(t.Context(), auth.Config{Username: "owner", PasswordHash: hash, Users: users, BootstrapEmail: "owner@example.test", AbsoluteTTL: time.Hour, IdleTTL: 30 * time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	repository := content.NewMemoryRepository(false)
	for index, title := range []string{"让创新走出实验室，走进每一个真实场景", "以开放协作，连接产业发展的新可能", "面向未来，探索更可持续的技术路径"} {
		_, err := repository.Create(t.Context(), content.PostInput{Type: content.TypePost, Status: content.StatusPublish, Slug: []string{"innovation-in-action", "connected-future", "sustainable-technology"}[index], Title: title, Excerpt: "从一个想法出发，让技术的价值在实践中被看见。我们持续探索科技与产业交汇的更多可能。", AuthorName: "NEXORA", Category: "企业动态", ContentSource: "<p>创新的意义，在于为真实世界带来积极的改变。</p><h2>从技术出发，向价值前行</h2><p>我们关注技术，也关注使用技术的人。通过开放协作，将研究与实践连接起来，回应产业发展的真实需求。</p><blockquote>每一次向前，都始于一次有价值的探索。</blockquote><p>这是一篇隔离预览环境中的示例文章，可通过 NewPress 后台编辑、发布和管理。</p>", CommentStatus: "open", PublishedAt: time.Date(2026, 11, 16-index, 9, 0, 0, 0, time.UTC)})
		if err != nil {
			t.Fatal(err)
		}
	}
	_, err = repository.Create(t.Context(), content.PostInput{Type: content.TypePage, Status: content.StatusPublish, Slug: "company", Title: "以科技之名，探索更大的未来", ContentSource: "<h2>关于我们</h2><p>以专业构筑信任，用创新驱动未来。</p>"})
	if err != nil {
		t.Fatal(err)
	}
	s := New(config.Config{SiteTitle: "NEXORA", SiteTagline: "以科技之名，探索更大的未来。", PublicURL: "http://127.0.0.1:19106", ActiveTheme: "nexora"}, repository, m, plugin.NewManager(t.TempDir()), identity)
	listener, err := net.Listen("tcp", "127.0.0.1:19106")
	if err != nil {
		t.Fatal(err)
	}
	server := &http.Server{Handler: s.routes(), ReadHeaderTimeout: 5 * time.Second}
	go server.Serve(listener)
	defer server.Close()
	t.Log("Nexora isolated preview: http://127.0.0.1:19106/")
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-t.Context().Done():
			return
		case <-ticker.C:
			if stop := os.Getenv("NEWPRESS_NEXORA_UI_STOP"); stop != "" {
				if _, err := os.Stat(stop); err == nil {
					return
				}
			}
		}
	}
}
