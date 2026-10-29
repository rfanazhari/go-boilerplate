package enum

type PublicationStatus string

const (
	PublicationDraft     PublicationStatus = "draft"
	PublicationPublished PublicationStatus = "published"
)

func ParsePublicationStatus(value string) (PublicationStatus, bool) {
	switch PublicationStatus(value) {
	case PublicationDraft, PublicationPublished:
		return PublicationStatus(value), true
	default:
		return "", false
	}
}
