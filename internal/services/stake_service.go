package services

import (
	"errors"
	"tonclient/internal/models"
	"tonclient/internal/repositories"
)

type StakeService struct {
	stakeRepo   *repositories.StakeRepository
	userService *UserService
	poolService *PoolService
}

func NewStakeService(
	stakeRepo *repositories.StakeRepository,
	userService *UserService,
	poolService *PoolService) *StakeService {
	return &StakeService{
		stakeRepo:   stakeRepo,
		userService: userService,
		poolService: poolService,
	}
}

func (s *StakeService) CreateStake(userId, poolId uint64, amount float64) (*models.Stake, error) {
	if _, err := s.userService.GetById(userId); err != nil {
		return nil, err
	}
	if _, err := s.poolService.GetId(poolId); err != nil {
		return nil, err
	}
	if amount < 1 {
		return nil, errors.New("amount must be greater than 0")
	}

	stake := &models.Stake{
		UserId:   userId,
		PoolId:   poolId,
		Amount:   amount,
		IsActive: true,
	}

	if err := s.stakeRepo.Save(stake); err != nil {
		return nil, err
	}

	return stake, nil
}
