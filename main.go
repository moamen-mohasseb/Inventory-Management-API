package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
)

const (
	// Rate limit: 1 request per second per client, with bursts of up to 5 requests.
	rateLimitPerSecond = 1
	rateLimitBurst     = 5
)

// setupRouter creates the Gin engine, attaches the middleware and registers the routes.
func setupRouter() *gin.Engine {
	router := gin.Default()

	// Do not trust proxy headers: the client IP is the real TCP peer address.
	if err := router.SetTrustedProxies(nil); err != nil {
		log.Fatal(err)
	}

	// Rate limiting is applied to every route through Gin's Use().
	router.Use(RateLimitMiddleware(NewRateLimiter(rateLimitPerSecond, rateLimitBurst)))

	inventory := router.Group("/inventory")
	{
		inventory.GET("", getItems)          // list (pagination, filtering, sorting)
		inventory.POST("", createItem)       // create
		inventory.GET("/:id", getItem)       // read one
		inventory.PUT("/:id", updateItem)    // update
		inventory.DELETE("/:id", deleteItem) // delete
	}

	return router
}

func main() {
	initDatabase()
	log.Println("Server successfully connected to the database and seeded data.")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	router := setupRouter()
	log.Printf("Inventory API listening on http://localhost:%s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}
