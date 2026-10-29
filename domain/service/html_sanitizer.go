package service

type HTMLSanitizer interface {
	Sanitize(raw string) string
}
