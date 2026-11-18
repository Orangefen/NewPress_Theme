package httpserver

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"newpress.local/newpress/internal/auth"
	"newpress.local/newpress/internal/config"
	"newpress.local/newpress/internal/plugin"
	"newpress.local/newpress/internal/theme"
)

func docsHubExportSettings(t *testing.T, m *theme.Manager) []byte {
	t.Helper()
	var out bytes.Buffer
	if err := m.ExportSettings(&out); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

func TestDocsHubSettingsPersistenceAndValidation(t *testing.T) {
	m := docsHubTheme(t)
	initial := m.SettingsSnapshot()
	initialExport := docsHubExportSettings(t, m)
	values := m.SettingsSnapshot().Values
	for key, value := range map[string]any{
		"accent": "#227754", "default_color_mode": "system", "allow_color_switch": false,
		"sidebar_width": 250, "content_width": 1400, "navigation_density": "compact", "reduce_motion": true,
		"hero_eyebrow": "开发指南", "tutorials_count": 6, "latest_count": 3,
		"announcement_text": "新版文档已上线", "announcement_link": "/explore/",
		"list_default_view": "list", "card_cover": "none", "toc_depth": "h2",
		"show_toc": false, "reading_font_size": 18, "comments_intro": "请描述你遇到的具体问题。",
	} {
		values[key] = value
	}
	preview, css, classes, fields := m.PreviewSettings(values)
	if len(fields) != 0 {
		t.Fatal(fields)
	}
	for _, want := range []string{"--accent:#227754;", "--docshub-sidebar-width:250px;", "--docshub-shell-width:1400px;"} {
		if !strings.Contains(css, want) {
			t.Fatalf("missing CSS %s", want)
		}
	}
	if !strings.Contains(classes, "docshub-nav-compact") {
		t.Fatal("navigation class not mapped")
	}
	if !reflect.DeepEqual(initial.Values, m.SettingsSnapshot().Values) || !bytes.Equal(initialExport, docsHubExportSettings(t, m)) {
		t.Fatal("preview mutated live settings")
	}
	form := url.Values{}
	for key, value := range values {
		form.Set(key, fmt.Sprint(value))
	}
	parsed, fields := m.ParseSettingsForm(form)
	if len(fields) != 0 {
		t.Fatal(fields)
	}
	saved, fields, err := m.SaveSettingsRevision(initial.Revision, parsed)
	if err != nil || len(fields) != 0 {
		t.Fatal(fields, err)
	}
	if !reflect.DeepEqual(saved.Values, preview.Values) {
		t.Fatal("form changed setting values")
	}
	if _, _, err := m.SaveSettingsRevision(initial.Revision, initial.Values); !errors.Is(err, theme.ErrSettingsRevisionConflict) {
		t.Fatal("stale revision accepted", err)
	}
	if err := m.Load(); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(saved.Values, m.SettingsSnapshot().Values) || css != m.SettingsCSS() {
		t.Fatal("reload lost saved settings")
	}
	exported := docsHubExportSettings(t, m)
	reset, err := m.ResetSettingsRevision(saved.Revision)
	if err != nil || !reflect.DeepEqual(initial.Values, reset.Values) {
		t.Fatal("defaults not restored", err)
	}
	imported, fields, err := m.ImportSettingsRevision(bytes.NewReader(exported), reset.Revision)
	if err != nil || len(fields) != 0 || !reflect.DeepEqual(imported.Values, saved.Values) {
		t.Fatal("round trip failed", fields, err)
	}
	before := m.SettingsSnapshot()
	beforeInvalidExport := docsHubExportSettings(t, m)
	for _, tc := range []struct {
		key   string
		value any
	}{
		{"sidebar_width", 189}, {"sidebar_width", 281}, {"content_width", 1801}, {"tutorials_count", 0}, {"tutorials_count", 9}, {"latest_count", 2.5},
		{"default_color_mode", "automatic"}, {"toc_depth", "h5"}, {"list_default_view", "tiles"}, {"card_cover", "video"},
		{"accent", "#fff;display:none"}, {"announcement_link", "javascript:alert(1)"}, {"start_link", "data:text/html,invalid"},
		{"show_toc", "false"}, {"comments_intro", strings.Repeat("字", 401)}, {"unknown_docshub_option", true},
	} {
		t.Run(tc.key+fmt.Sprint(tc.value), func(t *testing.T) {
			candidate := m.SettingsSnapshot().Values
			candidate[tc.key] = tc.value
			if _, _, _, fields := m.PreviewSettings(candidate); fields[tc.key] == "" {
				t.Fatal("invalid preview accepted")
			}
			if _, fields, err := m.SaveSettings(candidate); err != nil || fields[tc.key] == "" {
				t.Fatal("invalid save not rejected", fields, err)
			}
			if !reflect.DeepEqual(before.Values, m.SettingsSnapshot().Values) || !bytes.Equal(beforeInvalidExport, docsHubExportSettings(t, m)) {
				t.Fatal("invalid write changed settings")
			}
		})
	}
	legacy := `{"schema":1,"theme":"docshub","theme_version":"1.1.0","values":{"accent":"#336699","hero_title":"已有文档","show_tutorials":false,"start_link":"/posts/start"}}`
	old, fields, err := m.ImportSettingsRevision(strings.NewReader(legacy), m.SettingsSnapshot().Revision)
	if err != nil || len(fields) != 0 {
		t.Fatal(fields, err)
	}
	for key, want := range map[string]any{"accent": "#336699", "hero_title": "已有文档", "show_tutorials": false, "start_link": "/posts/start", "show_toc": true, "default_color_mode": "light", "list_default_view": "grid"} {
		if old.Values[key] != want {
			t.Fatalf("legacy %s=%v want %v", key, old.Values[key], want)
		}
	}
}

func TestDocsHubSettingsAdminFormAndPreview(t *testing.T) {
	m := docsHubTheme(t)
	password := "isolated-docshub-settings-fixture"
	hash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatal(err)
	}
	adminAuth, err := auth.New(auth.Config{Username: "docshub-editor", PasswordHash: hash, AbsoluteTTL: time.Hour, IdleTTL: 15 * time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	token, principal, err := adminAuth.Authenticate("docshub-settings", "docshub-editor", password)
	if err != nil {
		t.Fatal(err)
	}
	cookie := &http.Cookie{Name: auth.CookieName, Value: token, Path: "/"}
	s := New(config.Config{SiteTitle: "DocsHub", PublicURL: "https://example.test", ActiveTheme: "docshub", AdminSessionTTL: time.Hour}, docsHubContent(t), m, plugin.NewManager(t.TempDir()), adminAuth)
	h := s.routes()
	page := serveCookieRequest(h, "GET", "https://example.test/admin/appearance/", cookie, nil)
	if page.Code != 200 {
		t.Fatal("settings page", page.Code)
	}
	fieldCount := 0
	for _, group := range m.SettingsSnapshot().Groups {
		if !strings.Contains(page.Body.String(), group.Title) {
			t.Fatalf("group absent %s", group.Title)
		}
		for _, field := range group.Fields {
			fieldCount++
			if !strings.Contains(page.Body.String(), `name="`+field.Key+`"`) {
				t.Fatalf("control absent %s", field.Key)
			}
		}
	}
	if fieldCount != len(m.SettingsSnapshot().Values) {
		t.Fatal("settings controls incomplete", fieldCount, len(m.SettingsSnapshot().Values))
	}
	form := url.Values{"action": {"save"}, "csrf_token": {principal.CSRFToken}, "revision": {m.SettingsSnapshot().Revision}}
	for key, value := range m.SettingsSnapshot().Values {
		form.Set(key, fmt.Sprint(value))
	}
	form.Set("hero_title", "从配置到上线")
	form.Set("default_color_mode", "dark")
	form.Set("allow_color_switch", "false")
	form.Set("accent", "#227754")
	form.Set("comments_intro", "说说你正在解决的问题。")
	before := docsHubExportSettings(t, m)
	for _, target := range []string{"home", "article"} {
		preview := serveContentForm(h, cookie, "https://example.test/admin/appearance/preview/?page="+target, "https://example.test", form)
		var result map[string]string
		if preview.Code != 200 || json.Unmarshal(preview.Body.Bytes(), &result) != nil {
			t.Fatalf("preview %s: %d %s", target, preview.Code, preview.Body.String())
		}
		if !strings.Contains(result["css"], "--accent:#227754;") || !strings.Contains(result["html"], `data-default-mode="dark"`) {
			t.Fatal("preview ignored candidate values")
		}
		if target == "home" && !strings.Contains(result["html"], "从配置到上线") {
			t.Fatal("hero preview ignored content")
		}
	}
	if !bytes.Equal(before, docsHubExportSettings(t, m)) {
		t.Fatal("HTTP preview persisted candidate")
	}
	response := serveContentForm(h, cookie, "https://example.test/admin/appearance/", "https://example.test", form)
	if response.Code != 303 {
		t.Fatalf("save returned %d %s", response.Code, response.Body.String())
	}
	if err := m.Load(); err != nil {
		t.Fatal(err)
	}
	home := serveCookieRequest(h, "GET", "https://example.test/", nil, nil)
	if home.Code != 200 || !strings.Contains(home.Body.String(), "从配置到上线") || !strings.Contains(home.Body.String(), `data-mode="dark"`) {
		t.Fatal("saved content or default appearance not served")
	}
	export := serveCookieRequest(h, "GET", "https://example.test/admin/appearance/export/", cookie, nil)
	if export.Code != 200 || !strings.Contains(export.Body.String(), "从配置到上线") {
		t.Fatal("admin export failed", export.Code)
	}
}
