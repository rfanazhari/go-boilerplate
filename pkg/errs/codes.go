package errs

const (
	CodeInvalidTitle           = "INVALID_TITLE"
	CodeInvalidDueDate         = "INVALID_DUE_DATE"
	CodeInvalidIdentifier      = "INVALID_IDENTIFIER"
	CodeValidationFailed       = "VALIDATION_FAILED"
	CodeMalformedRequest       = "MALFORMED_REQUEST"
	CodeTaskNotFound           = "TASK_NOT_FOUND"
	CodeRouteNotFound          = "ROUTE_NOT_FOUND"
	CodeMethodNotAllowed       = "METHOD_NOT_ALLOWED"
	CodeInvalidTransition      = "INVALID_TRANSITION"
	CodeTaskFinal              = "TASK_FINAL"
	CodeEventUnavailable       = "EVENT_UNAVAILABLE"
	CodeInternalError          = "INTERNAL_ERROR"
	CodeInvalidRole            = "INVALID_ROLE"
	CodeInvalidStatus          = "INVALID_STATUS"
	CodeInvalidPassword        = "INVALID_PASSWORD"
	CodeAccountNotFound        = "ACCOUNT_NOT_FOUND"
	CodeEmailTaken             = "EMAIL_TAKEN"
	CodeLastAdmin              = "LAST_ADMIN"
	CodeInvalidCredentials     = "INVALID_CREDENTIALS"
	CodeUnauthenticated        = "UNAUTHENTICATED"
	CodeForbidden              = "FORBIDDEN"
	CodeInvalidBlogTitle       = "INVALID_BLOG_TITLE"
	CodeInvalidBlogContent     = "INVALID_BLOG_CONTENT"
	CodeInvalidBlogPagination  = "INVALID_BLOG_PAGINATION"
	CodeInvalidBlogFilter      = "INVALID_BLOG_FILTER"
	CodeBlogNotFound           = "BLOG_NOT_FOUND"
	CodeBlogDeleted            = "BLOG_DELETED"
	CodeBlogIdentifierConflict = "BLOG_IDENTIFIER_CONFLICT"
)

const (
	KindInvalidInput Kind = iota + 1
	KindNotFound
	KindConflict
	KindUnavailable
	KindInternal
	KindUnauthenticated
	KindForbidden
)

type Kind int

type Error struct {
	Code string
	Kind Kind
	Err  error
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return e.Code
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func New(code string, kind Kind, err error) *Error {
	return &Error{Code: code, Kind: kind, Err: err}
}
