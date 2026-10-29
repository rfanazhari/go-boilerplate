package aggregate

type BlogTitle struct {
	value string
}

func NewBlogTitle(raw string) (BlogTitle, error) {
	return BlogTitle{}, nil
}

func RestoreBlogTitle(stored string) BlogTitle {
	return BlogTitle{value: stored}
}

func (t BlogTitle) String() string {
	return t.value
}
