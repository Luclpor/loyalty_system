package dto

import "github.com/google/uuid"

type UserDto struct {
	ID    uuid.UUID
	Login string
}
