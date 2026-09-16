package presentation

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"push-be/internal/domain"
	"push-be/internal/domain/usecase"
)

func (h *Handler) listApprovals(c echo.Context) error {
	page, err := pageRequest(c)
	if err != nil {
		return err
	}
	f := domain.ApprovalFilter{Page: page}
	if a := c.QueryParam("applicationId"); a != "" {
		id, err := uuid.Parse(a)
		if err != nil {
			return domain.ValidationField("applicationId", "must be a UUID")
		}
		f.ApplicationID = &id
	}
	if s := c.QueryParam("status"); s != "" {
		st := domain.ApprovalStatus(s)
		switch st {
		case domain.ApprovalPending, domain.ApprovalApproved, domain.ApprovalDenied,
			domain.ApprovalExpired, domain.ApprovalConsumed:
		default:
			return domain.ValidationField("status", "unsupported status")
		}
		f.Status = &st
	}
	p, err := h.approvals.List(c.Request().Context(), userID(c), f)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, pageBody(p))
}

func (h *Handler) createApproval(c echo.Context) error {
	var req createApprovalReq
	if err := bindJSON(c, &req); err != nil {
		return err
	}
	a, err := h.approvals.Create(c.Request().Context(), userID(c), usecase.CreateApprovalInput{
		Kind: domain.ApprovalKind(req.Kind), ApplicationID: req.ApplicationID,
		TargetID: req.TargetID, TargetRevision: req.TargetRevision,
	})
	if err != nil {
		return err
	}
	return data(c, http.StatusCreated, a)
}

func (h *Handler) getApproval(c echo.Context) error {
	id, err := paramID(c, "id")
	if err != nil {
		return err
	}
	a, summary, err := h.approvals.Get(c.Request().Context(), userID(c), id)
	if err != nil {
		return err
	}
	return data(c, http.StatusOK, map[string]any{"approval": a, "target": summary})
}

func (h *Handler) decideApproval(c echo.Context) error {
	id, err := paramID(c, "id")
	if err != nil {
		return err
	}
	var req decisionReq
	if err := bindJSON(c, &req); err != nil {
		return err
	}
	a, err := h.approvals.Decide(c.Request().Context(), userID(c), id,
		req.ExpectedRevision, domain.ApprovalStatus(req.Decision))
	if err != nil {
		return err
	}
	return data(c, http.StatusOK, a)
}
