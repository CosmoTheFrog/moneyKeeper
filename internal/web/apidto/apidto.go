// Package apidto описывает формат JSON API для Mini App. Модели наружу не сериализуются.
package apidto

const CodeUnauthorized = "unauthorized"

type MeResponse struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
}

type ErrorResponse struct {
	Code string `json:"code"`
}
