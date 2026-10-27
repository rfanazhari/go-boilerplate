package enum

type AccountStatus string

const (
	AccountActive    AccountStatus = "active"
	AccountNonActive AccountStatus = "non_active"
)

func ParseAccountStatus(value string) (AccountStatus, bool) {
	switch AccountStatus(value) {
	case AccountActive, AccountNonActive:
		return AccountStatus(value), true
	default:
		return "", false
	}
}
