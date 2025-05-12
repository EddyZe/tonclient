package services

import (
	"database/sql"
	"errors"
	"time"
	"tonclient/internal/models"
	"tonclient/internal/repositories"
)

type UserService struct {
	userRepo *repositories.UserRepository
}

func NewUserService(userRepo *repositories.UserRepository) *UserService {
	return &UserService{
		userRepo: userRepo,
	}
}

func (s *UserService) CreateUser(username string, refererId sql.NullInt64) (*models.User, error) {
	user := s.userRepo.FindByUsername(username)
	if user != nil {
		return nil, errors.New("user already exists")
	}

	user = &models.User{
		Username:  username,
		CreatedAt: time.Now(),
		RefererId: refererId,
	}

	if err := s.userRepo.Save(user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *UserService) GetUserReferal(userId uint64) *[]models.User {
	return s.GetUserReferal(userId)
}

func (s *UserService) GetById(id uint64) (*models.User, error) {
	user := s.userRepo.FindById(id)
	if user == nil {
		return nil, errors.New("user not found")
	}

	return user, nil
}

func (s *UserService) GetByUsername(username string) (*models.User, error) {
	user := s.userRepo.FindByUsername(username)
	if user == nil {
		return nil, errors.New("user not found")
	}
	return user, nil
}
