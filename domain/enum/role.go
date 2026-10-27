package enum

type Role string

const (
	RoleAdmin Role = "admin"
	RoleUser  Role = "user"
)

func ParseRole(value string) (Role, bool) {
	switch Role(value) {
	case RoleAdmin, RoleUser:
		return Role(value), true
	default:
		return "", false
	}
}
