package services

import (
	"errors"
	"tonclient/internal/models"
	"tonclient/internal/repositories"
)

type PoolService struct {
	poolRepository    *repositories.PoolRepository
	tonConnectService *TonConnectService
	UserService       *UserService
}

func NewPoolService(
	poolRepository *repositories.PoolRepository,
	userService UserService,
) *PoolService {

	return &PoolService{
		poolRepository: poolRepository,
		UserService:    &userService,
	}
}

func (s *PoolService) CreatePool(ownerId uint64, reserve float64, jettonWallet string, reward uint, period uint) (*models.Pool, error) {
	user, err := s.UserService.GetById(ownerId)
	if user == nil {
		return nil, err
	}

	if reserve < 1 {
		return nil, errors.New("reserve must be greater than zero")
	}

	if jettonWallet == "" {
		return nil, errors.New("jettonWallet must be set")
	}

	if reward < 1 {
		return nil, errors.New("reward must be greater than zero")
	}

	if reward > 30 {
		return nil, errors.New("reward must be less than 30")
	}

	if period < 1 {
		return nil, errors.New("period must be greater than zero")
	}

	pool := &models.Pool{
		OwnerId:      ownerId,
		Reserve:      reserve,
		JettonWallet: jettonWallet,
		Reward:       reward,
		Period:       period,
		IsActive:     false,
	}
	if err := s.poolRepository.Save(pool); err != nil {
		return nil, err
	}

	return pool, nil
}

func (s *PoolService) SetActive(poolId uint64, b bool) error {
	pool := s.poolRepository.FindById(poolId)
	if pool == nil {
		return errors.New("pool not found")
	}

	pool.IsActive = b
	return s.poolRepository.Update(pool)
}

func (s *PoolService) AddReserve(poolId uint64, reserve float64) (newReserve float64, err error) {
	pool := s.poolRepository.FindById(poolId)
	if pool == nil {
		return 0, errors.New("pool not found")
	}

	currentReserve := pool.Reserve
	pool.Reserve = reserve + currentReserve
	if err = s.poolRepository.Update(pool); err != nil {
		return 0, err
	}

	return currentReserve, nil
}

func (s *PoolService) Delete(poolId uint64) error {
	return s.poolRepository.DeleteById(poolId)
}

func (s *PoolService) All() *[]models.Pool {
	return s.poolRepository.FindAll()
}

func (s *PoolService) AllLimit(offset, limit int) *[]models.Pool {
	return s.poolRepository.FindAllLimit(offset, limit)
}

func (s *PoolService) GetId(poolId uint64) (*models.Pool, error) {
	pool := s.poolRepository.FindById(poolId)
	if pool == nil {
		return nil, errors.New("pool not found")
	}
	return pool, nil
}
