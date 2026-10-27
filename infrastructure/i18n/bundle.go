package i18n

import (
	"io/fs"

	"github.com/BurntSushi/toml"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

func Load(files fs.FS) (*i18n.Bundle, error) {
	bundle := i18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("toml", toml.Unmarshal)
	if _, err := bundle.LoadMessageFileFS(files, "errors.en.toml"); err != nil {
		return nil, err
	}
	if _, err := bundle.LoadMessageFileFS(files, "errors.id.toml"); err != nil {
		return nil, err
	}
	return bundle, nil
}

func Message(bundle *i18n.Bundle, acceptLanguage, code string) string {
	localizer := i18n.NewLocalizer(bundle, acceptLanguage, "en")
	msg, err := localizer.Localize(&i18n.LocalizeConfig{MessageID: code})
	if err != nil {
		return code
	}
	return msg
}
