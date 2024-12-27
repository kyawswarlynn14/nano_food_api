package middlewares

import (
	"net/http"
	"strings"

	token "nano_food_api/tokens"

	"github.com/gin-gonic/gin"
)

func Authentication(roles []int) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.Request.Header.Get("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header not provided"})
			c.Abort()
			return
		}

		// Parse Bearer token
		fields := strings.Fields(authHeader)
		if len(fields) != 2 || strings.ToLower(fields[0]) != "bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
			c.Abort()
			return
		}

		clientToken := fields[1]
		claims, err := token.ValidateToken(clientToken)
		if err != "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err})
			c.Abort()
			return
		}

		isAuthorized := false
		if len(roles) > 0 {
			for _, role := range roles {
				if claims.Role == role {
					isAuthorized = true
					break
				}
			}
		} else {
			isAuthorized = true
		}

		if !isAuthorized {
			c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "Access denied"})
			c.Abort()
			return
		}

		// Set user information in context
		c.Set("email", claims.Email)
		c.Set("userId", claims.User_ID)
		c.Set("role", claims.Role)
		c.Next()
	}
}
