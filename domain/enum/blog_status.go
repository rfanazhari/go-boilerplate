package enum

type BlogStatus string

const (
	BlogDraft     BlogStatus = "draft"
	BlogPublished BlogStatus = "published"
)

func ParseBlogStatus(value string) (BlogStatus, bool) {
	return "", false
}
