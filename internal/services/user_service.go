package services

import (
	"errors"
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

func (s *UserService) CreateUser(user *models.User) (*models.User, error) {
	u := s.userRepo.FindByUsername(user.Username)
	if u != nil {
		return nil, errors.New("user already exists")
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

func (s *UserService) GetByTelegramChatId(telegramChatId uint64) (*models.User, error) {
	user := s.userRepo.FindByTelegramChatId(telegramChatId)
	if user == nil {
		return nil, errors.New("user not found")
	}

	return user, nil
}

func (s *UserService) DeleteById(id uint64) error {
	user := s.userRepo.FindById(id)
	if user == nil {
		return errors.New("user not found")
	}

	err := s.userRepo.DeleteById(id)
	if err != nil {
		return err
	}

	return nil
}
