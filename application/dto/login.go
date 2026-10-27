package dto

type LoginInput struct {
	Email    string
	Password string
}

type LoginResult struct {
	Token     string
	TokenType string
}
