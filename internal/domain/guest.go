package domain

type Guest struct {
	DocumentId string `json:"document_id"`
	GivenNames string `json:"given_names"`
	Surname    string `json:"surname"`
	Email      string `json:"email"`
}
