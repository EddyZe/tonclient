package services

import (
	"errors"
	"tonclient/internal/models"
	"tonclient/internal/repositories"
)

type WalletTonService struct {
	userService *UserService
	walletRep   *repositories.WalletTonRepository
}

func NewWalletTonService(userService *UserService, walletRepo *repositories.WalletTonRepository) *WalletTonService {
	return &WalletTonService{
		userService: userService,
		walletRep:   walletRepo,
	}
}

func (s *WalletTonService) CreateNewWallet(userId uint64, addr, name string) (*models.WalletTon, error) {
	if _, err := s.userService.GetById(userId); err != nil {
		return nil, err
	}

	w := s.walletRep.FindByAddr(addr)
	if w != nil {
		return nil, errors.New("address already exists")
	}

	w = &models.WalletTon{
		UserId: userId,
		Addr:   addr,
		Name:   name,
	}

	if err := s.walletRep.Save(w); err != nil {
		return nil, err
	}

	return w, nil
}
