package presentation

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"push-be/internal/domain"
	"push-be/internal/domain/usecase"
)

func (h *Handler) listApplications(c echo.Context) error {
	page, err := pageRequest(c)
	if err != nil {
		return err
	}
	f := domain.ApplicationFilter{Page: page, Query: c.QueryParam("query")}
	if s := c.QueryParam("stage"); s != "" {
		stage := domain.ApplicationStage(s)
		if !domain.ValidStage(stage) {
			return domain.ValidationField("stage", "unsupported stage")
		}
		f.Stage = &stage
	}
	p, err := h.applications.List(c.Request().Context(), userID(c), f)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, pageBody(p))
}

func (h *Handler) createApplication(c echo.Context) error {
	var req createApplicationReq
	if err := bindJSON(c, &req); err != nil {
		return err
	}
	a, err := h.applications.Create(c.Request().Context(), userID(c), req.JobID, req.Notes)
	if err != nil {
		return err
	}
	return data(c, http.StatusCreated, a)
}

func (h *Handler) getApplication(c echo.Context) error {
	id, err := paramID(c, "id")
	if err != nil {
		return err
	}
	a, err := h.applications.Get(c.Request().Context(), userID(c), id)
	if err != nil {
		return err
	}
	return data(c, http.StatusOK, a)
}

func optionalTime(raw json.RawMessage, field string) (*time.Time, bool, error) {
	if raw == nil {
		return nil, false, nil
	}
	if string(raw) == "null" {
		return nil, true, nil
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, false, domain.ValidationField(field, "must be RFC3339")
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return nil, false, domain.ValidationField(field, "must be RFC3339")
	}
	return &t, true, nil
}

func (h *Handler) patchApplication(c echo.Context) error {
	id, err := paramID(c, "id")
	if err != nil {
		return err
	}
	var req patchApplicationReq
	if err := bindJSON(c, &req); err != nil {
		return err
	}
	in := usecase.PatchApplicationInput{
		ExpectedRevision: req.ExpectedRevision, Notes: req.Notes,
	}
	if req.Stage != nil {
		stage := domain.ApplicationStage(*req.Stage)
		in.Stage = &stage
	}
	nextAt, nextSet, err := optionalTime(req.NextActionAt, "nextActionAt")
	if err != nil {
		return err
	}
	if nextSet {
		if nextAt == nil {
			in.ClearNextAction = true
		} else {
			in.NextActionAt = nextAt
		}
	}
	a, err := h.applications.Patch(c.Request().Context(), userID(c), id, in)
	if err != nil {
		return err
	}
	return data(c, http.StatusOK, a)
}

func (h *Handler) importApplication(c echo.Context) error {
	var req importApplicationReq
	if err := bindJSON(c, &req); err != nil {
		return err
	}
	a, err := h.applications.Import(c.Request().Context(), userID(c), usecase.ImportApplicationInput{
		JobID: req.JobID, Stage: domain.ApplicationStage(req.Stage),
		AppliedAt: req.AppliedAt, Notes: req.Notes, Confirmed: req.Confirmed,
	})
	if err != nil {
		return err
	}
	return data(c, http.StatusCreated, a)
}

func (h *Handler) applicationTimeline(c echo.Context) error {
	id, err := paramID(c, "id")
	if err != nil {
		return err
	}
	page, err := pageRequest(c)
	if err != nil {
		return err
	}
	p, err := h.applications.Timeline(c.Request().Context(), userID(c), id, page)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, pageBody(p))
}

func (h *Handler) applicationChecklist(c echo.Context) error {
	id, err := paramID(c, "id")
	if err != nil {
		return err
	}
	cl, err := h.applications.Checklist(c.Request().Context(), userID(c), id)
	if err != nil {
		return err
	}
	return data(c, http.StatusOK, cl)
}

func (h *Handler) createSubmissionDraft(c echo.Context) error {
	id, err := paramID(c, "id")
	if err != nil {
		return err
	}
	var req createDraftReq
	if err := bindJSON(c, &req); err != nil {
		return err
	}
	d, err := h.applications.CreateDraft(c.Request().Context(), userID(c), id, usecase.CreateDraftInput{
		ExpectedRevision: req.ExpectedRevision, Mode: domain.SubmissionMode(req.Mode),
		Adapter: req.Adapter, DocumentVersionIDs: req.DocumentVersionIDs,
		ConfirmedSubmitted: req.ConfirmedSubmitted,
	})
	if err != nil {
		return err
	}
	return data(c, http.StatusCreated, d)
}

func (h *Handler) submitApplication(c echo.Context) error {
	id, err := paramID(c, "id")
	if err != nil {
		return err
	}
	var req submitReq
	if err := bindJSON(c, &req); err != nil {
		return err
	}
	sub, err := h.applications.Submit(c.Request().Context(), userID(c), id, usecase.SubmitInput{
		ExpectedRevision: req.ExpectedRevision, DraftID: req.DraftID, ApprovalID: req.ApprovalID,
	})
	if err != nil {
		return err
	}
	return data(c, http.StatusCreated, sub)
}

func (h *Handler) listSubmissions(c echo.Context) error {
	id, err := paramID(c, "id")
	if err != nil {
		return err
	}
	page, err := pageRequest(c)
	if err != nil {
		return err
	}
	p, err := h.applications.ListSubmissions(c.Request().Context(), userID(c), id, page)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, pageBody(p))
}
