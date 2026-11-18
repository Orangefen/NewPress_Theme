package httpserver

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	"newpress.local/newpress/internal/presentation"
	"newpress.local/newpress/internal/theme"
)

func nexoraInnerRender(t *testing.T, manager *theme.Manager, name string, page presentation.Page) string {
	t.Helper()
	var output bytes.Buffer
	if err := manager.Render(&output, name, page); err != nil {
		t.Fatal(err)
	}
	body := output.String()
	if strings.Contains(body, "#ZgotmplZ") || strings.Contains(body, "<no value>") {
		t.Fatal("inner-page option produced an invalid template value")
	}
	return body
}

func nexoraInnerArticle() presentation.Page {
	p := presentation.Fixtures()["components"]
	p.Post.CoverURL = "/uploads/article-cover.webp"
	p.Post.Tags = []presentation.Term{{Name: "研究", URL: "/tags/research/"}}
	p.Post.UpdatedAt = p.Post.PublishedAt.Add(24 * time.Hour)
	p.Post.ContentHTML = "<h2>文章内容</h2><p>始终可阅读的正文。</p><h3>内容细节</h3>"
	return p
}

func TestNexoraArticleOptionsPreserveContentAndModuleBoundaries(t *testing.T) {
	m := nexoraFixtureTheme(t)
	controls := map[string]string{
		"show_breadcrumbs":     `class="breadcrumbs"`,
		"article_cover":        `class="article-cover"`,
		"article_summary":      `class="article-lead"`,
		"article_meta":         `class="article-meta"`,
		"article_tags":         `class="article-tags"`,
		"article_toc":          `data-reading-toc`,
		"article_reading_time": `data-reading-time`,
		"article_copy_link":    `data-copy-link`,
		"article_series":       `class="reading-series"`,
	}
	baseline := nexoraInnerRender(t, m, "single", nexoraInnerArticle())
	for key, marker := range controls {
		if !strings.Contains(baseline, marker) {
			t.Fatalf("unset option %s must preserve its default component", key)
		}
		t.Run(key, func(t *testing.T) {
			p := nexoraInnerArticle()
			p.Settings[key] = false
			body := nexoraInnerRender(t, m, "single", p)
			if strings.Contains(body, marker) || !strings.Contains(body, "始终可阅读的正文") || !strings.Contains(body, `data-comment-form`) {
				t.Fatalf("disabling %s left its component visible or removed content/comments", key)
			}
			if key == "article_meta" && (strings.Contains(body, "data-reading-time") || strings.Contains(body, `href="#comments"`)) {
				t.Fatal("hidden metadata retained child controls")
			}
		})
	}
	for _, preview := range []bool{false, true} {
		p := nexoraInnerArticle()
		p.IsPreview = preview
		p.Modules["series"] = false
		p.Modules["comments"] = false
		p.Settings["article_series"], p.Settings["article_copy_link"] = true, true
		body := nexoraInnerRender(t, m, "single", p)
		if strings.Contains(body, `class="reading-series"`) || strings.Contains(body, `data-comment-form`) || (preview && strings.Contains(body, `data-copy-link`)) {
			t.Fatal("theme options bypassed disabled modules or draft preview")
		}
	}
}

func TestNexoraArticlePresentationOptions(t *testing.T) {
	m := nexoraFixtureTheme(t)
	p := nexoraInnerArticle()
	for _, width := range []string{"narrow", "standard", "wide"} {
		p.Settings["article_width"], p.Settings["article_title_align"] = width, "left"
		p.Settings["article_toc_open"], p.Settings["article_updated"] = false, true
		body := nexoraInnerRender(t, m, "single", p)
		if !strings.Contains(body, `article-align-left`) || !strings.Contains(body, `data-toc-open="false"`) || !strings.Contains(body, `class="article-updated"`) {
			t.Fatal("article alignment, TOC preference or updated date was ignored")
		}
		if width != "standard" && !strings.Contains(body, "article-width-"+width) {
			t.Fatalf("article width %s was ignored", width)
		}
	}
	p.Post.UpdatedAt = p.Post.PublishedAt
	if strings.Contains(nexoraInnerRender(t, m, "single", p), `class="article-updated"`) {
		t.Fatal("an unchanged article is incorrectly labeled as updated")
	}
	p.Kind = "page"
	body := nexoraInnerRender(t, m, "single", p)
	if strings.Contains(body, `class="article-meta"`) || strings.Contains(body, `data-comment-form`) || !strings.Contains(body, "始终可阅读的正文") {
		t.Fatal("article options changed standalone-page metadata or comment rules")
	}
}

func TestNexoraCardOptionsAcrossHomeAndArchives(t *testing.T) {
	m := nexoraFixtureTheme(t)
	cardPattern := regexp.MustCompile(`(?s)<article class="news-card(?: [^"]*)?">.*?</article>`)
	for mask := 0; mask < 16; mask++ {
		for _, templateName := range []string{"explore", "home"} {
			t.Run(fmt.Sprintf("%s/%04b", templateName, mask), func(t *testing.T) {
				p := presentation.Fixtures()["article"]
				p.Kind = templateName
				p.Settings = m.SettingsSnapshot().Values
				p.Settings["show_news"] = true
				p.Post.Category, p.Post.CoverURL = "卡片分类", "/uploads/card.webp"
				p.Posts = []presentation.Post{p.Post}
				p.Home.Recent = p.Posts
				settings := []string{"card_image", "card_excerpt", "card_date", "card_category"}
				for bit, key := range settings {
					p.Settings[key] = mask&(1<<bit) != 0
				}
				card := cardPattern.FindString(nexoraInnerRender(t, m, templateName, p))
				if card == "" || !strings.Contains(card, `href="/posts/hello"`) || !strings.Contains(card, "Hello &lt;world&gt;") {
					t.Fatal("card controls removed the article or its primary link")
				}
				for bit, marker := range []string{`class="news-image"`, "示例摘要", "<time", "卡片分类"} {
					if strings.Contains(card, marker) != (mask&(1<<bit) != 0) {
						t.Fatalf("%s did not match the selected card option", settings[bit])
					}
				}
				if (mask&12 == 0) && strings.Contains(card, `class="news-meta"`) {
					t.Fatal("disabling date and category left an empty metadata row")
				}
			})
		}
	}
}

func TestNexoraListingAndCustomTextOptions(t *testing.T) {
	m := nexoraFixtureTheme(t)
	p := presentation.Fixtures()["article"]
	p.Kind = "explore"
	p.Settings = m.SettingsSnapshot().Values
	p.Settings["listing_title"], p.Settings["listing_description"] = "研究 & 观察", "来自团队的最新记录。"
	p.Settings["series_title"], p.Settings["series_description"] = "阅读专题", "按研究方向展开阅读。"
	p.Settings["show_breadcrumbs"] = false
	for _, columns := range []string{"2", "3", "4"} {
		p.Settings["listing_layout"], p.Settings["listing_columns"] = "grid", columns
		body := nexoraInnerRender(t, m, "explore", p)
		marker := map[string]string{"2": `class="news-grid news-columns-two"`, "3": `class="news-grid"`, "4": `class="news-grid news-columns-four"`}[columns]
		if !strings.Contains(body, marker) || !strings.Contains(body, "研究 &amp; 观察") || !strings.Contains(body, "来自团队的最新记录。") || strings.Contains(body, `class="breadcrumbs"`) {
			t.Fatalf("grid %s or listing text settings were ignored", columns)
		}
	}
	p.Settings["listing_layout"] = "list"
	if !strings.Contains(nexoraInnerRender(t, m, "explore", p), `class="news-grid news-list"`) {
		t.Fatal("list layout did not override grid column selection")
	}
	p.SearchQuery = "实际查询"
	if body := nexoraInnerRender(t, m, "explore", p); !strings.Contains(body, "搜索结果") || !strings.Contains(body, "实际查询") {
		t.Fatal("custom listing titles replaced the active search context")
	}
	p.Kind = "series"
	series := nexoraInnerRender(t, m, "series", p)
	if !strings.Contains(series, "阅读专题") || !strings.Contains(series, "按研究方向展开阅读。") || strings.Contains(series, `class="breadcrumbs"`) {
		t.Fatal("series directory settings were ignored")
	}
	p.Series.Current = presentation.Series{Title: "具体专题", Description: "专题的实际说明。", Label: "专题", URL: "/series/topic/"}
	if body := nexoraInnerRender(t, m, "series", p); !strings.Contains(body, "具体专题") || !strings.Contains(body, "专题的实际说明。") {
		t.Fatal("directory settings replaced the current series content")
	}
	p.Modules["comments"] = true
	p.Kind = "single"
	p.Settings["comments_title"], p.Settings["comments_intro"] = "读者 <交流>", "欢迎分享<script>alert(1)</script>"
	p.Comments.PrivacyNotice, p.Comments.ModerationNotice = "邮箱不会公开。", "评论需审核。"
	article := nexoraInnerRender(t, m, "single", p)
	if !strings.Contains(article, "读者 &lt;交流&gt;") || !strings.Contains(article, "欢迎分享&lt;script&gt;") || strings.Contains(article, "<script>alert(1)</script>") || !strings.Contains(article, "邮箱不会公开。") || !strings.Contains(article, "评论需审核。") {
		t.Fatal("comment introduction was not escaped or replaced the site's posting policies")
	}
}
