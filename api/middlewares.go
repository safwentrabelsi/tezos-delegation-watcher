package api

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

func ValidateYearParam(minValidYear int) gin.HandlerFunc {
	return func(c *gin.Context) {
		yearStr := c.Query("year")
		if yearStr != "" {
			year, err := strconv.Atoi(yearStr)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Year must be a valid number"})
				c.Abort()
				return
			}

			currentYear := time.Now().Year()
			if year < minValidYear || year > currentYear {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Year must be between %d and %d", minValidYear, currentYear)})
				c.Abort()
				return
			}
		}
		c.Next()
	}
}
