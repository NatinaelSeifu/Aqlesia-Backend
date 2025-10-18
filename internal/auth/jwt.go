package auth

import (
	"aqlesia/internal/constants/model/dto"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	// AccessTokenDuration defines how long access tokens are valid
	AccessTokenDuration = 15 * time.Minute
	// RefreshTokenDuration defines how long refresh tokens are valid  
	RefreshTokenDuration = 7 * 24 * time.Hour // 7 days
	// TokenTypeAccess represents access token type
	TokenTypeAccess = "access"
	// TokenTypeRefresh represents refresh token type
	TokenTypeRefresh = "refresh"
)

// JWTManager handles JWT token operations
type JWTManager struct {
	secretKey     string
	accessTokenDuration  time.Duration
	refreshTokenDuration time.Duration
}

// NewJWTManager creates a new JWT manager instance
func NewJWTManager(secretKey string) *JWTManager {
	return &JWTManager{
		secretKey:            secretKey,
		accessTokenDuration:  AccessTokenDuration,
		refreshTokenDuration: RefreshTokenDuration,
	}
}

// GenerateTokenPair generates both access and refresh tokens for a user
func (m *JWTManager) GenerateTokenPair(userID, phoneNumber string) (accessToken, refreshToken string, err error) {
	// Generate access token
	accessToken, err = m.generateToken(userID, phoneNumber, TokenTypeAccess, m.accessTokenDuration)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate access token: %w", err)
	}

	// Generate refresh token
	refreshToken, err = m.generateToken(userID, phoneNumber, TokenTypeRefresh, m.refreshTokenDuration)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return accessToken, refreshToken, nil
}

// generateToken creates a JWT token with the specified claims
func (m *JWTManager) generateToken(userID, phoneNumber, tokenType string, duration time.Duration) (string, error) {
	now := time.Now()
	expiresAt := now.Add(duration)

	claims := &dto.JWTClaims{
		UserID:      userID,
		PhoneNumber: phoneNumber,
		TokenType:   tokenType,
		ExpiresAt:   expiresAt.Unix(),
		IssuedAt:    now.Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":      claims.UserID,
		"phone_number": claims.PhoneNumber,
		"token_type":   claims.TokenType,
		"exp":          claims.ExpiresAt,
		"iat":          claims.IssuedAt,
	})

	tokenString, err := token.SignedString([]byte(m.secretKey))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// ValidateToken validates a JWT token and returns the claims
func (m *JWTManager) ValidateToken(tokenString string) (*dto.JWTClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Validate the signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(m.secretKey), nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid token claims")
	}

	// Extract claims
	userID, ok := claims["user_id"].(string)
	if !ok {
		return nil, errors.New("invalid user_id claim")
	}

	phoneNumber, ok := claims["phone_number"].(string)
	if !ok {
		return nil, errors.New("invalid phone_number claim")
	}

	tokenType, ok := claims["token_type"].(string)
	if !ok {
		return nil, errors.New("invalid token_type claim")
	}

	exp, ok := claims["exp"].(float64)
	if !ok {
		return nil, errors.New("invalid exp claim")
	}

	iat, ok := claims["iat"].(float64)
	if !ok {
		return nil, errors.New("invalid iat claim")
	}

	jwtClaims := &dto.JWTClaims{
		UserID:      userID,
		PhoneNumber: phoneNumber,
		TokenType:   tokenType,
		ExpiresAt:   int64(exp),
		IssuedAt:    int64(iat),
	}

	// Check if token is expired
	if time.Now().Unix() > jwtClaims.ExpiresAt {
		return nil, errors.New("token is expired")
	}

	return jwtClaims, nil
}

// RefreshAccessToken generates a new access token using a valid refresh token
func (m *JWTManager) RefreshAccessToken(refreshTokenString string) (newAccessToken string, err error) {
	// Validate the refresh token
	claims, err := m.ValidateToken(refreshTokenString)
	if err != nil {
		return "", fmt.Errorf("invalid refresh token: %w", err)
	}

	// Ensure it's a refresh token
	if claims.TokenType != TokenTypeRefresh {
		return "", errors.New("token is not a refresh token")
	}

	// Generate new access token
	newAccessToken, err = m.generateToken(claims.UserID, claims.PhoneNumber, TokenTypeAccess, m.accessTokenDuration)
	if err != nil {
		return "", fmt.Errorf("failed to generate new access token: %w", err)
	}

	return newAccessToken, nil
}

// GenerateRandomSecret generates a random secret key for JWT signing
func GenerateRandomSecret() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}
