package httpserver

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"newpress.local/newpress/internal/presentation"
	"newpress.local/newpress/internal/theme"
)

func renderNexoraRichHome(t *testing.T, manager *theme.Manager, values map[string]any) string {
	t.Helper()
	page := presentation.Fixtures()["empty"]
	page.Settings = manager.SettingsSnapshot().Values
	for key, value := range values {
		page.Settings[key] = value
	}
	page.Modules["feedback"] = true
	var output theme.RenderBuffer
	if err := manager.Render(&output, "home", page); err != nil {
		t.Fatal(err)
	}
	body := output.String()
	if strings.Contains(body, "#ZgotmplZ") || strings.Contains(body, "<no value>") {
		t.Fatal("homepage contains an invalid rendered value")
	}
	return body
}

func TestNexoraRichHomeRequiresRealContent(t *testing.T) {
	manager := nexoraFixtureTheme(t)
	for _, name := range []string{"facts", "partners", "faq"} {
		if manager.SettingsSnapshot().Values["show_"+name] != false {
			t.Fatalf("optional module %s must default to off", name)
		}
	}
	cases := map[string]map[string]any{
		"default":       {},
		"enabled-empty": {"show_facts": true, "show_partners": true, "show_faq": true},
		"incomplete": {
			"show_facts": true, "show_partners": true, "show_faq": true,
			"fact_1_value": "测试数值", "fact_2_label": "只有名称",
			"partner_1_image": "/media/fixture-logo.svg", "partner_2_url": "https://example.test/partner",
			"faq_1_question": "只有问题", "faq_2_answer": "只有回答",
		},
		"disabled-populated": {
			"fact_1_value": "测试数值", "fact_1_label": "测试事实",
			"partner_1_name": "测试合作伙伴", "faq_1_question": "测试问题", "faq_1_answer": "测试回答",
		},
	}
	for name, values := range cases {
		t.Run(name, func(t *testing.T) {
			body := renderNexoraRichHome(t, manager, values)
			for _, section := range []string{"facts", "partners", "faq"} {
				if strings.Contains(body, `data-home-section="`+section+`"`) {
					t.Fatalf("empty or disabled %s section remains", section)
				}
			}
		})
	}
}

func TestNexoraRichHomeEntriesEscapeAndOmitIncompleteItems(t *testing.T) {
	manager := nexoraFixtureTheme(t)
	values := map[string]any{
		"show_facts": true, "show_partners": true, "show_faq": true,
		"facts_title": "", "partners_title": "", "faq_title": "",
		"fact_1_value": "仅数值隐藏", "fact_2_label": "仅名称隐藏",
		"fact_3_value": "示例A", "fact_3_label": "事实A", "fact_4_value": "示例B", "fact_4_label": "事实B",
		"partner_1_image": "/media/unnamed.svg", "partner_2_url": "https://example.test/unnamed",
		"partner_3_name": "文字合作伙伴", "partner_4_name": "带链接的合作伙伴", "partner_4_url": "https://example.test/partner",
		"partner_5_name": "带标志的合作伙伴", "partner_5_image": "/media/partner.svg",
		"partner_6_name": "第六个合作伙伴 <名称>",
		"faq_1_question": "未完成的问题隐藏", "faq_2_answer": "未完成的回答隐藏",
		"faq_3_question": "怎样使用 <文本>？", "faq_3_answer": "<script>alert(1)</script>\n第二行回答",
		"faq_4_question": "第四个问题", "faq_4_answer": "第四个回答",
	}
	body := renderNexoraRichHome(t, manager, values)
	for _, section := range []string{"facts", "partners", "faq"} {
		if strings.Count(body, `data-home-section="`+section+`"`) != 1 {
			t.Fatalf("expected exactly one populated %s section", section)
		}
	}
	for token, count := range map[string]int{`class="fact-item"`: 2, `class="partner-item"`: 4, `class="faq-item"`: 2} {
		if strings.Count(body, token) != count {
			t.Fatalf("expected %d valid entries for %s", count, token)
		}
	}
	for _, absent := range []string{"仅数值隐藏", "仅名称隐藏", "/media/unnamed.svg", "https://example.test/unnamed", "未完成的问题隐藏", "未完成的回答隐藏", "<script>alert(1)</script>", `href=""`, `src=""`} {
		if strings.Contains(body, absent) {
			t.Fatalf("incomplete or unsafe content rendered: %s", absent)
		}
	}
	for _, wanted := range []string{
		`id="facts-title">企业事实`, `id="partners-title">合作伙伴`, `id="faq-title">常见问题`,
		`<div class="partner-display"><span>文字合作伙伴</span></div>`,
		`href="https://example.test/partner"`, `src="/media/partner.svg" alt=""`,
		`第六个合作伙伴 &lt;名称&gt;`, `怎样使用 &lt;文本&gt;？`, "&lt;script&gt;alert(1)&lt;/script&gt;\n第二行回答",
		`<details class="faq-item"><summary>`,
	} {
		if !strings.Contains(body, wanted) {
			t.Fatalf("missing usable optional content: %s", wanted)
		}
	}
}

func TestNexoraRichHomeSequencesRespectVisibility(t *testing.T) {
	manager := nexoraFixtureTheme(t)
	sequences := map[string][]string{
		"classic":        {"hero", "about", "facts", "strengths", "innovation", "business", "news", "partners", "careers", "faq", "contact"},
		"business-first": {"hero", "business", "strengths", "about", "facts", "innovation", "news", "partners", "careers", "faq", "contact"},
		"news-first":     {"hero", "news", "about", "facts", "strengths", "innovation", "business", "partners", "careers", "faq", "contact"},
	}
	sectionPattern := regexp.MustCompile(`data-home-section="([a-z]+)"`)
	for sequence, order := range sequences {
		for mask := 0; mask < 8; mask++ {
			t.Run(fmt.Sprintf("%s/%03b", sequence, mask), func(t *testing.T) {
				values := map[string]any{
					"home_sequence": sequence, "show_about": false, "show_strengths": false,
					"fact_4_value": "示例数值", "fact_4_label": "示例事实", "partner_6_name": "示例合作伙伴",
					"faq_4_question": "示例问题", "faq_4_answer": "示例回答",
				}
				for bit, module := range []string{"facts", "partners", "faq"} {
					values["show_"+module] = mask&(1<<bit) != 0
				}
				body := renderNexoraRichHome(t, manager, values)
				actual, expected := []string{}, []string{}
				for _, match := range sectionPattern.FindAllStringSubmatch(body, -1) {
					actual = append(actual, match[1])
				}
				for _, module := range order {
					if values["show_"+module] != false {
						expected = append(expected, module)
					}
				}
				if strings.Join(actual, ",") != strings.Join(expected, ",") {
					t.Fatalf("section order or visibility mismatch: got %v, want %v", actual, expected)
				}
			})
		}
	}
}

func TestNexoraRichHomeLinksFollowAvailableContent(t *testing.T) {
	manager := nexoraFixtureTheme(t)
	content := map[string]map[string]any{
		"facts":    {"fact_4_value": "示例数值", "fact_4_label": "示例事实"},
		"partners": {"partner_6_name": "示例合作伙伴"},
		"faq":      {"faq_4_question": "示例问题", "faq_4_answer": "示例回答"},
	}
	for module, entries := range content {
		for _, state := range []string{"disabled", "empty", "complete"} {
			t.Run(module+"/"+state, func(t *testing.T) {
				values := map[string]any{
					"show_hero": false, "show_contact": false, "show_" + module: state != "disabled",
					"hero_link": "#" + module, "hero_button": "目标区块按钮", "contact_action_url": "#" + module,
				}
				if state != "empty" {
					for key, value := range entries {
						values[key] = value
					}
				}
				body := renderNexoraRichHome(t, manager, values)
				hasLink := strings.Contains(body, `href="/#`+module+`"`)
				if hasLink != (state == "complete") || strings.Contains(body, `href="#`+module+`"`) {
					t.Fatal("module action is missing, unavailable, or not normalized to the homepage")
				}
				if state != "complete" && (!strings.Contains(body, `href="/feedback/"`) || strings.Contains(body, "目标区块按钮")) {
					t.Fatal("missing content must hide its CTA and use the contact fallback")
				}
			})
		}
	}
}

func TestNexoraRichHomeSettingsValidateContentAndURLs(t *testing.T) {
	manager := nexoraFixtureTheme(t)
	cases := []struct{ key, value string }{
		{"fact_1_value", "   "}, {"partner_1_name", "\t\n"}, {"faq_1_answer", " \n "},
		{"partner_1_url", "javascript:alert(1)"}, {"partner_1_url", "#facts"}, {"partner_1_url", "/#facts"},
		{"partner_1_url", "//example.test"}, {"partner_1_image", "data:image/svg+xml,test"}, {"home_sequence", "invalid"},
	}
	for _, tc := range cases {
		values := manager.SettingsSnapshot().Values
		values[tc.key] = tc.value
		_, _, _, fields := manager.PreviewSettings(values)
		if fields[tc.key] == "" {
			t.Fatalf("setting %s unexpectedly accepted %q", tc.key, tc.value)
		}
	}
	values := manager.SettingsSnapshot().Values
	values["partner_1_name"], values["partner_1_url"], values["partner_1_image"] = "示例合作伙伴", "https://example.test/company", "/media/partner.svg"
	values["home_sequence"] = "business-first"
	if _, _, _, fields := manager.PreviewSettings(values); len(fields) != 0 {
		t.Fatalf("valid optional content rejected: %v", fields)
	}
}
