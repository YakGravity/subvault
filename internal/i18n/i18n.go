package i18n

import (
	"embed"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

//go:embed locales/*.json
var localeFS embed.FS

// LanguageInfo holds a language code and its native display name
type LanguageInfo struct {
	Code string
	Name string
}

// I18nService manages translation bundles and localizer creation
type I18nService struct {
	bundle         *i18n.Bundle
	defaultLang    string
	supportedLangs []string
	languageNames  map[string]string
}

var localeFilePattern = regexp.MustCompile(`^active\.([a-z]{2})\.json$`)

// NewI18nService creates and initializes the i18n service
func NewI18nService(localeDir string) *I18nService {
	bundle := i18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("json", json.Unmarshal)

	langNames := make(map[string]string)
	langSet := make(map[string]bool)

	// Step 1: Load embedded locale files
	embeddedFiles := []string{
		"locales/active.en.json",
		"locales/active.de.json",
	}
	for _, file := range embeddedFiles {
		if _, err := bundle.LoadMessageFileFS(localeFS, file); err != nil {
			slog.Warn("failed to load embedded locale", "file", file, "error", err)
			continue
		}
		code := extractLangCode(filepath.Base(file))
		if code != "" {
			langSet[code] = true
			langNames[code] = readLangSelf(localeFS, file, code)
		}
	}

	// Step 2: Load additional locale files from filesystem directory
	if localeDir != "" {
		if entries, err := os.ReadDir(localeDir); err == nil {
			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}
				name := entry.Name()
				code := extractLangCode(name)
				if code == "" {
					continue
				}
				fullPath := filepath.Join(localeDir, name)
				if _, err := bundle.LoadMessageFile(fullPath); err != nil {
					slog.Warn("failed to load locale file", "path", fullPath, "error", err)
					continue
				}
				langSet[code] = true
				if selfName := readLangSelfFile(fullPath, code); selfName != code {
					langNames[code] = selfName
				}
				slog.Info("loaded locale file", "path", fullPath, "lang", code)
			}
		} else if os.IsNotExist(err) {
			slog.Info("locale directory not found, using built-in locales only", "dir", localeDir)
		} else {
			slog.Warn("failed to read locale directory", "dir", localeDir, "error", err)
		}
	}

	// Step 3: Build sorted supported languages list
	langs := make([]string, 0, len(langSet))
	for code := range langSet {
		langs = append(langs, code)
	}
	sort.Strings(langs)

	slog.Info("i18n initialized", "languages", langs)

	return &I18nService{
		bundle:         bundle,
		defaultLang:    "en",
		supportedLangs: langs,
		languageNames:  langNames,
	}
}

// extractLangCode extracts the language code from a filename like "active.fr.json"
func extractLangCode(filename string) string {
	matches := localeFilePattern.FindStringSubmatch(filename)
	if len(matches) == 2 {
		return matches[1]
	}
	return ""
}

// readLangSelf reads the lang_self key from an embedded locale file
func readLangSelf(fs embed.FS, path string, fallback string) string {
	data, err := fs.ReadFile(path)
	if err != nil {
		return fallback
	}
	return extractLangSelfFromJSON(data, fallback)
}

// readLangSelfFile reads the lang_self key from a filesystem locale file
func readLangSelfFile(path string, fallback string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return fallback
	}
	return extractLangSelfFromJSON(data, fallback)
}

// extractLangSelfFromJSON extracts the lang_self value from JSON data
func extractLangSelfFromJSON(data []byte, fallback string) string {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return fallback
	}
	if val, ok := raw["lang_self"]; ok {
		var name string
		if err := json.Unmarshal(val, &name); err == nil && name != "" {
			return name
		}
	}
	return fallback
}

// NewLocalizer creates a localizer for the given language with English fallback
func (s *I18nService) NewLocalizer(lang string) *i18n.Localizer {
	if lang == "" {
		lang = s.defaultLang
	}
	return i18n.NewLocalizer(s.bundle, lang, s.defaultLang)
}

// T translates a simple message by ID
func (s *I18nService) T(localizer *i18n.Localizer, messageID string) string {
	msg, err := localizer.Localize(&i18n.LocalizeConfig{
		MessageID: messageID,
	})
	if err != nil && msg == "" {
		return messageID
	}
	return msg
}

// TData translates a message with template data
func (s *I18nService) TData(localizer *i18n.Localizer, messageID string, data map[string]interface{}) string {
	msg, err := localizer.Localize(&i18n.LocalizeConfig{
		MessageID:    messageID,
		TemplateData: data,
	})
	if err != nil && msg == "" {
		return messageID
	}
	return msg
}

// TPluralCount translates a message with plural support
func (s *I18nService) TPluralCount(localizer *i18n.Localizer, messageID string, count int, data map[string]interface{}) string {
	if data == nil {
		data = map[string]interface{}{}
	}
	data["Count"] = count

	msg, err := localizer.Localize(&i18n.LocalizeConfig{
		MessageID:    messageID,
		TemplateData: data,
		PluralCount:  count,
	})
	if err != nil && msg == "" {
		return fmt.Sprintf("%s (%d)", messageID, count)
	}
	return msg
}

// SupportedLanguages returns the list of supported language codes
func (s *I18nService) SupportedLanguages() []string {
	return s.supportedLangs
}

// DefaultLanguage returns the default language code
func (s *I18nService) DefaultLanguage() string {
	return s.defaultLang
}

// Languages returns a sorted slice of LanguageInfo for use in templates
func (s *I18nService) Languages() []LanguageInfo {
	result := make([]LanguageInfo, 0, len(s.supportedLangs))
	for _, code := range s.supportedLangs {
		name := s.languageNames[code]
		if name == "" {
			name = strings.ToUpper(code)
		}
		result = append(result, LanguageInfo{Code: code, Name: name})
	}
	return result
}
