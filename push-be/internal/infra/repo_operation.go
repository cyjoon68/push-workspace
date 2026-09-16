package infra

import (
	"context"

	"github.com/google/uuid"

	"push-be/internal/domain"
)

type OperationRepo struct{ d *DB }

func NewOperationRepo(d *DB) *OperationRepo { return &OperationRepo{d: d} }

func scanOperation(row interface{ Scan(...any) error }) (*domain.Operation, error) {
	o := &domain.Operation{}
	var result, oerr, inputReq, pending []byte
	err := row.Scan(&o.ID, &o.UserID, &o.Type, &o.ApplicationID, &o.Status, &o.Progress,
		&result, &oerr, &inputReq, &pending, &o.CreatedAt, &o.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if len(result) > 0 {
		res := &domain.OperationResult{}
		if err := unj(result, res); err != nil {
			return nil, err
		}
		o.Result = res
	}
	if len(oerr) > 0 {
		e := &domain.OperationError{}
		if err := unj(oerr, e); err != nil {
			return nil, err
		}
		o.Error = e
	}
	if len(inputReq) > 0 {
		ir := &domain.InputRequest{}
		if err := unj(inputReq, ir); err != nil {
			return nil, err
		}
		o.InputRequest = ir
	}
	o.PendingPayload = pending
	return o, nil
}

const opCols = `id, user_id, type, application_id, status, progress, result, error,
	input_request, pending_payload, created_at, updated_at`

func (r *OperationRepo) Create(ctx context.Context, o *domain.Operation) error {
	result, _ := jv(o.Result)
	oerr, _ := jv(o.Error)
	inputReq, _ := jv(o.InputRequest)
	_, err := r.d.Q(ctx).Exec(ctx,
		`INSERT INTO operations (`+opCols+`) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		o.ID, o.UserID, o.Type, o.ApplicationID, o.Status, o.Progress,
		nilIfEmpty(result), nilIfEmpty(oerr), nilIfEmpty(inputReq), o.PendingPayload,
		o.CreatedAt, o.UpdatedAt)
	return err
}

func nilIfEmpty(b []byte) any {
	if len(b) == 0 || string(b) == "null" || string(b) == "[]" {
		return nil
	}
	return b
}

func (r *OperationRepo) Get(ctx context.Context, userID, id uuid.UUID) (*domain.Operation, error) {
	o, err := scanOperation(r.d.Q(ctx).QueryRow(ctx,
		`SELECT `+opCols+` FROM operations WHERE id = $1 AND user_id = $2`, id, userID))
	if err != nil {
		return nil, mapGetErr(err)
	}
	return o, nil
}

func (r *OperationRepo) Update(ctx context.Context, o *domain.Operation) error {
	result, _ := jv(o.Result)
	oerr, _ := jv(o.Error)
	inputReq, _ := jv(o.InputRequest)
	_, err := r.d.Q(ctx).Exec(ctx,
		`UPDATE operations SET status = $2, progress = $3, result = $4, error = $5,
		 input_request = $6, pending_payload = $7, updated_at = $8 WHERE id = $1`,
		o.ID, o.Status, o.Progress, nilIfEmpty(result), nilIfEmpty(oerr),
		nilIfEmpty(inputReq), o.PendingPayload, o.UpdatedAt)
	return err
}
