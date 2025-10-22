package telegram

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"aqlesia/internal/storage"
	"aqlesia/platform/logger"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type Config struct {
	BotToken    string
	AppBaseURL  string
	TokenPepper string
	LinkCodeTTL time.Duration
}

type BotService struct {
	bot         *tgbotapi.BotAPI
	config      Config
	userStorage storage.User
	logger      logger.Logger
	linkCodes   sync.Map // map[string]*LinkCode for temporary link codes
	ctx         context.Context
	cancel      context.CancelFunc
}

type LinkCode struct {
	Code        string
	PhoneNumber string
	ExpiresAt   time.Time
}

func NewBotService(config Config, userStorage storage.User, log logger.Logger) (*BotService, error) {
	bot, err := tgbotapi.NewBotAPI(config.BotToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create telegram bot: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	service := &BotService{
		bot:         bot,
		config:      config,
		userStorage: userStorage,
		logger:      log,
		ctx:         ctx,
		cancel:      cancel,
	}

	return service, nil
}

func (bs *BotService) Start() {
	bs.logger.Info(bs.ctx, "Starting Telegram bot", zap.String("bot_username", bs.bot.Self.UserName))

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 30

	updates := bs.bot.GetUpdatesChan(u)

	// Start cleanup goroutine for expired link codes
	go bs.cleanupExpiredLinkCodes()

	for update := range updates {
		if update.Message == nil {
			continue
		}

		bs.handleMessage(update.Message)
	}
}

func (bs *BotService) Stop() {
	bs.cancel()
	bs.bot.StopReceivingUpdates()
}

func (bs *BotService) handleMessage(message *tgbotapi.Message) {
	chatID := message.Chat.ID
	text := message.Text

	switch {
	case strings.HasPrefix(text, "/start"):
		bs.handleStartCommand(chatID, message.CommandArguments())
	case text == "/reset":
		bs.handleResetCommand(chatID)
	case text == "/link":
		bs.handleLinkCommand(chatID)
	case text == "/help":
		bs.handleHelpCommand(chatID)
	default:
		bs.sendMessage(chatID, "Unknown command. Use /help to see available commands.")
	}
}

func (bs *BotService) handleStartCommand(chatID int64, args string) {
	if args != "" {
		// This is a deep link with a link code
		bs.handleLinkWithCode(chatID, args)
		return
	}

	// Regular start command
	message := "Welcome! This bot helps you with password resets and account linking.\n\n" +
		"Available commands:\n" +
		"/reset - Request a password reset\n" +
		"/link - Link your Telegram account\n" +
		"/help - Show this help message"

	bs.sendMessage(chatID, message)
}

func (bs *BotService) handleLinkWithCode(chatID int64, linkCode string) {
	// Find the link code
	linkCodeData, exists := bs.linkCodes.Load(linkCode)
	if !exists {
		bs.sendMessage(chatID, "Invalid or expired link code. Please try again from the app.")
		return
	}

	code := linkCodeData.(*LinkCode)
	if time.Now().After(code.ExpiresAt) {
		bs.linkCodes.Delete(linkCode)
		bs.sendMessage(chatID, "Link code has expired. Please try again from the app.")
		return
	}

	// Link the Telegram account
	ctx := context.Background()
	telegramID := strconv.FormatInt(chatID, 10)

	_, err := bs.userStorage.UpdateTelegramInfo(ctx, telegramID, true, code.PhoneNumber)
	if err != nil {
		bs.logger.Error(ctx, "Failed to link Telegram account",
			zap.Error(err),
			zap.String("phone", code.PhoneNumber),
			zap.Int64("chat_id", chatID))
		bs.sendMessage(chatID, "Failed to link your account. Please try again.")
		return
	}

	// Remove the used link code
	bs.linkCodes.Delete(linkCode)

	bs.sendMessage(chatID, "✅ Your Telegram account has been successfully linked! You can now receive password reset links here.")
	bs.logger.Info(ctx, "Successfully linked Telegram account",
		zap.String("phone", code.PhoneNumber),
		zap.Int64("chat_id", chatID))
}

func (bs *BotService) handleResetCommand(chatID int64) {
	ctx := context.Background()
	telegramID := strconv.FormatInt(chatID, 10)

	// Check if this Telegram ID is linked to a user
	user, err := bs.userStorage.GetUserByTelegramID(ctx, telegramID)
	if err != nil {
		// Don't reveal whether the account exists or not
		bs.sendMessage(chatID, "To use password reset, please first link your account by visiting the app and following the Telegram linking instructions.")
		return
	}

	// Generate reset token and send it
	token, err := bs.generateResetToken(ctx, user.ID)
	if err != nil {
		bs.logger.Error(ctx, "Failed to generate reset token",
			zap.Error(err),
			zap.String("user_id", user.ID.String()))
		bs.sendMessage(chatID, "Failed to generate reset link. Please try again later.")
		return
	}

	resetURL := fmt.Sprintf("%s/reset-password?token=%s", bs.config.AppBaseURL, token)
	message := fmt.Sprintf("Password Reset Link\n\nClick this link to reset your password (valid for 10 minutes):\n\n%s", resetURL)

	// Send as plain text - Telegram will auto-linkify the URL
	msg := tgbotapi.NewMessage(chatID, message)
	msg.DisableWebPagePreview = true

	if _, err := bs.bot.Send(msg); err != nil {
		bs.logger.Error(ctx, "Failed to send reset link via Telegram", zap.Error(err))
		// Fall back to basic message
		bs.sendMessage(chatID, fmt.Sprintf("Password reset link (valid for 10 minutes):\n%s", resetURL))
	}

	bs.logger.Info(ctx, "Sent password reset link via Telegram",
		zap.String("user_id", user.ID.String()),
		zap.Int64("chat_id", chatID))
}

func (bs *BotService) handleLinkCommand(chatID int64) {
	message := "To link your Telegram account:\n\n" +
		"1. Go to your profile in the app\n" +
		"2. Click on 'Link Telegram Account'\n" +
		"3. Follow the instructions to complete the linking process\n\n" +
		"Once linked, you can use /reset to get password reset links directly here."

	bs.sendMessage(chatID, message)
}

func (bs *BotService) handleHelpCommand(chatID int64) {
	message := "**Available Commands:**\n\n" +
		"/reset - Request a password reset link\n" +
		"/link - Instructions for linking your account\n" +
		"/help - Show this help message\n\n" +
		"**Note:** You need to link your account first using the app before you can request password resets."

	msg := tgbotapi.NewMessage(chatID, message)
	msg.ParseMode = "Markdown"
	bs.bot.Send(msg)
}

func (bs *BotService) sendMessage(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	if _, err := bs.bot.Send(msg); err != nil {
		bs.logger.Error(bs.ctx, "Failed to send Telegram message",
			zap.Error(err),
			zap.Int64("chat_id", chatID))
	}
}

// GenerateLinkCode creates a temporary link code for Telegram account linking
func (bs *BotService) GenerateLinkCode(phoneNumber string) (string, string, error) {
	// Generate random code
	codeBytes := make([]byte, 16)
	if _, err := rand.Read(codeBytes); err != nil {
		return "", "", fmt.Errorf("failed to generate link code: %w", err)
	}

	code := base64.RawURLEncoding.EncodeToString(codeBytes)

	// Store the link code temporarily
	linkCode := &LinkCode{
		Code:        code,
		PhoneNumber: phoneNumber,
		ExpiresAt:   time.Now().Add(bs.config.LinkCodeTTL),
	}

	bs.linkCodes.Store(code, linkCode)

	// Generate Telegram deep link
	telegramURL := fmt.Sprintf("https://t.me/%s?start=%s", bs.bot.Self.UserName, code)

	return code, telegramURL, nil
}

// SendResetLink sends a password reset link to user's Telegram
func (bs *BotService) SendResetLink(ctx context.Context, telegramID string, resetToken string) error {
	chatID, err := strconv.ParseInt(telegramID, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid telegram ID: %w", err)
	}

	resetURL := fmt.Sprintf("%s?token=%s", bs.config.AppBaseURL, resetToken)
	message := fmt.Sprintf("Password Reset Link\n\nClick this link to reset your password (valid for 10 minutes):\n\n%s", resetURL)

	msg := tgbotapi.NewMessage(chatID, message)
	msg.DisableWebPagePreview = true

	if _, err := bs.bot.Send(msg); err != nil {
		return fmt.Errorf("failed to send Telegram message: %w", err)
	}

	return nil
}

// SendOTP sends a one-time password to user's Telegram
func (bs *BotService) SendOTP(ctx context.Context, telegramID string, otp string) error {
	chatID, err := strconv.ParseInt(telegramID, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid telegram ID: %w", err)
	}

	message := fmt.Sprintf("🔐 Password Reset OTP\n\nYour one-time password is: **%s**\n\nThis code will expire in 5 minutes.\n\nDo not share this code with anyone.", otp)

	msg := tgbotapi.NewMessage(chatID, message)
	msg.ParseMode = "Markdown"

	if _, err := bs.bot.Send(msg); err != nil {
		return fmt.Errorf("failed to send OTP via Telegram: %w", err)
	}

	return nil
}

func (bs *BotService) generateResetToken(ctx context.Context, userID uuid.UUID) (string, error) {
	// Generate random token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}

	token := base64.RawURLEncoding.EncodeToString(tokenBytes)

	// Create HMAC hash
	mac := hmac.New(sha256.New, []byte(bs.config.TokenPepper))
	mac.Write([]byte(token))
	tokenHash := hex.EncodeToString(mac.Sum(nil))

	// Store in database
	expiresAt := time.Now().Add(10 * time.Minute)
	_, err := bs.userStorage.CreatePasswordResetToken(ctx, userID, tokenHash, expiresAt)
	if err != nil {
		return "", fmt.Errorf("failed to store reset token: %w", err)
	}

	return token, nil
}

func (bs *BotService) cleanupExpiredLinkCodes() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-bs.ctx.Done():
			return
		case <-ticker.C:
			now := time.Now()
			bs.linkCodes.Range(func(key, value interface{}) bool {
				linkCode := value.(*LinkCode)
				if now.After(linkCode.ExpiresAt) {
					bs.linkCodes.Delete(key)
				}
				return true
			})
		}
	}
}
