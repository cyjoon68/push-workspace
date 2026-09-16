package presentation

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"push-be/internal/domain"
	"push-be/internal/domain/usecase"
)

func (h *Handler) listInterviews(c echo.Context) error {
	page, err := pageRequest(c)
	if err != nil {
		return err
	}
	f := domain.InterviewFilter{Page: page}
	if a := c.QueryParam("applicationId"); a != "" {
		id, err := uuid.Parse(a)
		if err != nil {
			return domain.ValidationField("applicationId", "must be a UUID")
		}
		f.ApplicationID = &id
	}
	for _, p := range []struct {
		name string
		out  **time.Time
	}{{"from", &f.From}, {"to", &f.To}} {
		if v := c.QueryParam(p.name); v != "" {
			t, err := time.Parse(time.RFC3339, v)
			if err != nil {
				return domain.ValidationField(p.name, "must be RFC3339")
			}
			*p.out = &t
		}
	}
	p, err := h.interviews.List(c.Request().Context(), userID(c), f)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, pageBody(p))
}

func (h *Handler) createInterview(c echo.Context) error {
	var req createInterviewReq
	if err := bindJSON(c, &req); err != nil {
		return err
	}
	v, err := h.interviews.Create(c.Request().Context(), userID(c), usecase.CreateInterviewInput{
		ApplicationID: req.ApplicationID, Title: req.Title, ScheduledAt: req.ScheduledAt,
		DurationMinutes: req.DurationMinutes, EvidenceIDs: req.EvidenceIDs,
		CompanySources: req.CompanySources, Notes: req.Notes, TimeZone: req.TimeZone,
	})
	if err != nil {
		return err
	}
	return data(c, http.StatusCreated, v)
}

func (h *Handler) getInterview(c echo.Context) error {
	id, err := paramID(c, "id")
	if err != nil {
		return err
	}
	v, err := h.interviews.Get(c.Request().Context(), userID(c), id)
	if err != nil {
		return err
	}
	return data(c, http.StatusOK, v)
}

func (h *Handler) patchInterview(c echo.Context) error {
	id, err := paramID(c, "id")
	if err != nil {
		return err
	}
	var req patchInterviewReq
	if err := bindJSON(c, &req); err != nil {
		return err
	}
	v, err := h.interviews.Patch(c.Request().Context(), userID(c), id, usecase.PatchInterviewInput{
		ExpectedRevision: req.ExpectedRevision, Title: req.Title,
		ScheduledAt: req.ScheduledAt, CompanySources: req.CompanySources,
		Notes: req.Notes, Reflection: req.Reflection,
	})
	if err != nil {
		return err
	}
	return data(c, http.StatusOK, v)
}

func (h *Handler) prepareInterview(c echo.Context) error {
	id, err := paramID(c, "id")
	if err != nil {
		return err
	}
	var req prepareReq
	if err := bindJSON(c, &req); err != nil {
		return err
	}
	op, err := h.interviews.Prepare(c.Request().Context(), userID(c), id, req.ExpectedRevision, req.AI)
	if err != nil {
		return err
	}
	return data(c, http.StatusAccepted, op)
}
