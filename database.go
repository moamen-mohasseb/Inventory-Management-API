package main

import (
	"log"
	"os"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// DB is the shared GORM connection used by all handlers.
var DB *gorm.DB

// defaultDSN is the PostgreSQL connection string used when the DATABASE_URL
// environment variable is not set. Change it here or override it with DATABASE_URL.
const defaultDSN = "host=localhost user=postgres password=postgres dbname=inventory port=5432 sslmode=disable TimeZone=UTC"

// getDSN returns the connection string (environment variable first, default second).
func getDSN() string {
	if dsn := os.Getenv("DATABASE_URL"); dsn != "" {
		return dsn
	}
	return defaultDSN
}

// initDatabase initializes the database connection, creates the table and seeds data.
func initDatabase() {
	var err error
	dsn := getDSN()

	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to the database! ", err)
	}

	if err := DB.AutoMigrate(&Item{}); err != nil {
		log.Fatal("Failed to migrate the database! ", err)
	}
	seedDatabase()
}

// seedDatabase will seed the database with 20 default items
func seedDatabase() {
	var count int64
	DB.Model(&Item{}).Count(&count)
	if count == 0 {
		items := []Item{
			{ID: uuid.New().String(), Name: "Laptop", Stock: 10, Price: 999.99},
			{ID: uuid.New().String(), Name: "Smartphone", Stock: 20, Price: 699.99},
			{ID: uuid.New().String(), Name: "Headphones", Stock: 15, Price: 199.99},
			{ID: uuid.New().String(), Name: "Keyboard", Stock: 25, Price: 89.99},
			{ID: uuid.New().String(), Name: "Mouse", Stock: 30, Price: 49.99},
			{ID: uuid.New().String(), Name: "Monitor", Stock: 12, Price: 299.99},
			{ID: uuid.New().String(), Name: "Webcam", Stock: 18, Price: 79.99},
			{ID: uuid.New().String(), Name: "Printer", Stock: 7, Price: 149.99},
			{ID: uuid.New().String(), Name: "Tablet", Stock: 5, Price: 399.99},
			{ID: uuid.New().String(), Name: "Smartwatch", Stock: 14, Price: 249.99},
			{ID: uuid.New().String(), Name: "External Hard Drive", Stock: 8, Price: 119.99},
			{ID: uuid.New().String(), Name: "USB Flash Drive", Stock: 50, Price: 19.99},
			{ID: uuid.New().String(), Name: "Router", Stock: 6, Price: 89.99},
			{ID: uuid.New().String(), Name: "Projector", Stock: 3, Price: 499.99},
			{ID: uuid.New().String(), Name: "Bluetooth Speaker", Stock: 22, Price: 129.99},
			{ID: uuid.New().String(), Name: "Gaming Console", Stock: 11, Price: 499.99},
			{ID: uuid.New().String(), Name: "Camera", Stock: 4, Price: 599.99},
			{ID: uuid.New().String(), Name: "Fitness Tracker", Stock: 16, Price: 99.99},
			{ID: uuid.New().String(), Name: "Drone", Stock: 2, Price: 899.99},
			{ID: uuid.New().String(), Name: "VR Headset", Stock: 9, Price: 399.99},
		}

		if err := DB.Create(&items).Error; err != nil {
			log.Fatal("Failed to seed the database! ", err)
		}
		log.Println("Database seeded with 20 sample items.")
	} else {
		log.Println("Database already contains data, skipping seeding.")
	}
}
