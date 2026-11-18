package httpserver

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"newpress.local/newpress/internal/theme"
)

func nexoraExportSettings(t *testing.T, manager *theme.Manager) []byte {
	t.Helper()
	var output bytes.Buffer
	if err := manager.ExportSettings(&output); err != nil {
		t.Fatal(err)
	}
	return output.Bytes()
}

func TestNexoraSettingsRoundTripAndPreviewIsolation(t *testing.T) {
	manager := nexoraFixtureTheme(t)
	initial := manager.SettingsSnapshot()
	initialExport := nexoraExportSettings(t, manager)
	values := manager.SettingsSnapshot().Values
	for key, value := range map[string]any{
		"accent": "#245B46", "ink": "#20241f", "paper": "#f5f4ed", "muted": "#657060", "surface": "#fffefa", "border": "#d7dacd",
		"header_background": "#f5f4ed", "header_color": "#20241f", "footer_background": "#202d25", "footer_color": "#e6eddf",
		"heading_font": "sans", "button_shape": "square", "content_density": "compact", "motion_enabled": false,
		"header_sticky": false, "header_transparent": false, "show_header_search": false, "header_contact_label": "商务咨询", "logo_width": 124,
		"show_footer_top": false, "footer_description": "由测试保存的品牌说明", "footer_copyright": "测试企业 版权所有", "show_back_to_top": false,
		"hero_mode": "single", "hero_autoplay": false, "hero_interval": 9, "hero_shade": 0.35, "hero_alignment": "center", "hero_height": "compact",
		"second_image": "/media/second.webp", "third_image": "/media/third.webp", "hero_mobile_image": "/media/mobile.webp", "hero_image_alt": "项目建筑入口",
		"home_sequence": "news-first", "show_facts": true, "fact_1_value": "2026", "fact_1_label": "项目资料更新时间", "show_partners": true, "partner_1_name": "测试合作方", "partner_1_url": "https://partner.example.test/",
		"show_faq": true, "faq_1_question": "如何联系？", "faq_1_answer": "请查看联系信息。\n工作日回复。",
		"contact_details_location": "both", "contact_email": "hello@example.test", "contact_phone": "+86 10 12345678", "contact_address": "测试园区 1 号", "contact_hours": "工作日 09:00–18:00", "contact_map_url": "https://maps.example.test/office", "contact_qr_image": "/media/contact-qr.png", "contact_qr_label": "扫描联系测试团队", "social_1_label": "项目主页", "social_1_url": "https://example.test/company",
		"listing_title": "企业资讯", "listing_layout": "list", "listing_columns": "4", "card_image": false, "card_excerpt_lines": 4,
		"article_width": "narrow", "article_title_align": "left", "article_toc_open": false, "article_updated": true, "article_copy_link": false,
		"comments_title": "读者交流", "comments_intro": "欢迎提交具体问题。", "series_title": "技术专题", "series_description": "按项目阅读。",
	} {
		values[key] = value
	}
	preview, css, classes, fields := manager.PreviewSettings(values)
	if len(fields) != 0 {
		t.Fatalf("preview rejected configured settings: %v", fields)
	}
	if preview.Revision == initial.Revision {
		t.Fatal("preview did not receive a revision for the candidate")
	}
	for _, want := range []string{"--accent:#245b46;", "--paper:#f5f4ed;", "--header-background:#f5f4ed;", "--footer-color:#e6eddf;", "--button-radius:0px;", "--logo-width:124px;", "--hero-overlay:0.35;", "--card-excerpt-lines:4;"} {
		if !strings.Contains(css, want) {
			t.Fatalf("candidate CSS missing %q: %s", want, css)
		}
	}
	for _, want := range []string{"nexora-font-sans", "nexora-density-compact", "nexora-no-motion", "nexora-header-static", "nexora-header-solid", "nexora-hero-align-center", "nexora-hero-height-compact"} {
		if !strings.Contains(" "+classes+" ", " "+want+" ") {
			t.Fatalf("candidate class missing %q: %s", want, classes)
		}
	}
	if current := manager.SettingsSnapshot(); !reflect.DeepEqual(current.Values, initial.Values) || current.Revision != initial.Revision || !current.UpdatedAt.Equal(initial.UpdatedAt) {
		t.Fatal("preview changed live settings")
	}
	if !bytes.Equal(initialExport, nexoraExportSettings(t, manager)) {
		t.Fatal("preview changed the persisted settings export")
	}
	if err := manager.Load(); err != nil {
		t.Fatal(err)
	}
	if current := manager.SettingsSnapshot(); !reflect.DeepEqual(current.Values, initial.Values) || current.Revision != initial.Revision || !current.UpdatedAt.Equal(initial.UpdatedAt) {
		t.Fatal("preview candidate was persisted and became live after reload")
	}

	// Exercise the same typed form parsing used by the settings administration.
	form := url.Values{}
	for key, value := range values {
		form.Set(key, fmt.Sprint(value))
	}
	parsed, fields := manager.ParseSettingsForm(form)
	if len(fields) != 0 {
		t.Fatalf("settings form did not parse: %v", fields)
	}
	saved, fields, err := manager.SaveSettingsRevision(initial.Revision, parsed)
	if err != nil || len(fields) != 0 {
		t.Fatalf("settings save failed: %v %v", fields, err)
	}
	if !reflect.DeepEqual(saved.Values, preview.Values) || saved.Revision != preview.Revision || saved.UpdatedAt.IsZero() {
		t.Fatal("saved settings differ from the validated candidate")
	}
	if _, _, err := manager.SaveSettingsRevision(initial.Revision, initial.Values); !errors.Is(err, theme.ErrSettingsRevisionConflict) {
		t.Fatalf("stale settings revision overwrote a newer save: %v", err)
	}
	if err := manager.Load(); err != nil {
		t.Fatal(err)
	}
	reloaded := manager.SettingsSnapshot()
	if !reflect.DeepEqual(saved.Values, reloaded.Values) || saved.Revision != reloaded.Revision || !saved.UpdatedAt.Equal(reloaded.UpdatedAt) {
		t.Fatal("saved settings did not survive theme reload")
	}
	if manager.SettingsCSS() != css || manager.SettingsBodyClasses() != classes {
		t.Fatal("reloaded settings lost their CSS or body classes")
	}
	exported := nexoraExportSettings(t, manager)
	reset, err := manager.ResetSettingsRevision(reloaded.Revision)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(reset.Values, initial.Values) {
		t.Fatal("reset did not restore schema defaults")
	}
	imported, fields, err := manager.ImportSettingsRevision(bytes.NewReader(exported), reset.Revision)
	if err != nil || len(fields) != 0 || !reflect.DeepEqual(imported.Values, saved.Values) {
		t.Fatalf("export/import round trip lost configured settings: %v %v", fields, err)
	}
	if err := manager.Load(); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(manager.SettingsSnapshot().Values, saved.Values) {
		t.Fatal("imported settings did not survive reload")
	}
}

func TestNexoraSettingsLegacyImportPreservesContentAndAddsDefaults(t *testing.T) {
	manager := nexoraFixtureTheme(t)
	legacy := `{"schema":1,"theme":"nexora","theme_version":"1.0.0","values":{"accent":"#446688","hero_title":"原站点标题","hero_link":"/pages/company","show_hero":false,"show_about":false,"contact_action_url":"https://example.test/contact"}}`
	snapshot, fields, err := manager.ImportSettingsRevision(strings.NewReader(legacy), manager.SettingsSnapshot().Revision)
	if err != nil || len(fields) != 0 {
		t.Fatalf("legacy settings import failed: %v %v", fields, err)
	}
	for key, want := range map[string]any{"accent": "#446688", "hero_title": "原站点标题", "hero_link": "/pages/company", "show_hero": false, "show_about": false, "contact_action_url": "https://example.test/contact", "home_sequence": "classic", "heading_font": "serif", "hero_mode": "carousel", "hero_autoplay": false, "show_facts": false, "show_partners": false, "show_faq": false, "card_image": true, "article_toc": true, "contact_details_location": "hidden"} {
		if snapshot.Values[key] != want {
			t.Fatalf("legacy import %s = %v, want %v", key, snapshot.Values[key], want)
		}
	}
	if len(snapshot.Values) > 200 {
		t.Fatal("settings exceed the core field limit")
	}
	if err := manager.Load(); err != nil {
		t.Fatal(err)
	}
	if current := manager.SettingsSnapshot(); !reflect.DeepEqual(current.Values, snapshot.Values) {
		t.Fatal("legacy defaults changed after reload")
	}
}

func TestNexoraSettingsRejectInvalidValuesWithoutWriting(t *testing.T) {
	manager := nexoraFixtureTheme(t)
	initial := manager.SettingsSnapshot()
	initialExport := nexoraExportSettings(t, manager)
	cases := []struct {
		key   string
		value any
	}{
		{"logo_width", 47}, {"logo_width", 181}, {"hero_interval", 4}, {"hero_interval", 16}, {"hero_interval", 7.5},
		{"hero_shade", -0.01}, {"hero_shade", 0.71}, {"card_excerpt_lines", 0}, {"card_excerpt_lines", 5},
		{"hero_mode", "video"}, {"listing_columns", "5"}, {"article_width", "full"}, {"contact_details_location", "sidebar"},
		{"ink", "red"}, {"accent", "#fff;display:none"}, {"contact_email", "invalid@@example.test"}, {"contact_phone", "tel:example"},
		{"contact_map_url", "javascript:alert(1)"}, {"contact_qr_image", "data:image/svg+xml,example"}, {"social_1_url", "//outside.example.test/"},
		{"second_image", "https://example.test/%ZZ"}, {"third_image", "/media/image\nnext.jpg"}, {"hero_mobile_image", "vbscript:alert(1)"},
		{"comments_intro", strings.Repeat("字", 301)}, {"motion_enabled", "false"}, {"unknown_nexora_setting", true},
	}
	for index, tc := range cases {
		t.Run(fmt.Sprintf("%02d_%s", index, tc.key), func(t *testing.T) {
			values := manager.SettingsSnapshot().Values
			values[tc.key] = tc.value
			if _, _, _, fields := manager.PreviewSettings(values); fields[tc.key] == "" {
				t.Fatalf("preview accepted invalid value %v", tc.value)
			}
			if _, fields, err := manager.SaveSettingsRevision(initial.Revision, values); err != nil || fields[tc.key] == "" {
				t.Fatalf("save did not return a field error: %v %v", fields, err)
			}
			if current := manager.SettingsSnapshot(); current.Revision != initial.Revision || !reflect.DeepEqual(current.Values, initial.Values) {
				t.Fatal("invalid save mutated live settings")
			}
			if !bytes.Equal(initialExport, nexoraExportSettings(t, manager)) {
				t.Fatal("invalid save mutated the persisted document")
			}
		})
	}
	var invalidImport map[string]any
	if err := json.Unmarshal(initialExport, &invalidImport); err != nil {
		t.Fatal(err)
	}
	invalidImport["values"].(map[string]any)["contact_map_url"] = "javascript:alert(1)"
	payload, err := json.Marshal(invalidImport)
	if err != nil {
		t.Fatal(err)
	}
	if _, fields, err := manager.ImportSettingsRevision(bytes.NewReader(payload), initial.Revision); err != nil || fields["contact_map_url"] == "" {
		t.Fatalf("import bypassed URI validation: %v %v", fields, err)
	}
	if err := manager.Load(); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(manager.SettingsSnapshot().Values, initial.Values) {
		t.Fatal("invalid writes changed settings after reload")
	}
}
