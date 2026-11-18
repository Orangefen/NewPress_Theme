package httpserver

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
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
)

func TestDocsHubCommentIdentityAndReply(t *testing.T) {
	users, err := auth.NewFileUserStore("")
	if err != nil {
		t.Fatal(err)
	}
	password := "DocsHub fixture password 2026!"
	hash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatal(err)
	}
	identity, err := auth.NewWithContext(t.Context(), auth.Config{Username: "reader", PasswordHash: hash, Users: users, BootstrapEmail: "reader@example.test", BootstrapDisplayName: "文档测试读者", AbsoluteTTL: time.Hour, IdleTTL: 15 * time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	token, _, err := identity.Authenticate("docshub-tests", "reader", password)
	if err != nil {
		t.Fatal(err)
	}
	s := New(config.Config{SiteTitle: "DocsHub", PublicURL: "https://example.test", ActiveTheme: "docshub"}, docsHubContent(t), docsHubTheme(t), plugin.NewManager(t.TempDir()), identity)
	h := s.routes()
	cookie := &http.Cookie{Name: auth.CookieName, Value: token, Path: "/"}
	parent, err := s.comments.Create(t.Context(), comment.CreateInput{PostID: 1, AuthorName: "原评论作者", AuthorEmail: "parent@example.test", Status: comment.StatusApproved, ContentSource: "请问可以补充环境配置示例吗？"})
	if err != nil {
		t.Fatal(err)
	}
	page := serveCommentRequest(h, cookie, "GET", "https://example.test/posts/getting-started", "", "", "198.51.100.18:4000")
	body := page.Body.String()
	if page.Code != 200 || !strings.Contains(body, "文档测试读者") || !strings.Contains(body, `data-comment-reply=`) || strings.Contains(body, `id="comment-name"`) || strings.Contains(body, `id="comment-email"`) || !strings.Contains(body, `type="hidden" name="author_name"`) || !strings.Contains(body, `type="hidden" name="email"`) {
		t.Fatal("signed-in reader must use account identity and can reply without guest fields")
	}
	form := url.Values{"form_token": {extractCommentToken(t, body)}, "parent_id": {fmt.Sprint(parent.ID)}, "website": {""}, "author_name": {""}, "email": {""}, "content": {"我也尝试了这些步骤，可以按文档完成配置。"}}
	r := httptest.NewRequest("POST", "https://example.test/posts/getting-started/comments", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Set("Origin", "https://example.test")
	r.RemoteAddr = "198.51.100.18:4000"
	r.AddCookie(cookie)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusSeeOther {
		t.Fatalf("signed-in reply: %d %s", w.Code, w.Body.String())
	}
	items, err := s.comments.ListAdmin(t.Context(), comment.Query{Limit: 10})
	if err != nil || items.Total != 2 {
		t.Fatal("reply not saved", items, err)
	}
	var found bool
	for _, item := range items.Items {
		if item.ParentID == parent.ID && item.UserID > 0 {
			found = true
		}
	}
	if !found {
		t.Fatal("reply lost parent or authenticated identity")
	}
}

func TestDocsHubPagination(t *testing.T) {
	repository := docsHubContent(t)
	for index := 0; index < 25; index++ {
		if _, err := repository.Create(t.Context(), content.PostInput{Type: content.TypePost, Status: content.StatusPublish, Slug: fmt.Sprintf("pagination-%d", index), Title: fmt.Sprintf("分页文档 %02d", index), ContentSource: "<p>分页检查</p>"}); err != nil {
			t.Fatal(err)
		}
		if _, err := repository.SaveSeries(t.Context(), 0, content.Mutation{}, content.SeriesInput{Kind: content.SeriesKindSequence, Status: content.StatusPublish, Slug: fmt.Sprintf("pagination-series-%d", index), Title: fmt.Sprintf("分页系列 %02d", index)}); err != nil {
			t.Fatal(err)
		}
	}
	s := New(config.Config{SiteTitle: "DocsHub", PublicURL: "https://example.test", ActiveTheme: "docshub"}, repository, docsHubTheme(t), plugin.NewManager(t.TempDir()))
	h := s.routes()
	for _, row := range []struct {
		route          string
		previous, next bool
	}{
		{"/series/", false, true}, {"/series/?page=2", true, false},
		{"/explore/?q=" + url.QueryEscape("分页") + "&sort=title&view=list", false, true},
	} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", row.route, nil))
		body := w.Body.String()
		if w.Code != 200 || strings.Contains(body, `rel="prev"`) != row.previous || strings.Contains(body, `rel="next"`) != row.next {
			t.Fatalf("pagination %s: %d", row.route, w.Code)
		}
		if strings.HasPrefix(row.route, "/explore/") {
			match := regexp.MustCompile(`rel="next" href="([^"]+)"`).FindStringSubmatch(body)
			if len(match) != 2 {
				t.Fatal("next page link absent")
			}
			target, err := url.Parse(strings.ReplaceAll(match[1], "&amp;", "&"))
			if err != nil || target.Query().Get("q") != "分页" || target.Query().Get("sort") != "title" || target.Query().Get("view") != "list" {
				t.Fatal("pagination discarded filters", target, err)
			}
		}
	}
}

func docsHubTheme(t *testing.T) *theme.Manager {
	t.Helper()
	source := os.Getenv("NEWPRESS_DOCSHUB_SOURCE")
	if source == "" {
		t.Fatal("set NEWPRESS_DOCSHUB_SOURCE via scripts/check_core.py")
	}
	root := t.TempDir()
	if err := extension.CopyTree(t.Context(), source, filepath.Join(root, "docshub")); err != nil {
		t.Fatal(err)
	}
	m := theme.NewManager(root, "docshub", theme.WithSettingsDir(t.TempDir()))
	if err := m.Load(); err != nil {
		t.Fatal(err)
	}
	return m
}

func docsHubContent(t *testing.T) *content.MemoryRepository {
	t.Helper()
	repository := content.NewMemoryRepository(false)
	titles := []string{"环境搭建与项目初始化", "组件化开发实践", "从零构建一个 Web 应用", "性能优化指南", "V4.0 版本更新说明", "高级状态管理指南", "插件开发指南", "性能调优实战"}
	excerpts := []string{"快速搭建开发环境，开启项目之旅", "掌握组件设计与开发技巧", "完整项目实战，学以致用", "提升应用性能的最佳实践", "新增多项功能，优化开发体验", "深入理解状态管理的最佳实践", "扩展系统能力，打造专属插件", "从原理到实践，全面提升应用性能"}
	slugs := []string{"getting-started", "components", "web-app", "performance", "release-notes", "state-management", "plugins", "performance-practice"}
	levels := []string{"入门", "进阶", "实战", "进阶", "更新", "进阶", "API", "实战"}
	var ids []int64
	for i, title := range titles {
		post, err := repository.Create(t.Context(), content.PostInput{Type: content.TypePost, Status: content.StatusPublish, Slug: slugs[i], Title: title, Excerpt: excerpts[i], AuthorName: "DocsHub 团队", Category: levels[i], CommentStatus: "open", PublishedAt: time.Now().Add(-time.Duration(2+i) * time.Hour), ContentSource: `<p>欢迎来到 DocsHub。在这篇教程中，我们会一步步搭建开发环境，了解项目结构，并完成第一个功能。</p><h2>开始之前</h2><p>准备好熟悉的代码编辑器与命令行工具。清晰的目录结构，能让后续的开发更加轻松。</p><blockquote><p>把复杂的事情拆成小步骤，每一步都值得认真对待。</p></blockquote><h2>搭建开发环境</h2><p>创建项目目录，安装依赖，然后启动本地开发服务。</p><pre><code>mkdir my-project
cd my-project
npm install
npm run dev</code></pre><h3>认识项目结构</h3><p>将页面、可复用组件和样式放在各自的目录中，保持职责清晰。</p><table><thead><tr><th>目录</th><th>用途</th></tr></thead><tbody><tr><td>src/pages</td><td>页面与路由</td></tr><tr><td>src/components</td><td>可复用组件</td></tr><tr><td>src/styles</td><td>样式与设计变量</td></tr></tbody></table><h2>创建第一个组件</h2><p>从一个简单的问候组件开始，通过参数传递内容，把界面拆分为可以独立理解的小块。</p><pre><code>export function greeting(name) {
  return ` + "`你好，${name}。欢迎开始开发之旅！`" + `;
}</code></pre><h2>接下来做什么</h2><ul><li>阅读组件化开发实践，理解组件边界。</li><li>跟随实战教程，构建完整的 Web 应用。</li><li>在社区分享遇到的问题与新的收获。</li></ul><p>你已经迈出了第一步。继续探索，把知识变成真正的作品。</p>`})
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, post.ID)
	}
	_, err := repository.SaveSeries(t.Context(), 0, content.Mutation{}, content.SeriesInput{Kind: content.SeriesKindSequence, Status: content.StatusPublish, Title: "开发者入门之旅", Slug: "developer-journey", Description: "从环境搭建到完整项目，循序渐进地掌握开发技能。", PostIDs: ids[:4]})
	if err != nil {
		t.Fatal(err)
	}
	_, err = repository.Create(t.Context(), content.PostInput{Type: content.TypePage, Status: content.StatusPublish, Slug: "about", Title: "关于 DocsHub", ContentSource: "<h2>让知识更易抵达</h2><p>清晰的文档，是优秀开发体验的开始。</p>"})
	if err != nil {
		t.Fatal(err)
	}
	return repository
}

func TestDocsHubPublicTheme(t *testing.T) {
	m := docsHubTheme(t)
	s := New(config.Config{SiteTitle: "DocsHub", PublicURL: "https://example.test", ActiveTheme: "docshub"}, docsHubContent(t), m, plugin.NewManager(t.TempDir()))
	h := s.routes()
	readPage := func(route string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", route, nil))
		return w
	}
	for _, route := range []string{"/", "/explore/", "/explore/?q=" + url.QueryEscape("组件"), "/explore/?q=not-a-document", "/posts/getting-started", "/pages/about", "/series/", "/series/developer-journey/", "/account/", "/admin/"} {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", route, nil)
		r.RemoteAddr = "127.0.0.1:1234"
		h.ServeHTTP(w, r)
		if w.Code != 200 || strings.Contains(w.Body.String(), "#ZgotmplZ") || strings.Contains(w.Body.String(), "<no value>") {
			t.Fatalf("%s status=%d body=%s", route, w.Code, w.Body.String())
		}
		if route == "/pages/about" && (!strings.Contains(w.Body.String(), "优秀开发体验") || strings.Contains(w.Body.String(), "comment-form")) {
			t.Fatal("page body or comment ownership incorrect")
		}
		if route == "/posts/getting-started" && (!strings.Contains(w.Body.String(), "data-article-body") || !strings.Contains(w.Body.String(), "/posts/components")) {
			t.Fatal("article body or reading series navigation missing")
		}
		if route == "/admin/" && strings.Contains(w.Body.String(), "/theme-assets/docshub/") {
			t.Fatal("theme assets leaked into core admin")
		}
		if route == "/explore/?q=not-a-document" && !strings.Contains(w.Body.String(), "暂时没有找到相关文档") {
			t.Fatal("search empty state missing")
		}
		if route == "/series/" && strings.Contains(w.Body.String(), `aria-label="系列分页"`) {
			t.Fatal("single-page series must not offer a nonexistent next page")
		}
	}
	for _, asset := range []string{"css/docshub.css", "js/docshub.js", "js/docshub-mode.js", "js/docshub-verification.js", "images/glass-ribbon.webp", "images/tutorial-workspace.webp", "images/community-mountains.webp", "images/preview.webp"} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", m.AssetURL(asset), nil))
		if w.Code != 200 || !strings.Contains(w.Header().Get("Cache-Control"), "immutable") {
			t.Fatalf("asset %s: %d", asset, w.Code)
		}
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/missing-document", nil))
	if w.Code != 404 || !strings.Contains(w.Body.String(), "文档不存在") {
		t.Fatal("404 template failed")
	}
	snapshot := s.siteSettings.Snapshot()
	settings := snapshot.Settings
	settings.Discussion.RequireLogin = false
	if _, fields, err := s.siteSettings.Save(snapshot.Revision, settings); err != nil || len(fields) > 0 {
		t.Fatal(fields, err)
	}
	article := serveCommentRequest(h, nil, "GET", "https://example.test/posts/getting-started", "", "", "198.51.100.8:4000")
	form := url.Values{"form_token": {extractCommentToken(t, article.Body.String())}, "parent_id": {"0"}, "website": {""}, "author_name": {"文档读者"}, "email": {"reader@example.test"}, "content": {"这篇入门教程清晰易懂。"}}
	created := servePublicCommentForm(h, "getting-started", form, "https://example.test", "198.51.100.8:4000")
	if created.Code != 303 {
		t.Fatalf("comment failed: %d %s", created.Code, created.Body.String())
	}
	pending, err := s.comments.ListAdmin(t.Context(), comment.Query{Status: comment.StatusPending, Limit: 10})
	if err != nil || pending.Total != 1 {
		t.Fatal("comment not persisted", pending, err)
	}
	// Empty installations keep real empty states and never invent tutorial data.
	empty := New(config.Config{SiteTitle: "DocsHub", PublicURL: "https://example.test", ActiveTheme: "docshub"}, content.NewMemoryRepository(false), m, plugin.NewManager(t.TempDir()))
	w = httptest.NewRecorder()
	empty.routes().ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	if w.Code != 200 || !strings.Contains(w.Body.String(), "教程发布后会显示在这里") {
		t.Fatal("empty installation failed")
	}
	// Search/category links remain usable without JavaScript and do not discard q.
	filtered := readPage("/explore/?q=" + url.QueryEscape("指南"))
	if !strings.Contains(filtered.Body.String(), `data-filter-category=`) || !strings.Contains(filtered.Body.String(), `name="sort"`) || !strings.Contains(filtered.Body.String(), `data-list-view="list"`) {
		t.Fatal("discovery controls missing")
	}
	for _, match := range regexp.MustCompile(`data-filter-category="[^"]+" href="([^"]+)"`).FindAllStringSubmatch(filtered.Body.String(), -1) {
		link, err := url.Parse(strings.ReplaceAll(match[1], "&amp;", "&"))
		if err != nil || link.Query().Get("q") != "指南" {
			t.Fatalf("category filter lost search: %s", match[1])
		}
	}
	for _, route := range []string{"/categories/" + url.PathEscape("进阶") + "/", "/authors/newpress-team/", "/series/?q=missing"} {
		if page := readPage(route); page.Code != 200 {
			t.Fatalf("%s returned %d", route, page.Code)
		}
	}
	// Turning off modules must remove their markup and the default dangling anchor.
	themeSettings := m.SettingsSnapshot()
	for _, key := range []string{"show_quick_links", "show_tutorials", "show_latest", "show_community", "show_learning_path"} {
		themeSettings.Values[key] = false
	}
	if _, fields, err := m.SaveSettings(themeSettings.Values); err != nil || len(fields) > 0 {
		t.Fatal(fields, err)
	}
	withoutModules := readPage("/").Body.String()
	for _, marker := range []string{`class="quick-links"`, `id="tutorials"`, `class="home-bottom`, `class="community-card"`, `class="learning-path"`, `href="/#tutorials"`} {
		if strings.Contains(withoutModules, marker) {
			t.Fatalf("disabled module left %s", marker)
		}
	}
	// Closed and maintenance states must expose only valid next actions.
	latest := s.siteSettings.Snapshot()
	latest.Settings.Operations.MaintenanceEnabled = true
	latest.Settings.Operations.MaintenanceMessage = "文档维护测试"
	if _, fields, err := s.siteSettings.Save(latest.Revision, latest.Settings); err != nil || len(fields) > 0 {
		t.Fatal(fields, err)
	}
	maintenance := readPage("/")
	if maintenance.Code != 503 || !strings.Contains(maintenance.Body.String(), "文档维护测试") || strings.Contains(maintenance.Body.String(), `id="error-query"`) {
		t.Fatal("maintenance page has incorrect state")
	}
}

// Temporary runtime and content for visual checks; production files are never opened.
func TestDocsHubInteractiveUI(t *testing.T) {
	if os.Getenv("NEWPRESS_DOCSHUB_UI") != "1" {
		t.Skip("opt-in browser fixture")
	}
	m := docsHubTheme(t)
	snapshot := m.SettingsSnapshot()
	snapshot.Values["start_link"] = "/posts/getting-started"
	snapshot.Values["advanced_link"] = "/posts/components"
	snapshot.Values["practice_link"] = "/posts/web-app"
	snapshot.Values["about_link"] = "/pages/about"
	if _, fields, err := m.SaveSettings(snapshot.Values); err != nil || len(fields) > 0 {
		t.Fatal(fields, err)
	}
	// Optional browser administration uses an isolated in-memory identity only.
	var identity *auth.Manager
	if sessionFile := os.Getenv("NEWPRESS_DOCSHUB_UI_SESSION"); sessionFile != "" {
		password := "DocsHub isolated settings UI fixture"
		hash, err := auth.HashPassword(password)
		if err != nil {
			t.Fatal(err)
		}
		identity, err = auth.New(auth.Config{Username: "docshub-preview", PasswordHash: hash, AbsoluteTTL: time.Hour, IdleTTL: time.Hour})
		if err != nil {
			t.Fatal(err)
		}
		token, principal, err := identity.Authenticate("docshub-ui", "docshub-preview", password)
		if err != nil {
			t.Fatal(err)
		}
		payload, err := json.Marshal(map[string]string{"name": auth.CookieName, "value": token, "csrf": principal.CSRFToken})
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(sessionFile, payload, 0600); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { os.Remove(sessionFile) })
	}
	s := New(config.Config{SiteTitle: "DocsHub", SiteTagline: "让知识更易抵达", PublicURL: "http://127.0.0.1:19107", ActiveTheme: "docshub", AdminSessionTTL: time.Hour}, docsHubContent(t), m, plugin.NewManager(t.TempDir()), identity)
	siteSettings := s.siteSettings.Snapshot()
	siteSettings.Settings.Discussion.RequireLogin = false
	siteSettings.Settings.Reading.ExplorePosts = 10
	if _, fields, err := s.siteSettings.Save(siteSettings.Revision, siteSettings.Settings); err != nil || len(fields) > 0 {
		t.Fatal(fields, err)
	}
	if _, err := s.comments.Create(t.Context(), comment.CreateInput{PostID: 1, AuthorName: "文档读者", AuthorEmail: "reader@example.test", Status: comment.StatusApproved, ContentSource: "环境搭建完成了，期待更多项目示例。"}); err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:19107")
	if err != nil {
		t.Fatal(err)
	}
	server := &http.Server{Handler: s.routes(), ReadHeaderTimeout: 5 * time.Second}
	go server.Serve(listener)
	defer server.Close()
	t.Log("DocsHub isolated preview: http://127.0.0.1:19107/")
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-t.Context().Done():
			return
		case <-ticker.C:
			if stop := os.Getenv("NEWPRESS_DOCSHUB_UI_STOP"); stop != "" {
				if _, err := os.Stat(stop); err == nil {
					return
				}
			}
		}
	}
}
