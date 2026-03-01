package service

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"

	"iptv-tool-v2/internal/model"
	"iptv-tool-v2/pkg/auth"
)

var (
	ErrUserExists      = errors.New("user already exists")
	ErrInvalidPassword = errors.New("invalid username or password")
	ErrSystemNotInit   = errors.New("system not initialized, please create admin account first")
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
	if s.IsInitialized() {
		return nil, ErrUserExists
	}

	if len(username) < 3 {
		return nil, fmt.Errorf("username must be at least 3 characters")
	}
	if len(password) < 6 {
		return nil, fmt.Errorf("password must be at least 6 characters")
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
		return fmt.Errorf("user not found")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(oldPassword)); err != nil {
		return fmt.Errorf("old password is incorrect")
	}

	if len(newPassword) < 6 {
		return fmt.Errorf("new password must be at least 6 characters")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	return model.DB.Model(&user).Update("password_hash", string(hash)).Error
}
