package enum

type BlogStatus string

const (
	BlogDraft     BlogStatus = "draft"
	BlogPublished BlogStatus = "published"
)

func ParseBlogStatus(value string) (BlogStatus, bool) {
	switch BlogStatus(value) {
	case BlogDraft, BlogPublished:
		return BlogStatus(value), true
	default:
		return "", false
	}
}
