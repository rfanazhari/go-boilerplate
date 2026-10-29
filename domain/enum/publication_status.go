package enum

type PublicationStatus string

const (
	PublicationDraft     PublicationStatus = "draft"
	PublicationPublished PublicationStatus = "published"
)

func ParsePublicationStatus(value string) (PublicationStatus, bool) {
	return "", false
}
