package middleware

import (
	"subtrackr/internal/i18n"
	"subtrackr/internal/service"

	"github.com/gin-gonic/gin"
)

// I18nMiddleware creates per-request localizer based on user language setting
func I18nMiddleware(i18nService *i18n.I18nService, settingsService *service.SettingsService) gin.HandlerFunc {
	return func(c *gin.Context) {
		lang := settingsService.GetLanguage()
		localizer := i18nService.NewLocalizer(lang)
		helper := i18n.NewTranslationHelper(i18nService, localizer, lang)

		c.Set("lang", lang)
		c.Set("localizer", localizer)
		c.Set("i18n_helper", helper)

		c.Next()
	}
}
