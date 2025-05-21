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
	userService *UserService,
) *PoolService {

	return &PoolService{
		poolRepository: poolRepository,
		UserService:    userService,
	}
}

func (s *PoolService) CreatePool(pool *models.Pool) (*models.Pool, error) {
	user, err := s.UserService.GetById(pool.OwnerId)
	if user == nil {
		return nil, err
	}

	if pool.Reserve < 1 {
		return nil, errors.New("reserve must be greater than zero")
	}

	if pool.JettonWallet == "" {
		return nil, errors.New("jettonWallet must be set")
	}

	if pool.Reward < 1 {
		return nil, errors.New("reward must be greater than zero")
	}

	if pool.Period < 1 {
		return nil, errors.New("period must be greater than zero")
	}

	if pool.InsuranceCoating < 1 {
		return nil, errors.New("insurance_coating must be greater than zero")
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

	return pool.Reserve, nil
}

func (s *PoolService) SetCommissionPaid(poolId uint64, b bool) error {
	pool := s.poolRepository.FindById(poolId)
	if pool == nil {
		return errors.New("pool not found")
	}

	pool.IsCommissionPaid = b
	return s.poolRepository.Update(pool)
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

func (s *PoolService) AllByStatus(isActive bool) *[]models.Pool {
	return s.poolRepository.FindAllByStatus(isActive)
}

func (s *PoolService) AllLimitByStatus(isActive bool, offset, limit int) *[]models.Pool {
	return s.poolRepository.FindAllByStatusLimit(isActive, offset, limit)
}

func (s *PoolService) GetId(poolId uint64) (*models.Pool, error) {
	pool := s.poolRepository.FindById(poolId)
	if pool == nil {
		return nil, errors.New("pool not found")
	}
	return pool, nil
}

func (s *PoolService) CountAllByStatus(isActive bool) int {
	return s.poolRepository.CountAllByStatus(isActive)
}

func (s *PoolService) CountAll() int {
	return s.poolRepository.CountAll()
}

func (s *PoolService) CountUserPool(userId uint64) int {
	return s.poolRepository.CountUser(userId)
}

func (s *PoolService) GetPoolsByUserId(userId uint64) *[]models.Pool {
	return s.poolRepository.FindByOwnerId(userId)
}

func (s *PoolService) GetPoolsByUserIdLimit(userId uint64, offset, limit int) *[]models.Pool {
	return s.poolRepository.FindByOwnerIdLimit(userId, offset, limit)
}

func (s *PoolService) Update(pool *models.Pool) error {
	return s.poolRepository.Update(pool)
}
