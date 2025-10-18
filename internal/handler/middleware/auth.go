package middleware

import (
	"aqlesia/internal/auth"
	"aqlesia/internal/constants/model/dto"
	"aqlesia/internal/storage"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	AuthorizationHeaderKey = "authorization"
	AuthorizationTypeBearer = "bearer"
	AuthorizationPayloadKey = "authorization_payload"
)

// AuthMiddleware creates a middleware function that validates JWT tokens and fetches user details
func AuthMiddleware(jwtManager *auth.JWTManager, userStorage storage.User) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		authorizationHeader := ctx.GetHeader(AuthorizationHeaderKey)
		
		if len(authorizationHeader) == 0 {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"ok": false,
				"error": gin.H{
					"code": http.StatusUnauthorized,
					"message": "authorization header is required",
				},
			})
			return
		}

		fields := strings.Fields(authorizationHeader)
		if len(fields) < 2 {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"ok": false,
				"error": gin.H{
					"code": http.StatusUnauthorized,
					"message": "invalid authorization header format",
				},
			})
			return
		}

		authorizationType := strings.ToLower(fields[0])
		if authorizationType != AuthorizationTypeBearer {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"ok": false,
				"error": gin.H{
					"code": http.StatusUnauthorized,
					"message": "unsupported authorization type",
				},
			})
			return
		}

		accessToken := fields[1]
		claims, err := jwtManager.ValidateToken(accessToken)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"ok": false,
				"error": gin.H{
					"code": http.StatusUnauthorized,
					"message": "invalid or expired token",
				},
			})
			return
		}

		// Ensure it's an access token
		if claims.TokenType != auth.TokenTypeAccess {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"ok": false,
				"error": gin.H{
					"code": http.StatusUnauthorized,
					"message": "invalid token type",
				},
			})
			return
		}

		// Fetch full user details from storage
		userID, err := uuid.Parse(claims.UserID)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"ok": false,
				"error": gin.H{
					"code": http.StatusUnauthorized,
					"message": "invalid user ID in token",
				},
			})
			return
		}

		user, err := userStorage.Get(ctx, userID)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"ok": false,
				"error": gin.H{
					"code": http.StatusUnauthorized,
					"message": "user not found or invalid token",
				},
			})
			return
		}

		// Store auth context in the request context (for backward compatibility)
		authContext := &dto.AuthContext{
			UserID:      claims.UserID,
			PhoneNumber: claims.PhoneNumber,
			Claims:      claims,
		}
		ctx.Set(AuthorizationPayloadKey, authContext)

		// Store full user details for authorization middleware
		ctx.Set("auth_user", user)
		ctx.Next()
	}
}

// GetAuthContext extracts the authentication context from the Gin context
func GetAuthContext(ctx *gin.Context) (*dto.AuthContext, bool) {
	value, exists := ctx.Get(AuthorizationPayloadKey)
	if !exists {
		return nil, false
	}

	authContext, ok := value.(*dto.AuthContext)
	return authContext, ok
}
