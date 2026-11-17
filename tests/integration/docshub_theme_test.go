package httpserver

import (
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"newpress.local/newpress/internal/comment"
	"newpress.local/newpress/internal/config"
	"newpress.local/newpress/internal/content"
	"newpress.local/newpress/internal/extension"
	"newpress.local/newpress/internal/plugin"
	"newpress.local/newpress/internal/theme"
)

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
	if w.Code != 404 || !strings.Contains(w.Body.String(), "这篇文档还没被找到") {
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
	s := New(config.Config{SiteTitle: "DocsHub", SiteTagline: "让知识更易抵达", PublicURL: "http://127.0.0.1:19107", ActiveTheme: "docshub"}, docsHubContent(t), m, plugin.NewManager(t.TempDir()))
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
