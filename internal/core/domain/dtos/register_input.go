package domain_dtos

type RegisterInputDTO struct {
	Username string `json:"username" validate:"required,min=3,max=32"`
	Email    string `json:"email" validate:"required,email,max=255"`
	Password string `json:"password" validate:"required,min=8,max=255"`
}

func NewRegisterInputDTO(username, email, password string) *RegisterInputDTO {
	return &RegisterInputDTO{
		Username: username,
		Email:    email,
		Password: password,
	}
}
