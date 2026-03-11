package middleware

import (
	"MonCR/utils"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// get the token from header
		authToken := ctx.GetHeader("Authorization")

		// check the "Bearer " string
		if authToken == "" || !strings.HasPrefix(authToken, "Bearer ") {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error" : "Missing or invalid authorization header"})
			return 
		}

		// trim the prefix
		tokenString := strings.TrimPrefix(authToken, "Bearer ")

		// get the data from token
		claims, err := utils.ValidateJWT(tokenString)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		ctx.Set("claims", claims)

		ctx.Next()
	}
}