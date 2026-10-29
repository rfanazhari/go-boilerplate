package aggregate

type BlogSlug struct {
	value string
}

func NewBlogSlug(trimmedTitle, suffix string) (BlogSlug, error) {
	return BlogSlug{}, nil
}

func RestoreBlogSlug(stored string) BlogSlug {
	return BlogSlug{value: stored}
}

func (s BlogSlug) String() string {
	return s.value
}
