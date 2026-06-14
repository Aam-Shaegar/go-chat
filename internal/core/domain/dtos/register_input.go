package domain_dtos

type RegisterInputDTO struct {
	Username string `json:"username" validate:"required,min=3,max=100"`
	Email    string `json:"email" validate:"required,min=5,max=255"`
	Password string `json:"password" validate:"required,min=8,max=255"`
}

func NewRegisterInputDTO(username, email, password string) *RegisterInputDTO {
	return &RegisterInputDTO{
		Username: username,
		Email:    email,
		Password: password,
	}
}
