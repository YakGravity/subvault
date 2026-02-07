package handlers

import (
	"subtrackr/internal/i18n"
	"subtrackr/internal/version"

	"github.com/gin-gonic/gin"
)

// baseTemplateData creates a gin.H with common template data including i18n
func baseTemplateData(c *gin.Context) gin.H {
	data := gin.H{}

	if helper, exists := c.Get("i18n_helper"); exists {
		data["T"] = helper.(*i18n.TranslationHelper)
	}
	if lang, exists := c.Get("lang"); exists {
		data["Lang"] = lang.(string)
	} else {
		data["Lang"] = "en"
	}

	data["CurrentPath"] = c.Request.URL.Path
	data["Version"] = version.GetVersion()

	if token, exists := c.Get("csrf_token"); exists {
		data["CSRFToken"] = token.(string)
	}

	return data
}

// mergeTemplateData merges additional data into the base template data
func mergeTemplateData(base gin.H, extra gin.H) gin.H {
	for k, v := range extra {
		base[k] = v
	}
	return base
}

// getTranslator returns the TranslationHelper from the context for use in handlers
func getTranslator(c *gin.Context) *i18n.TranslationHelper {
	if helper, exists := c.Get("i18n_helper"); exists {
		return helper.(*i18n.TranslationHelper)
	}
	return nil
}

// tr translates a message ID using the context's translator, with English fallback
func tr(c *gin.Context, messageID string, fallback string) string {
	if t := getTranslator(c); t != nil {
		if translated := t.Tr(messageID); translated != messageID {
			return translated
		}
	}
	return fallback
}
