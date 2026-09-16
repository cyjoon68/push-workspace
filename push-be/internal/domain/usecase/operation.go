package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"push-be/internal/domain"
)

type OperationService struct {
	ops      domain.OperationStore
	uow      domain.UnitOfWork
	evidence *EvidenceService
}

func NewOperationService(ops domain.OperationStore, uow domain.UnitOfWork, evidence *EvidenceService) *OperationService {
	return &OperationService{ops: ops, uow: uow, evidence: evidence}
}

func (s *OperationService) Get(ctx context.Context, userID, id uuid.UUID) (*domain.Operation, error) {
	op, err := s.ops.Get(ctx, userID, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.NotFound()
		}
		return nil, domain.Internal()
	}
	return op, nil
}

func (s *OperationService) SubmitInput(ctx context.Context, userID, id uuid.UUID, fields map[string]string, sourceID *uuid.UUID) (*domain.Operation, error) {
	var out *domain.Operation
	err := s.uow.Do(ctx, func(ctx context.Context) error {
		op, err := s.ops.Get(ctx, userID, id)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return domain.NotFound()
			}
			return err
		}
		if domain.TerminalOperationStatus(op.Status) {
			return domain.InvalidTransition("operation already finished")
		}
		if op.Status != domain.OpNeedsInput || op.InputRequest == nil {
			return domain.InvalidTransition("operation does not await input")
		}
		switch op.Type {
		case domain.OpEvidenceImport:
			if err := s.evidence.CompleteImportInput(ctx, op, fields); err != nil {
				return err
			}
		default:
			return domain.InvalidTransition("operation does not accept input")
		}
		op.InputRequest = nil
		op.UpdatedAt = time.Now().UTC()
		if op.Status == domain.OpNeedsInput {
			op.Status = domain.OpQueued
		}
		if err := s.ops.Update(ctx, op); err != nil {
			return err
		}
		out = op
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (s *OperationService) Cancel(ctx context.Context, userID, id uuid.UUID) (*domain.Operation, error) {
	var out *domain.Operation
	err := s.uow.Do(ctx, func(ctx context.Context) error {
		op, err := s.ops.Get(ctx, userID, id)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return domain.NotFound()
			}
			return err
		}
		if domain.TerminalOperationStatus(op.Status) {
			out = op
			return nil
		}
		op.Status = domain.OpCancelled
		op.UpdatedAt = time.Now().UTC()
		if err := s.ops.Update(ctx, op); err != nil {
			return err
		}
		out = op
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
