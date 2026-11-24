package main

import (
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	defaultLimit = 10  // page size when ?limit is not provided
	maxLimit     = 100 // upper bound for ?limit
)

// sortableFields maps the accepted ?sort_by values to real column names.
// Using a whitelist prevents SQL injection through the sort parameter.
var sortableFields = map[string]string{
	"name":  "name",
	"stock": "stock",
	"price": "price",
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func internalError(c *gin.Context, err error) {
	log.Printf("internal error: %v", err)
	c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "internal server error"})
}

func notFound(c *gin.Context) {
	c.JSON(http.StatusNotFound, ErrorResponse{Error: "item not found"})
}

// parseID validates the :id path parameter. A malformed UUID can never match an
// existing item, so it is answered with 404 as well.
func parseID(c *gin.Context) (string, bool) {
	parsed, err := uuid.Parse(c.Param("id"))
	if err != nil {
		notFound(c)
		return "", false
	}
	return parsed.String(), true
}

// findItem loads the item referenced by :id, or writes the 404/500 response itself.
func findItem(c *gin.Context) (Item, bool) {
	var item Item

	id, ok := parseID(c)
	if !ok {
		return item, false
	}

	if err := DB.First(&item, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			notFound(c)
		} else {
			internalError(c, err)
		}
		return item, false
	}
	return item, true
}

// bindItemInput reads and validates the JSON body, or writes the 400 response itself.
func bindItemInput(c *gin.Context) (ItemInput, bool) {
	var input ItemInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request body: " + err.Error()})
		return input, false
	}

	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "name must not be blank"})
		return input, false
	}

	// The database column has two decimals, so round before storing/returning.
	*input.Price = math.Round(*input.Price*100) / 100
	return input, true
}

// parseIntQuery reads an optional integer query parameter and checks its range.
func parseIntQuery(c *gin.Context, key string, def, lo, hi int) (int, error) {
	raw := c.Query(key)
	if raw == "" {
		return def, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < lo || value > hi {
		return 0, fmt.Errorf("invalid %s: must be an integer between %d and %d", key, lo, hi)
	}
	return value, nil
}

// likeEscaper escapes the wildcard characters of LIKE/ILIKE so that a search
// for "50%" is treated literally.
var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

// ---------------------------------------------------------------------------
// Handlers (one per endpoint)
// ---------------------------------------------------------------------------

// getItems handles GET /inventory
//
// Query parameters (all optional):
//
//	limit     page size, 1-100 (default 10)
//	offset    number of items to skip (default 0)
//	name      case-insensitive "contains" filter on the item name
//	min_stock only items whose stock is >= this value
//	sort_by   name | stock | price (default name)
//	order     asc | desc (default asc)
func getItems(c *gin.Context) {
	limit, err := parseIntQuery(c, "limit", defaultLimit, 1, maxLimit)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}
	offset, err := parseIntQuery(c, "offset", 0, 0, math.MaxInt32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	// Filtering
	name := strings.TrimSpace(c.Query("name"))
	var minStock *int
	if raw := c.Query("min_stock"); raw != "" {
		value, convErr := strconv.Atoi(raw)
		if convErr != nil || value < 0 {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid min_stock: must be a non-negative integer"})
			return
		}
		minStock = &value
	}

	// Sorting
	sortBy := strings.ToLower(c.DefaultQuery("sort_by", "name"))
	column, ok := sortableFields[sortBy]
	if !ok {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid sort_by: must be one of name, stock, price"})
		return
	}
	order := strings.ToLower(c.DefaultQuery("order", "asc"))
	if order != "asc" && order != "desc" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid order: must be asc or desc"})
		return
	}

	// A fresh query is built for each statement (COUNT and SELECT) with the same filters.
	filtered := func() *gorm.DB {
		query := DB.Model(&Item{})
		if name != "" {
			query = query.Where("name ILIKE ?", "%"+likeEscaper.Replace(name)+"%")
		}
		if minStock != nil {
			query = query.Where("stock >= ?", *minStock)
		}
		return query
	}

	var total int64
	if err := filtered().Count(&total).Error; err != nil {
		internalError(c, err)
		return
	}

	items := []Item{}
	// "id" is a tie-breaker so pages are stable when many items share the same sort value.
	err = filtered().
		Order(column + " " + order).
		Order("id asc").
		Limit(limit).
		Offset(offset).
		Find(&items).Error
	if err != nil {
		internalError(c, err)
		return
	}

	c.JSON(http.StatusOK, ListResponse{
		Data:       items,
		Pagination: PaginationMeta{Total: total, Limit: limit, Offset: offset},
	})
}

// getItem handles GET /inventory/:id
func getItem(c *gin.Context) {
	item, ok := findItem(c)
	if !ok {
		return
	}
	c.JSON(http.StatusOK, item)
}

// createItem handles POST /inventory
func createItem(c *gin.Context) {
	input, ok := bindItemInput(c)
	if !ok {
		return
	}

	item := Item{
		ID:    uuid.New().String(),
		Name:  input.Name,
		Stock: *input.Stock,
		Price: *input.Price,
	}
	if err := DB.Create(&item).Error; err != nil {
		internalError(c, err)
		return
	}

	c.Header("Location", "/inventory/"+item.ID)
	c.JSON(http.StatusCreated, item)
}

// updateItem handles PUT /inventory/:id (full replacement of name, stock and price)
func updateItem(c *gin.Context) {
	item, ok := findItem(c)
	if !ok {
		return
	}
	input, ok := bindItemInput(c)
	if !ok {
		return
	}

	item.Name = input.Name
	item.Stock = *input.Stock
	item.Price = *input.Price
	if err := DB.Save(&item).Error; err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, item)
}

// deleteItem handles DELETE /inventory/:id
func deleteItem(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	result := DB.Delete(&Item{}, "id = ?", id)
	if result.Error != nil {
		internalError(c, result.Error)
		return
	}
	if result.RowsAffected == 0 {
		notFound(c)
		return
	}
	c.JSON(http.StatusOK, MessageResponse{Message: "item deleted"})
}
