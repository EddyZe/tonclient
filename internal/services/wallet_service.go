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

func (s *WalletTonService) EditWallet(walletId uint64, newWalletAddr, newWalletName string) (*models.WalletTon, error) {
	w := s.walletRep.FindById(walletId)
	if w == nil {
		return nil, errors.New("wallet not found")
	}

	w.Name = newWalletName
	w.Addr = newWalletAddr
	if err := s.walletRep.Update(w); err != nil {
		return nil, err
	}

	return w, nil
}

func (s *WalletTonService) DeleteWallet(walletId uint64) error {
	w := s.walletRep.FindById(walletId)
	if w == nil {
		return errors.New("wallet not found")
	}

	return s.walletRep.DeleteById(walletId)
}

func (s *WalletTonService) FindWalletByAddr(addr string) (*models.WalletTon, error) {
	w := s.walletRep.FindByAddr(addr)
	if w == nil {
		return nil, errors.New("wallet not found")
	}

	return w, nil
}

func (s *WalletTonService) GetById(id uint64) (*models.WalletTon, error) {
	w := s.walletRep.FindById(id)
	if w == nil {
		return nil, errors.New("wallet not found")
	}

	return w, nil
}
