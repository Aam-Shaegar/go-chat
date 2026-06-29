package domain_dtos

type LoginInputDTO struct {
	Email    string `json:"email" validate:"required,email,max=255"`
	Password string `json:"password" validate:"required,min=8,max=255"`
}

func NewLoginInputDTO(email, password string) *LoginInputDTO {
	return &LoginInputDTO{
		Email:    email,
		Password: password,
	}
}
