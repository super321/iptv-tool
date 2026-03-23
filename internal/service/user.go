package service

import (
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"iptv-tool-v2/internal/model"
	"iptv-tool-v2/pkg/auth"
	"iptv-tool-v2/pkg/utils"
)

var (
	ErrUserExists      = errors.New("error.user_exists")
	ErrInvalidPassword = errors.New("error.invalid_credentials")
	ErrSystemNotInit   = errors.New("error.system_not_init")
)

// UserService handles user-related business logic
type UserService struct{}

func NewUserService() *UserService {
	return &UserService{}
}

// IsInitialized checks whether the system has been initialized (at least one admin user exists)
func (s *UserService) IsInitialized() bool {
	var count int64
	model.DB.Model(&model.User{}).Count(&count)
	return count > 0
}

// Register creates the first admin user. Only allowed when no users exist.
func (s *UserService) Register(username, password string) (*model.User, error) {
	// Trim whitespace from username
	username = strings.TrimSpace(username)

	if s.IsInitialized() {
		return nil, ErrUserExists
	}

	if len(username) < 3 {
		return nil, fmt.Errorf("error.username_min_length")
	}
	if len(password) < 6 {
		return nil, fmt.Errorf("error.password_min_length")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user := &model.User{
		Username:     username,
		PasswordHash: string(hash),
	}

	if err := model.DB.Create(user).Error; err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

// Login validates credentials and returns a JWT token
func (s *UserService) Login(username, password string) (string, error) {
	if !s.IsInitialized() {
		return "", ErrSystemNotInit
	}

	var user model.User
	if err := model.DB.Where("username = ?", username).First(&user).Error; err != nil {
		return "", ErrInvalidPassword
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", ErrInvalidPassword
	}

	token, err := auth.GenerateToken(user.ID, user.Username)
	if err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}

	return token, nil
}

// ChangePassword changes the password for a user
func (s *UserService) ChangePassword(userID uint, oldPassword, newPassword string) error {
	var user model.User
	if err := model.DB.First(&user, userID).Error; err != nil {
		return fmt.Errorf("error.user_not_found")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(oldPassword)); err != nil {
		return fmt.Errorf("error.wrong_old_password")
	}

	if len(newPassword) < 6 {
		return fmt.Errorf("error.new_password_min_length")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	return model.DB.Model(&user).Update("password_hash", string(hash)).Error
}

// ResetCredentials resets the admin user's username and password.
// Returns the generated plaintext password. Only allowed when the system is already initialized.
func (s *UserService) ResetCredentials(newUsername string) (string, error) {
	if !s.IsInitialized() {
		return "", ErrSystemNotInit
	}

	// Trim whitespace from username
	newUsername = strings.TrimSpace(newUsername)
	if len(newUsername) < 3 {
		return "", fmt.Errorf("error.username_min_length")
	}

	// Get the first (admin) user
	var user model.User
	if err := model.DB.First(&user).Error; err != nil {
		return "", fmt.Errorf("failed to find user: %w", err)
	}

	// Generate a random 8-character password
	newPassword := utils.GenerateRandomPassword(8)

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}

	// Update username and password
	if err := model.DB.Model(&user).Updates(map[string]interface{}{
		"username":      newUsername,
		"password_hash": string(hash),
	}).Error; err != nil {
		return "", fmt.Errorf("failed to update credentials: %w", err)
	}

	return newPassword, nil
}
