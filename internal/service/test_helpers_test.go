package service

// mockLangProvider implements LanguageProvider for tests.
type mockLangProvider struct {
	langs []string
}

func (m *mockLangProvider) SupportedLanguages() []string {
	return m.langs
}

// defaultLangProvider returns a mock LanguageProvider with en/de.
func defaultLangProvider() LanguageProvider {
	return &mockLangProvider{langs: []string{"de", "en"}}
}
