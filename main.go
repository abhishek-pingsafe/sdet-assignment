package main

import (
	"database/sql"
	"fmt"
	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"math/rand"
	"net/http"
	"regexp"
	"strings"
	"time"
)

type Customer struct {
	ID          string `json:"id" binding:"required"`
	Name        string `json:"name"`
	PhoneNumber string `json:"phone_number" binding:"required"`
}

func main() {
	db := initDB("customers.db")
	defer db.Close()
	router := gin.Default()

	router.POST("/api", func(c *gin.Context) {
		sessionToken := c.Request.Header.Get("x-session-token")
		userAgent := c.Request.Header.Get("user-agent")

		// Check if both headers are present
		if sessionToken != "authorized-user" {
			c.JSON(http.StatusForbidden, gin.H{"error": "request cannot be authenticated!"})
			return
		}

		if strings.Contains(strings.ToLower(userAgent), "bot") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "bad bot, go away!"})
			return
		}

		var customer Customer
		if err := c.ShouldBindJSON(&customer); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Check if name has special characters
		if !isAlpha(customer.Name) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name has special characters"})
			return
		}

		err := insertCustomer(db, customer)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("%v", err)})
			return
		}

		go sendSMS(db, customer.ID)

		c.JSON(http.StatusOK, gin.H{"message": "customer created"})
	})

	router.Run(":8080")
}

func sendSMS(db *sql.DB, customerID string) {
	// Sleep for a random amount of time between 10 and 20 seconds
	time.Sleep(time.Second * time.Duration(10+rand.Intn(10)))

	// Update the sms_sent column to true
	updateSMSSentSQL := "UPDATE customers SET sms_sent = true WHERE id = ?"
	stmt, err := db.Prepare(updateSMSSentSQL)
	if err != nil {
		fmt.Printf("Error preparing statement: %v\n", err)
		return
	}

	_, err = stmt.Exec(customerID)
	if err != nil {
		fmt.Printf("Error updating customer: %v\n", err)
		return
	}
	fmt.Printf("sent sms to customer: %v\n", customerID)
}

func initDB(filepath string) *sql.DB {
	db, err := sql.Open("sqlite3", filepath)
	if err != nil {
		panic(err)
	}
	if db == nil {
		panic("db nil")
	}

	// Create the table if it doesn't exist
	createTableSQL := `
		CREATE TABLE IF NOT EXISTS customers(
		id TEXT NOT NULL PRIMARY KEY,
		name TEXT NOT NULL CHECK(length(name) <= 50),
		phone_number TEXT NOT NULL CHECK(length(phone_number) = 10),
		sms_sent BOOLEAN
	);
    `
	_, err = db.Exec(createTableSQL)
	if err != nil {
		panic(err)
	}

	return db
}

func insertCustomer(db *sql.DB, customer Customer) error {
	insertCustomerSQL := "INSERT INTO customers(id, name, phone_number) VALUES(?, ?, ?)"
	stmt, err := db.Prepare(insertCustomerSQL)
	if err != nil {
		panic(err)
	}

	_, err = stmt.Exec(customer.ID, customer.Name, customer.PhoneNumber)
	if err != nil {
		return err
	}
	return nil
}

func isAlpha(str string) bool {
	match, _ := regexp.MatchString("^[A-Za-z]*$", str)
	return match
}
