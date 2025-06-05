package services

import (
	"time"
	"tonclient/internal/models"
	"tonclient/internal/repositories"
)

type OperationService struct {
	rep *repositories.OperationRepository
}

func NewOperationService(rep *repositories.OperationRepository) *OperationService {
	return &OperationService{rep}
}

func (s *OperationService) Create(userId uint64, numOperation int, description string) (*models.Operation, error) {
	opName := OperationName(numOperation)
	op := models.Operation{
		UserId:       userId,
		NumOperation: numOperation,
		Name:         opName,
		CreatedAt:    time.Now(),
		Description:  description,
	}

	if err := s.rep.Save(&op); err != nil {
		return nil, err
	}
	return &op, nil
}

func (s *OperationService) GetById(id uint64) (*models.Operation, error) {
	return s.rep.FindById(id)
}

func (s *OperationService) GetAll() ([]models.Operation, error) {
	return s.rep.FindAll()
}

func (s *OperationService) GetAllLimit(offset, limit int) ([]models.Operation, error) {
	return s.rep.FindAllLimit(offset, limit)
}

func (s *OperationService) GetByUserId(userId uint64) ([]models.Operation, error) {
	return s.rep.FindByUserId(userId)
}

func (s *OperationService) GetByUserIdLimit(userId uint64, offset, limit int) ([]models.Operation, error) {
	return s.rep.FindByUserIdLimit(userId, offset, limit)
}

func (s *OperationService) Count() int {
	return s.rep.CountAll()
}

func (s *OperationService) CountByUserId(userId uint64) int {
	return s.rep.CountByUserId(userId)
}

func OperationName(numOperation int) string {
	switch numOperation {
	case models.OP_ADMIN_CREATE_POOL:
		return "Создание пула"
	case models.OP_ADMIN_ADD_RESERVE:
		return "Пополнение резерва"
	case models.OP_PAY_COMMISION:
		return "Оплата комиссии за пул"
	case models.OP_STAKE:
		return "Создание стейка"
	case models.OP_ADMIN_CLOSE_POOL:
		return "Закрытие пула"
	case models.OP_CLAIM_INSURANCE:
		return "Получение страховки"
	case models.OP_CLAIM:
		return "Получение награды"
	case models.OP_ADMIN_OPEN_POOL:
		return "Открытие пула"
	case models.OP_RETURNING_TOKENS:
		return "Возврат токенов"
	case models.OP_CLAIM_RESERVE:
		return "Снятие резерва"
	case models.OP_PAID_COMMISSION_STAKE:
		return "Оплата комиссии за стейк"
	case models.OP_EARLY_CLOSOURE:
		return "Досрочное закрытие стейка"
	case models.OP_DELETE_POOL:
		return "Удаление пула"
	default:
		return "Неизвестная команда"
	}
}
