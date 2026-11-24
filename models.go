package main

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Item represents a single product stored in the inventory table.
// GORM maps this struct to the "items" table in PostgreSQL.
type Item struct {
	ID    string  `json:"id" gorm:"type:uuid;primaryKey"`
	Name  string  `json:"name" gorm:"type:varchar(255);not null;index"`
	Stock int     `json:"stock" gorm:"not null;index"`
	Price float64 `json:"price" gorm:"type:numeric(10,2);not null;index"`
}

// BeforeCreate is a GORM hook: it guarantees every new item gets a unique UUID
// even if the caller forgot to set one.
func (i *Item) BeforeCreate(tx *gorm.DB) error {
	if i.ID == "" {
		i.ID = uuid.New().String()
	}
	return nil
}

// ItemInput is the JSON body accepted by POST /inventory and PUT /inventory/:id.
// Stock and Price are pointers so that a missing field (nil) can be told apart
// from a legitimate zero value.
type ItemInput struct {
	Name  string   `json:"name" binding:"required,max=255"`
	Stock *int     `json:"stock" binding:"required,gte=0"`
	Price *float64 `json:"price" binding:"required,gte=0,lte=99999999.99"`
}

// ErrorResponse is the JSON shape used for every error returned by the API.
type ErrorResponse struct {
	Error string `json:"error"`
}

// MessageResponse is used for simple success messages (e.g. after a delete).
type MessageResponse struct {
	Message string `json:"message"`
}

// PaginationMeta describes the page that was returned by GET /inventory.
type PaginationMeta struct {
	Total  int64 `json:"total"`  // number of items matching the filters
	Limit  int   `json:"limit"`  // page size
	Offset int   `json:"offset"` // number of items skipped
}

// ListResponse is the JSON shape returned by GET /inventory.
type ListResponse struct {
	Data       []Item         `json:"data"`
	Pagination PaginationMeta `json:"pagination"`
}
