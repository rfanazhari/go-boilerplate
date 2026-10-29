package service

// BlogContent is sanitized HTML and its normalized readable text.
type BlogContent struct {
	HTML       string
	Text       string
	TextLength int
}

func ProcessBlogContent(source string) (BlogContent, error) { return BlogContent{}, nil }
