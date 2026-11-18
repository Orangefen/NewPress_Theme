package httpserver

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"

	"golang.org/x/net/html"
	"newpress.local/newpress/internal/presentation"
	"newpress.local/newpress/internal/theme"
)

// Render only validated settings, including schema defaults and integer
// normalization, just as an appearance preview does in the core.
func docsHubOptionsRender(t *testing.T, manager *theme.Manager, name string, page presentation.Page, changes map[string]any) *html.Node {
	t.Helper()
	values := manager.SettingsSnapshot().Values
	for key, value := range changes {
		values[key] = value
	}
	preview, _, classes, fields := manager.PreviewSettings(values)
	if len(fields) != 0 {
		t.Fatalf("settings preview rejected: %v", fields)
	}
	page.Settings, page.ThemeClass = preview.Values, classes
	var output bytes.Buffer
	if err := manager.Render(&output, name, page); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(output.String(), "#ZgotmplZ") || strings.Contains(output.String(), "<no value>") {
		t.Fatal("setting produced an invalid template value")
	}
	doc, err := html.Parse(&output)
	if err != nil {
		t.Fatal(err)
	}
	return doc
}

func docsHubOptionAttr(node *html.Node, key string) string {
	if node != nil {
		for _, attribute := range node.Attr {
			if attribute.Key == key {
				return attribute.Val
			}
		}
	}
	return ""
}

func docsHubOptionNodes(node *html.Node, match func(*html.Node) bool) []*html.Node {
	var found []*html.Node
	if node == nil {
		return found
	}
	if node.Type == html.ElementNode && match(node) {
		found = append(found, node)
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		found = append(found, docsHubOptionNodes(child, match)...)
	}
	return found
}

func docsHubOptionClass(node *html.Node, class string) []*html.Node {
	return docsHubOptionNodes(node, func(n *html.Node) bool {
		for _, value := range strings.Fields(docsHubOptionAttr(n, "class")) {
			if value == class {
				return true
			}
		}
		return false
	})
}

func docsHubOptionAttribute(node *html.Node, key, value string) []*html.Node {
	return docsHubOptionNodes(node, func(n *html.Node) bool {
		for _, attribute := range n.Attr {
			if attribute.Key == key && attribute.Val == value {
				return true
			}
		}
		return false
	})
}

func docsHubOptionText(node *html.Node) string {
	if node == nil {
		return ""
	}
	if node.Type == html.TextNode {
		return node.Data
	}
	var result strings.Builder
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		result.WriteString(docsHubOptionText(child))
	}
	return result.String()
}

func docsHubOptionCount(t *testing.T, doc *html.Node, class string, expected int) {
	t.Helper()
	if actual := len(docsHubOptionClass(doc, class)); actual != expected {
		t.Fatalf(".%s: got %d nodes, want %d", class, actual, expected)
	}
}

func docsHubOptionOne(t *testing.T, nodes []*html.Node) *html.Node {
	t.Helper()
	if len(nodes) != 1 {
		t.Fatalf("expected exactly one node, got %d", len(nodes))
	}
	return nodes[0]
}

func docsHubOptionHome(count int) presentation.Page {
	page := presentation.Fixtures()["empty"]
	page.Modules["series"] = true
	for i := 0; i < count; i++ {
		page.Posts = append(page.Posts, presentation.Post{ID: int64(i + 1), Slug: fmt.Sprintf("document-%d", i), Title: fmt.Sprintf("真实文档 %d", i), URL: fmt.Sprintf("/posts/document-%d", i), Excerpt: "文档操作与步骤说明。", Category: "开发", PublishedAt: time.Date(2026, 11, 16-i, 1, 0, 0, 0, time.UTC)})
	}
	return page
}

func TestDocsHubHomeSettingsAndModuleCombinations(t *testing.T) {
	manager := docsHubTheme(t)
	page := docsHubOptionHome(10)
	baseline := docsHubOptionsRender(t, manager, "home", page, nil)
	docsHubOptionCount(t, baseline, "tutorial-card", 4)
	docsHubOptionCount(t, baseline, "document-row", 4)
	docsHubOptionCount(t, baseline, "hero-art", 1)
	docsHubOptionCount(t, baseline, "hero-eyebrow", 0)
	keys := []string{"show_quick_links", "show_tutorials", "show_latest", "show_community", "show_learning_path"}
	classes := []string{"quick-links", "tutorials", "latest", "community-card", "learning-path"}
	for mask := 0; mask < 32; mask++ {
		t.Run(fmt.Sprintf("modules-%05b", mask), func(t *testing.T) {
			values := map[string]any{}
			for bit, key := range keys {
				values[key] = mask&(1<<bit) != 0
			}
			doc := docsHubOptionsRender(t, manager, "home", page, values)
			for bit, class := range classes {
				expected := 0
				if mask&(1<<bit) != 0 {
					expected = 1
				}
				docsHubOptionCount(t, doc, class, expected)
			}
			if len(docsHubOptionAttribute(doc, "href", "/#tutorials")) > 0 && mask&2 == 0 {
				t.Fatal("disabled tutorials left a dangling hero anchor")
			}
			bottom := docsHubOptionClass(doc, "home-bottom")
			if mask&28 == 0 && len(bottom) != 0 {
				t.Fatal("empty lower homepage layout remains")
			}
			if mask&28 != 0 {
				node := docsHubOptionOne(t, bottom)
				wantSingle := !(mask&4 != 0 && mask&24 != 0)
				if (len(docsHubOptionClass(node, "home-bottom-single")) == 1) != wantSingle {
					t.Fatal("lower homepage columns do not match enabled modules")
				}
			}
			if mask&30 == 0 {
				docsHubOptionCount(t, doc, "home-content", 0)
			}
		})
	}
	for _, count := range []int{1, 3, 8} {
		t.Run(fmt.Sprintf("article-count-%d", count), func(t *testing.T) {
			doc := docsHubOptionsRender(t, manager, "home", page, map[string]any{"tutorials_count": count, "latest_count": count})
			docsHubOptionCount(t, doc, "tutorial-card", count)
			docsHubOptionCount(t, doc, "document-row", count)
		})
	}
	page.Featured = page.Posts[:2]
	doc := docsHubOptionsRender(t, manager, "home", page, map[string]any{"tutorials_count": 8, "latest_count": 8})
	docsHubOptionCount(t, doc, "tutorial-card", 2)
	docsHubOptionCount(t, doc, "tutorial-grid-count-2", 1)
	docsHubOptionCount(t, doc, "document-row", 8)
	empty := docsHubOptionsRender(t, manager, "home", docsHubOptionHome(0), nil)
	docsHubOptionCount(t, empty, "tutorial-card", 0)
	if !strings.Contains(docsHubOptionText(empty), "教程发布后会显示在这里") {
		t.Fatal("empty installation no longer explains its empty tutorial section")
	}
}

func TestDocsHubHomeCustomCopyAndOptionalContent(t *testing.T) {
	manager := docsHubTheme(t)
	values := map[string]any{
		"hero_eyebrow": "产品手册", "hero_title": "部署 <script>文档</script>", "hero_primary_label": "开始配置", "hero_secondary_label": "查看示例",
		"tutorials_title": "精选课程", "tutorials_description": "按实践顺序学习。", "tutorials_more_label": "更多课程", "tutorials_more_link": "/courses/",
		"latest_title": "近期更新", "latest_description": "团队文档动态", "latest_more_link": "/updates/",
		"community_description": "向团队提交问题", "community_button_label": "参与讨论", "learning_path_title": "实践路线",
		"quick_start_title": "初始化项目", "quick_start_description": "创建你的第一个项目", "learning_start_title": "准备环境", "learning_start_description": "先完成工具安装",
	}
	doc := docsHubOptionsRender(t, manager, "home", docsHubOptionHome(3), values)
	for _, expected := range []string{"部署 <script>文档</script>", "产品手册", "开始配置", "精选课程", "按实践顺序学习。", "近期更新", "团队文档动态", "向团队提交问题", "参与讨论", "实践路线", "初始化项目", "准备环境"} {
		if !strings.Contains(docsHubOptionText(doc), expected) {
			t.Fatalf("custom homepage copy was ignored: %s", expected)
		}
	}
	if len(docsHubOptionNodes(doc, func(n *html.Node) bool { return n.Data == "script" && docsHubOptionText(n) == "文档" })) != 0 {
		t.Fatal("custom homepage title injected an element")
	}
	docsHubOptionOne(t, docsHubOptionAttribute(doc, "href", "/courses/"))
	docsHubOptionOne(t, docsHubOptionAttribute(doc, "href", "/updates/"))
	optional := map[string]any{
		"hero_title": "", "hero_title_second": "", "hero_description": "", "hero_description_second": "", "hero_eyebrow": "",
		"show_hero_artwork": false, "show_hero_secondary": false, "tutorials_description": "", "latest_description": "",
		"tutorials_more_label": "", "latest_more_link": "", "community_description": "", "community_link": "", "show_community_artwork": false,
		"show_latest_date": false, "show_latest_excerpt": false, "show_learning_series_link": false,
		"quick_start_description": "", "quick_advanced_description": "", "quick_api_description": "", "quick_tools_description": "",
		"learning_start_description": "", "learning_advanced_description": "", "learning_practice_description": "",
	}
	doc = docsHubOptionsRender(t, manager, "home", docsHubOptionHome(3), optional)
	for _, class := range []string{"hero-art", "hero-eyebrow", "hero-description", "section-description"} {
		docsHubOptionCount(t, doc, class, 0)
	}
	hero := docsHubOptionOne(t, docsHubOptionClass(doc, "hero"))
	docsHubOptionCount(t, hero, "button-secondary", 0)
	if text := docsHubOptionText(docsHubOptionOne(t, docsHubOptionAttribute(hero, "id", "hero-title"))); text != "示例站点" {
		t.Fatalf("blank primary heading fallback = %q", text)
	}
	latest := docsHubOptionOne(t, docsHubOptionClass(doc, "latest"))
	if len(docsHubOptionNodes(latest, func(n *html.Node) bool { return n.Data == "time" })) != 0 || strings.Contains(docsHubOptionText(latest), "文档操作与步骤说明。") {
		t.Fatal("disabled latest metadata remains")
	}
	community := docsHubOptionOne(t, docsHubOptionClass(doc, "community-card"))
	if len(docsHubOptionNodes(community, func(n *html.Node) bool { return n.Data == "a" || n.Data == "img" || n.Data == "p" })) != 0 {
		t.Fatal("blank community details left empty controls")
	}
	for _, class := range []string{"quick-links", "learning-path"} {
		node := docsHubOptionOne(t, docsHubOptionClass(doc, class))
		if len(docsHubOptionNodes(node, func(n *html.Node) bool { return n.Data == "small" })) != 0 {
			t.Fatal("blank optional descriptions left markup")
		}
	}
	calendar := docsHubOptionsRender(t, manager, "home", docsHubOptionHome(2), map[string]any{"relative_dates": false})
	latest = docsHubOptionOne(t, docsHubOptionClass(calendar, "latest"))
	if len(docsHubOptionAttribute(latest, "data-relative-time", "")) != 0 || len(docsHubOptionNodes(latest, func(n *html.Node) bool { return n.Data == "time" })) != 2 {
		t.Fatal("absolute dates did not preserve actual publication dates")
	}
}

func TestDocsHubShellOptionsRespectCoreNavigationAndFooter(t *testing.T) {
	manager := docsHubTheme(t)
	page := docsHubOptionHome(0)
	page.Site.PublicAPIEnabled = true
	baseline := docsHubOptionsRender(t, manager, "home", page, nil)
	footer := docsHubOptionOne(t, docsHubOptionAttribute(baseline, "id", "site-footer"))
	docsHubOptionOne(t, docsHubOptionAttribute(footer, "href", "/feed.xml"))
	docsHubOptionOne(t, docsHubOptionAttribute(footer, "href", "/"))
	values := map[string]any{
		"default_color_mode": "dark", "allow_color_switch": false, "sticky_header": false, "reduce_motion": true,
		"show_search": false, "show_help": false, "show_language": false, "show_account": false, "show_utility_navigation": false,
		"show_series_navigation": false, "show_back_to_top": false, "announcement_text": "维护 <b>通知</b>", "announcement_link": "/status/", "announcement_link_label": "",
	}
	doc := docsHubOptionsRender(t, manager, "home", page, values)
	root := docsHubOptionOne(t, docsHubOptionNodes(doc, func(n *html.Node) bool { return n.Data == "html" }))
	for key, expected := range map[string]string{"data-mode": "dark", "data-default-mode": "dark", "data-allow-color-switch": "false", "data-sticky-header": "false", "data-reduce-motion": "true"} {
		if docsHubOptionAttr(root, key) != expected {
			t.Fatalf("%s did not receive the configured preference", key)
		}
	}
	for _, class := range []string{"search-form", "topbar-actions", "secondary-nav", "appearance-controls", "back-to-top"} {
		docsHubOptionCount(t, doc, class, 0)
	}
	docsHubOptionCount(t, doc, "topbar-navigation-only", 1)
	docsHubOptionOne(t, docsHubOptionAttribute(doc, "data-menu-toggle", ""))
	announcement := docsHubOptionOne(t, docsHubOptionClass(doc, "site-announcement"))
	if !strings.Contains(docsHubOptionText(announcement), "维护 <b>通知</b>") || len(docsHubOptionNodes(announcement, func(n *html.Node) bool { return n.Data == "b" })) != 0 {
		t.Fatal("announcement was not escaped")
	}
	if !strings.Contains(docsHubOptionText(docsHubOptionOne(t, docsHubOptionAttribute(announcement, "href", "/status/"))), "查看详情") {
		t.Fatal("empty announcement link label has no fallback")
	}
	for _, group := range []string{"primary", "utility", "footer"} {
		page.Navigation[group] = presentation.Menu{Configured: true}
	}
	doc = docsHubOptionsRender(t, manager, "home", page, map[string]any{"show_utility_navigation": true, "show_footer_navigation": true})
	for _, class := range []string{"primary-nav", "secondary-nav", "nav-label"} {
		docsHubOptionCount(t, doc, class, 0)
	}
	footer = docsHubOptionOne(t, docsHubOptionAttribute(doc, "id", "site-footer"))
	if len(docsHubOptionNodes(footer, func(n *html.Node) bool { return n.Data == "nav" })) != 0 {
		t.Fatal("explicitly empty footer menu resurrected fallback links")
	}
	page.Navigation["footer"] = presentation.Menu{}
	page.Integrations.FooterHideCredit = true
	page.Integrations.FooterCopyright = "核心配置的版权"
	page.Integrations.FooterHTML = "<p>业务页脚</p>"
	page.Integrations.ICPNumber = "示例备案号"
	doc = docsHubOptionsRender(t, manager, "home", page, map[string]any{"show_theme_credit": true, "footer_description": "不应替换版权"})
	footer = docsHubOptionOne(t, docsHubOptionAttribute(doc, "id", "site-footer"))
	if len(docsHubOptionAttribute(footer, "href", "/")) != 0 || !strings.Contains(docsHubOptionText(footer), "核心配置的版权") || strings.Contains(docsHubOptionText(footer), "不应替换版权") {
		t.Fatal("theme footer preferences overrode core credit or copyright policy")
	}
	doc = docsHubOptionsRender(t, manager, "home", page, map[string]any{"show_footer_navigation": false})
	footer = docsHubOptionOne(t, docsHubOptionAttribute(doc, "id", "site-footer"))
	if len(docsHubOptionNodes(footer, func(n *html.Node) bool { return n.Data == "nav" })) != 0 || !strings.Contains(docsHubOptionText(footer), "业务页脚") || !strings.Contains(docsHubOptionText(footer), "示例备案号") {
		t.Fatal("hiding footer navigation removed core integration content")
	}
}

func docsHubOptionArticle() presentation.Page {
	page := presentation.Fixtures()["components"]
	page.Post.ContentHTML = "<h2>文章正文</h2><p>始终可阅读的内容。</p><h3>操作步骤</h3><pre><code>run example</code></pre>"
	page.Post.UpdatedAt = page.Post.PublishedAt.Add(time.Hour)
	page.Post.Tags = []presentation.Term{{Name: "实践", URL: "/tags/practice/"}}
	page.Comments.PrivacyNotice, page.Comments.ModerationNotice = "邮箱不会公开。", "评论需经审核。"
	return page
}

func TestDocsHubReadingOptionsAndModuleBoundaries(t *testing.T) {
	manager := docsHubTheme(t)
	page := docsHubOptionArticle()
	baseline := docsHubOptionsRender(t, manager, "single", page, nil)
	for _, class := range []string{"article-aside", "mobile-toc", "table-of-contents", "series-reading", "article-tags", "article-meta", "comments-section"} {
		docsHubOptionCount(t, baseline, class, 1)
	}
	docsHubOptionCount(t, baseline, "related-reading", 0)
	withRelated := docsHubOptionsRender(t, manager, "single", page, map[string]any{"show_series_related": true})
	docsHubOptionCount(t, withRelated, "related-reading", 1)
	for _, depth := range []string{"h2", "h3", "h4"} {
		doc := docsHubOptionsRender(t, manager, "single", page, map[string]any{"toc_depth": depth})
		body := docsHubOptionOne(t, docsHubOptionAttribute(doc, "data-article-body", ""))
		if docsHubOptionAttr(body, "data-toc-depth") != depth {
			t.Fatalf("TOC depth %q did not reach reading behavior", depth)
		}
	}
	changes := map[string]any{
		"show_toc": false, "show_reading_help": false, "show_reading_back_to_top": false, "show_reading_progress": false,
		"show_reading_series": false, "show_series_related": false, "show_article_author": false, "show_article_date": false,
		"show_article_excerpt": false, "show_article_tags": false, "show_reading_time": false, "show_copy_link": false, "show_code_copy": false,
		"comments_title": "文档讨论", "comments_intro": "补充你的实际体验。", "comments_placeholder": "描述问题与复现步骤",
	}
	doc := docsHubOptionsRender(t, manager, "single", page, changes)
	for _, class := range []string{"article-aside", "mobile-toc", "table-of-contents", "series-reading", "related-reading", "article-tags", "article-meta", "article-lead", "reading-progress"} {
		docsHubOptionCount(t, doc, class, 0)
	}
	docsHubOptionCount(t, doc, "reading-layout-no-aside", 1)
	body := docsHubOptionOne(t, docsHubOptionAttribute(doc, "data-article-body", ""))
	if docsHubOptionAttr(body, "data-code-copy") != "false" || !strings.Contains(docsHubOptionText(body), "始终可阅读的内容。") {
		t.Fatal("reading controls removed article content or retained code copy")
	}
	comments := docsHubOptionOne(t, docsHubOptionClass(doc, "comments-section"))
	for _, expected := range []string{"文档讨论", "补充你的实际体验。", "邮箱不会公开。", "评论需经审核。"} {
		if !strings.Contains(docsHubOptionText(comments), expected) {
			t.Fatalf("comment option or core policy missing: %s", expected)
		}
	}
	docsHubOptionOne(t, docsHubOptionAttribute(comments, "placeholder", "描述问题与复现步骤"))
	docsHubOptionOne(t, docsHubOptionAttribute(comments, "data-comment-form", ""))
	page.Modules["series"], page.Modules["comments"] = false, false
	doc = docsHubOptionsRender(t, manager, "single", page, map[string]any{"show_reading_series": true, "show_series_related": true})
	for _, class := range []string{"series-reading", "related-reading", "reading-series-summary", "comments-section"} {
		docsHubOptionCount(t, doc, class, 0)
	}
	page.Modules["comments"] = true
	page.IsPreview = true
	doc = docsHubOptionsRender(t, manager, "single", page, nil)
	docsHubOptionCount(t, doc, "comments-section", 0)
	docsHubOptionCount(t, doc, "preview-notice", 1)
	page.IsPreview, page.Kind = false, "page"
	doc = docsHubOptionsRender(t, manager, "page", page, nil)
	docsHubOptionCount(t, doc, "comments-section", 0)
	if !strings.Contains(docsHubOptionText(doc), "始终可阅读的内容。") {
		t.Fatal("standalone page lost its main body")
	}
}

func TestDocsHubListingCardAndSeriesSettings(t *testing.T) {
	manager := docsHubTheme(t)
	page := docsHubOptionHome(2)
	page.Kind = "explore"
	page.Posts[0].CoverURL = "/uploads/real-cover.webp"
	page.Pagination = presentation.Pagination{Total: 2, Page: 1, Pages: 1}
	page.Categories = []presentation.Term{{Name: "开发", Slug: "development", URL: "/categories/development/", Count: 2}}
	for _, templateName := range []string{"home", "explore", "archive", "author"} {
		for _, cover := range []string{"all", "real", "none"} {
			t.Run(templateName+"-"+cover, func(t *testing.T) {
				page.Kind = templateName
				doc := docsHubOptionsRender(t, manager, templateName, page, map[string]any{"card_cover": cover, "show_card_category": false, "show_card_excerpt": false, "card_meta": "none"})
				cards := docsHubOptionClass(doc, "tutorial-card")
				if len(cards) != 2 {
					t.Fatalf("card options removed results: %d", len(cards))
				}
				images := 0
				for _, card := range cards {
					images += len(docsHubOptionClass(card, "tutorial-image"))
					for _, class := range []string{"category-badge", "tutorial-meta"} {
						docsHubOptionCount(t, card, class, 0)
					}
					if strings.Contains(docsHubOptionText(card), "文档操作与步骤说明。") || len(docsHubOptionNodes(card, func(n *html.Node) bool { return n.Data == "h3" })) != 1 {
						t.Fatal("card display settings lost the title or kept the excerpt")
					}
				}
				if images != map[string]int{"all": 2, "real": 1, "none": 0}[cover] {
					t.Fatalf("%s cover mode rendered %d images", cover, images)
				}
			})
		}
	}
	page.Kind = "explore"
	page.SearchQuery, page.Category, page.CategorySlug = "操作指南", "开发", "development"
	doc := docsHubOptionsRender(t, manager, "explore", page, map[string]any{"list_default_view": "list", "show_category_filters": false, "show_sort_control": false, "show_view_switch": false, "listing_title": "自定义文档入口"})
	for _, class := range []string{"category-filters", "sort-form", "view-switch", "listing-controls"} {
		docsHubOptionCount(t, doc, class, 0)
	}
	docsHubOptionOne(t, docsHubOptionAttribute(doc, "data-default-view", "list"))
	docsHubOptionOne(t, docsHubOptionAttribute(doc, "data-clear-filter", "q"))
	docsHubOptionOne(t, docsHubOptionAttribute(doc, "data-clear-filter", "category"))
	intro := docsHubOptionOne(t, docsHubOptionClass(doc, "page-intro"))
	if !strings.Contains(docsHubOptionText(intro), "搜索「操作指南」") || strings.Contains(docsHubOptionText(intro), "自定义文档入口") {
		t.Fatal("listing title settings replaced actual search context")
	}
	seriesPage := presentation.Fixtures()["components"]
	seriesPage.Kind = "series"
	seriesPage.Series.Current = presentation.Series{}
	doc = docsHubOptionsRender(t, manager, "series", seriesPage, map[string]any{"series_title": "学习专题", "series_description": "从基础到实战。", "show_series_search": false})
	docsHubOptionCount(t, doc, "series-search", 0)
	intro = docsHubOptionOne(t, docsHubOptionClass(doc, "page-intro"))
	if !strings.Contains(docsHubOptionText(intro), "学习专题") || !strings.Contains(docsHubOptionText(intro), "从基础到实战。") {
		t.Fatal("series directory did not use configured copy")
	}
	seriesPage.Series.Current = presentation.Series{Title: "真实学习系列", URL: "/series/real/", Count: 1, Description: "真实系列说明"}
	doc = docsHubOptionsRender(t, manager, "series", seriesPage, map[string]any{"series_title": "学习专题", "show_chapter_excerpt": false, "show_chapter_meta": false})
	chapters := docsHubOptionOne(t, docsHubOptionClass(doc, "chapter-list"))
	if len(docsHubOptionNodes(chapters, func(n *html.Node) bool { return n.Data == "p" || n.Data == "small" })) != 0 || !strings.Contains(docsHubOptionText(doc), "真实学习系列") {
		t.Fatal("chapter settings hid content or retained disabled details")
	}
}
