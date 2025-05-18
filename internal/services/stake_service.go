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

func (s *StakeService) CreateStake(stake *models.Stake) (*models.Stake, error) {
	if _, err := s.userService.GetById(stake.UserId); err != nil {
		return nil, err
	}
	if _, err := s.poolService.GetId(stake.PoolId); err != nil {
		return nil, err
	}
	if stake.Amount < 1 {
		return nil, errors.New("amount must be greater than 0")
	}

	if err := s.stakeRepo.Save(stake); err != nil {
		return nil, err
	}

	return stake, nil
}

func (s *StakeService) CountAll() int {
	return s.stakeRepo.CountAll()
}

func (s *StakeService) CountUser(userId uint64) int {
	return s.stakeRepo.CountUser(userId)
}

func (s *StakeService) CountPool(poolId uint64) int {
	return s.stakeRepo.CountPoolStakes(poolId)
}

func (s *StakeService) CountByUserIdIsActive(userId uint64, b bool) int {
	return s.stakeRepo.CountUserAndStatusStake(userId, b)
}

func (s *StakeService) GetPoolStakes(poolId uint64) *[]models.Stake {
	return s.stakeRepo.FindStakesByPoolId(poolId)
}

func (s *StakeService) GetStakesUserIdStatus(userId uint64, b bool) *[]models.Stake {
	stakes := s.stakeRepo.GetStakeStatusUser(userId, b)
	if stakes == nil {
		return &[]models.Stake{}
	}

	return stakes
}

func (s *StakeService) GetStakesUser(userid uint64) *[]models.Stake {
	stakes := s.stakeRepo.GetUserStakes(userid)
	if stakes == nil {
		return &[]models.Stake{}
	}

	return stakes
}
