package middleware

import (
	"aqlesia/internal/constants/model/dto"
	"aqlesia/internal/opa"
	"aqlesia/platform/logger"
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// AuthorizationMiddleware provides OPA-based authorization
type AuthorizationMiddleware struct {
	opaService opa.OPAService
	log        logger.Logger
}

// NewAuthorizationMiddleware creates a new authorization middleware
func NewAuthorizationMiddleware(opaService opa.OPAService, log logger.Logger) *AuthorizationMiddleware {
	return &AuthorizationMiddleware{
		opaService: opaService,
		log:        log,
	}
}

// RequirePermission creates middleware that checks for specific permissions
func (a *AuthorizationMiddleware) RequirePermission(permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user from JWT middleware context
		userInterface, exists := c.Get("auth_user")
		if !exists {
			a.log.Warn(context.Background(), "Authorization middleware called without user context")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
			c.Abort()
			return
		}

		user, ok := userInterface.(*dto.User)
		if !ok {
			a.log.Error(context.Background(), "Invalid user type in context")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			c.Abort()
			return
		}

		// Prepare resource context from URL parameters
		resource := make(map[string]interface{})
		if userID := c.Param("id"); userID != "" {
			resource["user_id"] = userID
		}

		// Evaluate permission using OPA
		allowed, err := a.opaService.EvaluatePermission(c.Request.Context(), user, permission, resource)
		if err != nil {
			a.log.Error(c.Request.Context(), "Authorization evaluation failed", 
				zap.Error(err),
				zap.String("user_id", user.ID.String()),
				zap.String("permission", permission))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Authorization evaluation failed"})
			c.Abort()
			return
		}

		if !allowed {
			a.log.Warn(c.Request.Context(), "Access denied", 
				zap.String("user_id", user.ID.String()),
				zap.String("user_role", user.Role),
				zap.String("permission", permission))
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			c.Abort()
			return
		}

		a.log.Debug(c.Request.Context(), "Access granted", 
			zap.String("user_id", user.ID.String()),
			zap.String("user_role", user.Role),
			zap.String("permission", permission))

		c.Next()
	}
}

// RequireUserAccess creates middleware that checks user-specific access
func (a *AuthorizationMiddleware) RequireUserAccess(action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user from JWT middleware context
		userInterface, exists := c.Get("auth_user")
		if !exists {
			a.log.Warn(context.Background(), "Authorization middleware called without user context")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
			c.Abort()
			return
		}

		user, ok := userInterface.(*dto.User)
		if !ok {
			a.log.Error(context.Background(), "Invalid user type in context")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			c.Abort()
			return
		}

		// Get target user ID from URL parameter
		targetUserID := c.Param("id")
		if targetUserID == "" {
			a.log.Error(c.Request.Context(), "Missing user ID in URL parameter")
			c.JSON(http.StatusBadRequest, gin.H{"error": "User ID is required"})
			c.Abort()
			return
		}

		// Validate UUID format
		if _, err := uuid.Parse(targetUserID); err != nil {
			a.log.Error(c.Request.Context(), "Invalid user ID format", zap.String("user_id", targetUserID))
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID format"})
			c.Abort()
			return
		}

		// Evaluate user access using OPA
		allowed, err := a.opaService.EvaluateUserAccess(c.Request.Context(), user, action, targetUserID)
		if err != nil {
			a.log.Error(c.Request.Context(), "User access evaluation failed", 
				zap.Error(err),
				zap.String("user_id", user.ID.String()),
				zap.String("action", action),
				zap.String("target_user_id", targetUserID))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Authorization evaluation failed"})
			c.Abort()
			return
		}

		if !allowed {
			a.log.Warn(c.Request.Context(), "User access denied", 
				zap.String("user_id", user.ID.String()),
				zap.String("user_role", user.Role),
				zap.String("action", action),
				zap.String("target_user_id", targetUserID))
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			c.Abort()
			return
		}

		a.log.Debug(c.Request.Context(), "User access granted", 
			zap.String("user_id", user.ID.String()),
			zap.String("user_role", user.Role),
			zap.String("action", action),
			zap.String("target_user_id", targetUserID))

		c.Next()
	}
}

// RequireRole creates middleware that checks for specific roles (simple role-based check)
func (a *AuthorizationMiddleware) RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user from JWT middleware context
		userInterface, exists := c.Get("auth_user")
		if !exists {
			a.log.Warn(context.Background(), "Authorization middleware called without user context")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
			c.Abort()
			return
		}

		user, ok := userInterface.(*dto.User)
		if !ok {
			a.log.Error(context.Background(), "Invalid user type in context")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			c.Abort()
			return
		}

		// Check if user has one of the required roles
		hasRole := false
		for _, role := range roles {
			if user.Role == role {
				hasRole = true
				break
			}
		}

		if !hasRole {
			a.log.Warn(c.Request.Context(), "Role access denied", 
				zap.String("user_id", user.ID.String()),
				zap.String("user_role", user.Role),
				zap.Strings("required_roles", roles))
			c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient privileges"})
			c.Abort()
			return
		}

		a.log.Debug(c.Request.Context(), "Role access granted", 
			zap.String("user_id", user.ID.String()),
			zap.String("user_role", user.Role),
			zap.Strings("required_roles", roles))

		c.Next()
	}
}
