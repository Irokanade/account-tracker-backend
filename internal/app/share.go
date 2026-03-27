package app

import (
	"context"
	"crypto/rand"
	"math/big"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Generate a random 6-character alphanumeric code
func generateShareCode() (string, error) {
	const charset = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // Avoid ambiguous chars O/0, I/1
	result := make([]byte, 6)
	for i := range result {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		result[i] = charset[num.Int64()]
	}
	return string(result), nil
}

func shareBookHandler(c *gin.Context) {
	var payload interface{}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if dbPool == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database not connected"})
		return
	}

	code, err := generateShareCode()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate code"})
		return
	}

	ctx := context.Background()
	_, err = dbPool.Exec(ctx,
		"INSERT INTO shared_spaces (code, payload, updated_at) VALUES ($1, $2, $3)",
		code, payload, time.Now())
	
	if err != nil {
		// Basic collision retry (one attempt)
		code, _ = generateShareCode()
		_, err = dbPool.Exec(ctx,
			"INSERT INTO shared_spaces (code, payload, updated_at) VALUES ($1, $2, $3)",
			code, payload, time.Now())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save shared book"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"code": code})
}

func getSharedBookHandler(c *gin.Context) {
	code := c.Param("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Code is required"})
		return
	}

	if dbPool == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database not connected"})
		return
	}

	var payload interface{}
	err := dbPool.QueryRow(context.Background(),
		"SELECT payload FROM shared_spaces WHERE code = $1", code).Scan(&payload)
	
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Shared book not found"})
		return
	}

	c.JSON(http.StatusOK, payload)
}

func updateSharedBookHandler(c *gin.Context) {
	code := c.Param("code")
	var payload interface{}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if dbPool == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database not connected"})
		return
	}

	ctx := context.Background()
	res, err := dbPool.Exec(ctx,
		"UPDATE shared_spaces SET payload = $1, updated_at = $2 WHERE code = $3",
		payload, time.Now(), code)
	
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update shared book"})
		return
	}

	if res.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Shared book not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
