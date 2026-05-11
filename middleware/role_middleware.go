package middleware

import (
	"MonCR/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

// RoleMiddleware checks if the user has one of the allowed roles
func RoleMiddleware(allowedRoles ...string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// Ensure AuthMiddleware has already set claims
		claimsVal, exists := ctx.Get("claims")
		if !exists {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized. Missing claims."})
			return
		}

		claims, ok := claimsVal.(*utils.Claims)
		if !ok {
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse claims"})
			return
		}

		userRole := claims.Role

		// Check if userRole is in allowedRoles
		isAllowed := false
		for _, role := range allowedRoles {
			if userRole == role {
				isAllowed = true
				break
			}
		}

		if !isAllowed {
			ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "Access restricted. Insufficient permissions.",
			})
			return
		}

		ctx.Next()
	}
}
