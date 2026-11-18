package httpserver

import (
	"strings"
	"testing"

	"golang.org/x/net/html"
	"newpress.local/newpress/internal/presentation"
	"newpress.local/newpress/internal/theme"
)

// The full matrix catches CTAs whose destination was removed by a module switch.
func TestNexoraHomepageModuleCombinations(t *testing.T) {
	m := nexoraFixtureTheme(t)
	modules := []string{"hero", "about", "strengths", "innovation", "business", "news", "careers", "contact"}
	for mask := 0; mask < 1<<len(modules); mask++ {
		p := presentation.Fixtures()["empty"]
		p.Settings = m.SettingsSnapshot().Values
		for bit, name := range modules {
			p.Settings["show_"+name] = mask&(1<<bit) != 0
		}
		var output theme.RenderBuffer
		if err := m.Render(&output, "home", p); err != nil {
			t.Fatalf("mask %d: %v", mask, err)
		}
		home := output.String()
		if strings.Contains(home, "#ZgotmplZ") || strings.Contains(home, "<no value>") {
			t.Fatalf("mask %d: invalid rendered value", mask)
		}
		document, err := html.Parse(strings.NewReader(home))
		if err != nil {
			t.Fatal(err)
		}
		ids, links, sections, headings := map[string]bool{}, []string{}, map[string]bool{}, 0
		var visit func(*html.Node)
		visit = func(node *html.Node) {
			if node.Type == html.ElementNode {
				if node.Data == "h1" {
					headings++
				}
				for _, a := range node.Attr {
					if a.Key == "id" {
						ids[a.Val] = true
					}
					if a.Key == "data-home-section" {
						sections[a.Val] = true
					}
					if a.Key == "href" {
						links = append(links, a.Val)
					}
				}
			}
			for child := node.FirstChild; child != nil; child = child.NextSibling {
				visit(child)
			}
		}
		visit(document)
		if headings != 1 {
			t.Fatalf("mask %d: expected one H1, got %d", mask, headings)
		}
		for bit, name := range modules {
			if sections[name] != (mask&(1<<bit) != 0) {
				t.Fatalf("mask %d: module %s visibility mismatch", mask, name)
			}
		}
		for _, link := range links {
			anchor := ""
			if strings.HasPrefix(link, "/#") {
				anchor = strings.TrimPrefix(link, "/#")
			}
			if strings.HasPrefix(link, "#") {
				anchor = strings.TrimPrefix(link, "#")
			}
			if anchor != "" && !ids[anchor] {
				t.Fatalf("mask %d: missing anchor %s", mask, link)
			}
		}
	}
}

func TestNexoraHomepageContactFallbackAndOptionalButtons(t *testing.T) {
	m := nexoraFixtureTheme(t)
	cases := []struct {
		name         string
		feedback     bool
		action, want string
	}{
		{"account", false, "", "/account/"},
		{"feedback", true, "", "/feedback/"},
		{"custom", true, "/pages/contact", "/pages/contact"},
		{"self-link", true, "/#contact", "/feedback/"},
		{"hidden-link", false, "/#about", "/account/"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := presentation.Fixtures()["empty"]
			p.Settings = m.SettingsSnapshot().Values
			p.Settings["show_contact"], p.Settings["show_about"] = false, false
			p.Settings["contact_action_url"] = tc.action
			p.Settings["hero_link"], p.Settings["about_link"] = "", ""
			p.Modules["feedback"] = tc.feedback
			var output theme.RenderBuffer
			if err := m.Render(&output, "home", p); err != nil {
				t.Fatal(err)
			}
			home := output.String()
			if !strings.Contains(home, `href="`+tc.want+`"`) {
				t.Fatalf("missing contact destination %s", tc.want)
			}
			if strings.Contains(home, `href=""`) || strings.Contains(home, `href="/#contact"`) || strings.Contains(home, `href="/#about"`) {
				t.Fatal("empty or unavailable CTA remains")
			}
		})
	}
}
