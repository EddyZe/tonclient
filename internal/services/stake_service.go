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

func (s *StakeService) GetStakesPoolIdAndStatus(poolId uint64, b bool) *[]models.Stake {
	return s.stakeRepo.GetStakesPoolIdAndStatus(poolId, b)
}

func (s *StakeService) Update(stake *models.Stake) error {
	return s.stakeRepo.Update(stake)
}

func (s *StakeService) CountStakesPoolIdAndStatus(poolId uint64, b bool) int {
	return s.stakeRepo.CountStakesPoolIdAndStatus(poolId, b)
}

func (s *StakeService) GetStakesUser(userid uint64) *[]models.Stake {
	stakes := s.stakeRepo.GetUserStakes(userid)
	if stakes == nil {
		return &[]models.Stake{}
	}

	return stakes
}

func (s *StakeService) GetAllIsStatus(b bool) *[]models.Stake {
	return s.stakeRepo.FindAllByStatus(b)
}

func (s *StakeService) GroupFromPoolByUserId(userId uint64) *[]models.GroupElements {
	return s.stakeRepo.GroupFromPoolNameByUserId(userId)
}

func (s *StakeService) GroupFromPoolByUserIdLimit(userId uint64, limit, offset int) *[]models.GroupElements {
	return s.stakeRepo.GroupFromPoolNameByUserIdLimit(userId, offset, limit)
}

func (s *StakeService) GroupFromPoolByUserIdLimitIsInsurancePaid(userId uint64, limit, offset int, b bool, isActive bool) *[]models.GroupElements {
	return s.stakeRepo.GroupFromPoolNameByUserIdLimitIsInsurancePaid(userId, offset, limit, b, isActive)
}

func (s *StakeService) GroupFromPoolByUserIdLimitIsProfitPaid(userId uint64, limit, offset int, b bool, isAcive bool) *[]models.GroupElements {
	return s.stakeRepo.GroupFromPoolNameByUserIdLimitIsProfitPaid(userId, offset, limit, b, isAcive)
}

func (s *StakeService) GetByJettonNameAndUserId(userId uint64, jettonName string) *[]models.Stake {
	return s.stakeRepo.FindByJettonNameAndUserId(userId, jettonName)
}

func (s *StakeService) GetByJettonNameAndUserIdLimit(userId uint64, jettonName string, offset, limit int) *[]models.Stake {
	return s.stakeRepo.FindByJettonNameAndUserIdLimit(userId, jettonName, offset, limit)
}

func (s *StakeService) GetByJettonNameAndUserIdLimitIsInsurancePaid(userId uint64, jettonName string, offset, limit int, b bool, isActive bool) *[]models.Stake {
	return s.stakeRepo.FindByJettonNameAndUserIdLimitIsInsurancePaid(userId, jettonName, offset, limit, b, isActive)
}

func (s *StakeService) GetByJettonNameAndUserIdLimitIsProfitPaid(userId uint64, jettonName string, offset, limit int, b bool, isActive bool) *[]models.Stake {
	return s.stakeRepo.FindByJettonNameAndUserIdLimitIsProfitPaid(userId, jettonName, offset, limit, b, isActive)
}

func (s *StakeService) CountGroupsStakesUserId(userId uint64) int {
	return s.stakeRepo.CountGroupsStakesUserId(userId)
}

func (s *StakeService) CountGroupsStakesUserIdIsInsurancePaid(userId uint64, b bool, isActive bool) int {
	return s.stakeRepo.CountGroupsStakesUserIdIsInsurancePaid(userId, b, isActive)
}

func (s *StakeService) CountGroupsStakesUserIdProfitPaid(userId uint64, b, isActive bool) int {
	return s.stakeRepo.CountGroupsStakesUserIdIsProfitPaid(userId, b, isActive)
}

func (s *StakeService) CountGroupsStakesByUserIdAndJettonName(userId uint64, jettonName string) int {
	return s.stakeRepo.CountGroupsStakesByUserIdAndJettonName(userId, jettonName)
}

func (s *StakeService) CountGroupsStakesByUserIdAndJettonNameIsInsurancePaid(userId uint64, jettonName string, b, isActive bool) int {
	return s.stakeRepo.CountGroupsStakesByUserIdAndJettonNameIsInsurancePaid(userId, jettonName, b, isActive)
}

func (s *StakeService) CountGroupsStakesByUserIdAndJettonNameIsProfitPaid(userId uint64, jettonName string, b, isActive bool) int {
	return s.stakeRepo.CountGroupsStakesByUserIdAndJettonNameIsProfitPaid(userId, jettonName, b, isActive)
}

func (s *StakeService) GetById(stakeId uint64) (*models.Stake, error) {
	stake := s.stakeRepo.GetById(stakeId)
	if stake == nil {
		return nil, errors.New("stake not found")
	}

	return stake, nil
}

func (s *StakeService) SetCommissionPaid(stakeId uint64, isPaid bool) error {
	return s.stakeRepo.SetCommissionPaid(stakeId, isPaid)
}
