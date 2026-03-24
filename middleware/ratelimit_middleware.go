package middleware

import (
	"MonCR/utils"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

func RateLimitByUser() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		claimsVal, exists := ctx.Get("claims")
		if !exists {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error" : "Missing claims",
			})
			return
		}

		claims, ok := claimsVal.(*utils.Claims)
		if !ok {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error" : "Invalid claims",
			})
			return
		}

		userID := fmt.Sprintf("%d", claims.UserID)
		if !utils.GetLimiter(userID).Allow() {
			ctx.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error" : "Too many requests",
			})
			return
		}

		ctx.Next()
	}
}