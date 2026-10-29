package service

type HTMLSanitizer struct{}

func (HTMLSanitizer) Sanitize(raw string) string {
	return raw
}
