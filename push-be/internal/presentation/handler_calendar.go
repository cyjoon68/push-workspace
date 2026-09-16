package presentation

import (
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"push-be/internal/domain"
	"push-be/internal/domain/usecase"
)

func (h *Handler) listCalendarEvents(c echo.Context) error {
	page, err := pageRequest(c)
	if err != nil {
		return err
	}
	f := domain.CalendarEventFilter{Page: page}
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
	p, err := h.calendar.List(c.Request().Context(), userID(c), f)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, pageBody(p))
}

func (h *Handler) createCalendarEvent(c echo.Context) error {
	var req createCalendarEventReq
	if err := bindJSON(c, &req); err != nil {
		return err
	}
	e, err := h.calendar.Create(c.Request().Context(), userID(c), usecase.CreateEventInput{
		ApplicationID: req.ApplicationID, Type: domain.CalendarEventType(req.Type),
		Title: req.Title, StartsAt: req.StartsAt, EndsAt: req.EndsAt,
		TimeZone: req.TimeZone, Notes: req.Notes,
	})
	if err != nil {
		return err
	}
	return data(c, http.StatusCreated, e)
}

func (h *Handler) patchCalendarEvent(c echo.Context) error {
	id, err := paramID(c, "id")
	if err != nil {
		return err
	}
	var req patchCalendarEventReq
	if err := bindJSON(c, &req); err != nil {
		return err
	}
	e, err := h.calendar.Patch(c.Request().Context(), userID(c), id, usecase.PatchEventInput{
		ExpectedRevision: req.ExpectedRevision, Title: req.Title,
		StartsAt: req.StartsAt, EndsAt: req.EndsAt,
		TimeZone: req.TimeZone, Notes: req.Notes,
	})
	if err != nil {
		return err
	}
	return data(c, http.StatusOK, e)
}

func (h *Handler) deleteCalendarEvent(c echo.Context) error {
	id, err := paramID(c, "id")
	if err != nil {
		return err
	}
	m := c.Request().Header.Get("If-Match")
	rev, err := strconv.ParseInt(m, 10, 64)
	if m == "" || err != nil || rev < 1 {
		return domain.ValidationField("If-Match", "revision header required")
	}
	if err := h.calendar.Delete(c.Request().Context(), userID(c), id, rev); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}
