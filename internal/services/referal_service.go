package services

import (
	"tonclient/internal/models"
	"tonclient/internal/repositories"
)

type ReferalService struct {
	repo *repositories.ReferralRepository
}

func NewReferalService(repo *repositories.ReferralRepository) *ReferalService {
	return &ReferalService{
		repo: repo,
	}
}

func (s *ReferalService) Save(ref *models.Referral) error {
	return s.repo.Save(ref)
}
