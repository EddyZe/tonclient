package services

import (
	"errors"
	"tonclient/internal/models"
	"tonclient/internal/repositories"
)

type TelegramService struct {
	telegramRepo *repositories.TelegramRepository
	userService  *UserService
}

func NewTelegramService(tgRepo *repositories.TelegramRepository, userServ *UserService) *TelegramService {
	return &TelegramService{
		telegramRepo: tgRepo,
		userService:  userServ,
	}
}

func (s *TelegramService) CreateTelegram(userId uint64, tgUsername string, tgId uint64) (*models.Telegram, error) {
	user, err := s.userService.GetById(userId)
	if user == nil {
		return nil, err
	}

	tg := &models.Telegram{
		Username:   tgUsername,
		TelegramId: tgId,
	}

	if err := s.telegramRepo.Save(tg); err != nil {
		return nil, err
	}
	return tg, nil
}

func (s *TelegramService) GetId(id uint64) (*models.Telegram, error) {
	telegram := s.telegramRepo.FindById(id)
	if telegram == nil {
		return nil, errors.New("telegram not found")
	}
	return telegram, nil
}

func (s *TelegramService) GetTelegramId(telegramId uint64) (*models.Telegram, error) {
	telegram := s.telegramRepo.FindByTelegramId(telegramId)
	if telegram == nil {
		return nil, errors.New("telegram not found")
	}
	return telegram, nil
}
