package httpserver

import (
	"net/url"
	"strings"
	"testing"

	"golang.org/x/net/html"
	"newpress.local/newpress/internal/presentation"
	"newpress.local/newpress/internal/theme"
)

func TestNexoraChromeOptions(t *testing.T) {
	m := nexoraFixtureTheme(t)
	render := func(values map[string]any) string {
		t.Helper()
		p := presentation.Fixtures()["empty"]
		p.Settings = m.SettingsSnapshot().Values
		for key, value := range values {
			p.Settings[key] = value
		}
		var out theme.RenderBuffer
		if err := m.Render(&out, "home", p); err != nil {
			t.Fatal(err)
		}
		if strings.Contains(out.String(), "#ZgotmplZ") || strings.Contains(out.String(), "<no value>") {
			t.Fatal("invalid rendered value")
		}
		return out.String()
	}
	t.Run("single-hero-and-images", func(t *testing.T) {
		out := render(map[string]any{"hero_mode": "single", "hero_image": "/custom-wide.webp", "hero_mobile_image": "/custom-phone.webp", "hero_image_alt": "我们的园区"})
		if strings.Count(out, " data-slide>") != 1 || strings.Contains(out, "data-hero-controls") || !strings.Contains(out, `srcset="/custom-phone.webp"`) || !strings.Contains(out, `alt="我们的园区"`) {
			t.Fatal("single hero or mobile image was not rendered correctly")
		}
		out = render(map[string]any{"second_image": "/second.webp", "third_image": "/third.webp", "hero_autoplay": true, "hero_interval": 9})
		for _, want := range []string{`src="/second.webp"`, `src="/third.webp"`, `data-autoplay="true"`, `data-interval="9"`, `data-slide-playback`} {
			if !strings.Contains(out, want) {
				t.Fatalf("missing %s", want)
			}
		}
		out = render(map[string]any{"about_image": "/about-fallback.webp", "innovation_image": "/innovation-fallback.webp"})
		if strings.Count(out, `src="/about-fallback.webp"`) != 2 || strings.Count(out, `src="/innovation-fallback.webp"`) != 2 {
			t.Fatal("existing slide image settings lost their fallback")
		}
	})
	t.Run("header-footer-options", func(t *testing.T) {
		out := render(map[string]any{"header_sticky": false, "show_header_search": false, "header_contact_label": "洽谈合作", "show_footer_top": false, "show_back_to_top": false, "footer_copyright": "自定义版权"})
		for _, absent := range []string{`data-search-trigger`, `class="footer-top"`, `class="back-to-top"`} {
			if strings.Contains(out, absent) {
				t.Fatalf("hidden control remains: %s", absent)
			}
		}
		for _, want := range []string{`data-sticky="false"`, `洽谈合作`, `自定义版权`} {
			if !strings.Contains(out, want) {
				t.Fatalf("missing %s", want)
			}
		}
		out = render(map[string]any{"show_footer_top": false, "show_back_to_top": true})
		if strings.Count(out, `class="back-to-top"`) != 1 {
			t.Fatal("footer-bottom back-to-top fallback missing")
		}
	})
	t.Run("contact-placement-and-escaping", func(t *testing.T) {
		for place, count := range map[string]int{"hidden": 0, "contact": 1, "footer": 1, "both": 2} {
			out := render(map[string]any{"contact_details_location": place, "contact_email": "team@example.test", "contact_phone": "+86 400 123 4567", "contact_address": "研发中心 <script>alert(1)</script>", "contact_hours": "周一至周五 09:00–18:00", "contact_map_url": "https://example.test/map", "social_1_label": "技术社区", "social_1_url": "https://example.test/community", "social_2_label": "未配置链接"})
			if strings.Count(out, `class="contact-details"`) != count {
				t.Fatalf("%s contact placement mismatch", place)
			}
			if strings.Contains(out, "未配置链接") || strings.Contains(out, "<script>alert(1)</script>") {
				t.Fatal("incomplete or unescaped contact details rendered")
			}
			if count > 0 {
				doc, err := html.Parse(strings.NewReader(out))
				if err != nil {
					t.Fatal(err)
				}
				links := map[string]bool{}
				var walk func(*html.Node)
				walk = func(node *html.Node) {
					for _, attr := range node.Attr {
						if attr.Key == "href" {
							value, _ := url.PathUnescape(attr.Val)
							links[value] = true
						}
					}
					for child := node.FirstChild; child != nil; child = child.NextSibling {
						walk(child)
					}
				}
				walk(doc)
				if !links["mailto:team@example.test"] || !links["tel:+86 400 123 4567"] {
					t.Fatal("contact action links missing")
				}
			}
		}
		out := render(map[string]any{"contact_details_location": "both", "social_1_label": "只有名称"})
		if strings.Contains(out, `class="contact-details"`) {
			t.Fatal("empty contact section rendered")
		}
	})
}
