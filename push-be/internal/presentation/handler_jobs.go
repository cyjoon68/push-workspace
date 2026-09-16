package presentation

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"push-be/internal/domain"
	"push-be/internal/domain/usecase"
)

func jobDTO(j *domain.JobPosting) map[string]any {
	var deadline any
	if j.Deadline != nil {
		deadline = j.Deadline.Format("2006-01-02")
	}
	return map[string]any{
		"id": j.ID, "revision": j.Revision, "company": j.Company, "title": j.Title,
		"sourceKind": j.SourceKind, "sourceUrl": j.SourceURL, "sourceText": j.SourceText,
		"requirements": j.Requirements, "preferred": j.Preferred, "keywords": j.Keywords,
		"risks": j.Risks, "deadline": deadline, "language": j.Language,
		"createdAt": j.CreatedAt.UTC().Format(time.RFC3339),
		"updatedAt": j.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func (h *Handler) listJobs(c echo.Context) error {
	page, err := pageRequest(c)
	if err != nil {
		return err
	}
	f := domain.JobFilter{Page: page, Query: c.QueryParam("query")}
	if a := c.QueryParam("archived"); a != "" {
		v := a == "true"
		f.Archived = &v
	}
	p, err := h.jobs.List(c.Request().Context(), userID(c), f)
	if err != nil {
		return err
	}
	items := make([]any, len(p.Items))
	for i := range p.Items {
		items[i] = jobDTO(&p.Items[i])
	}
	return c.JSON(http.StatusOK, map[string]any{
		"data": items,
		"page": map[string]any{"nextCursor": p.NextCursor, "hasMore": p.HasMore},
	})
}

func (h *Handler) createJob(c echo.Context) error {
	var req createJobReq
	if err := bindJSON(c, &req); err != nil {
		return err
	}
	deadline, _, err := optionalDate(req.Deadline)
	if err != nil {
		return err
	}
	j, err := h.jobs.Create(c.Request().Context(), userID(c), usecase.CreateJobInput{
		Company: req.Company, Title: req.Title, SourceKind: domain.JobSourceKind(req.SourceKind),
		SourceURL: req.SourceURL, SourceText: req.SourceText,
		Requirements: req.Requirements, Preferred: req.Preferred,
		Deadline: deadline, Language: req.Language,
	})
	if err != nil {
		return err
	}
	return data(c, http.StatusCreated, jobDTO(j))
}

func (h *Handler) getJob(c echo.Context) error {
	id, err := paramID(c, "id")
	if err != nil {
		return err
	}
	j, err := h.jobs.Get(c.Request().Context(), userID(c), id)
	if err != nil {
		return err
	}
	return data(c, http.StatusOK, jobDTO(j))
}

func (h *Handler) patchJob(c echo.Context) error {
	id, err := paramID(c, "id")
	if err != nil {
		return err
	}
	var req patchJobReq
	if err := bindJSON(c, &req); err != nil {
		return err
	}
	deadline, deadlineSet, err := optionalDate(req.Deadline)
	if err != nil {
		return err
	}
	in := usecase.PatchJobInput{
		ExpectedRevision: req.ExpectedRevision, Company: req.Company, Title: req.Title,
		Requirements: req.Requirements, Preferred: req.Preferred,
	}
	if deadlineSet {
		if deadline == nil {
			in.ClearDeadline = true
		} else {
			in.Deadline = deadline
		}
	}
	j, err := h.jobs.Patch(c.Request().Context(), userID(c), id, in)
	if err != nil {
		return err
	}
	return data(c, http.StatusOK, jobDTO(j))
}

func (h *Handler) analyzeJob(c echo.Context) error {
	id, err := paramID(c, "id")
	if err != nil {
		return err
	}
	var req analyzeReq
	if err := bindJSON(c, &req); err != nil {
		return err
	}
	op, err := h.jobs.Analyze(c.Request().Context(), userID(c), id, usecase.AnalyzeInput{
		ApplicationID: req.ApplicationID, ExpectedRevision: req.ExpectedRevision,
		EvidenceIDs: req.EvidenceIDs, AI: req.AI,
	})
	if err != nil {
		return err
	}
	return data(c, http.StatusAccepted, op)
}

func (h *Handler) listAnalyses(c echo.Context) error {
	id, err := paramID(c, "id")
	if err != nil {
		return err
	}
	page, err := pageRequest(c)
	if err != nil {
		return err
	}
	p, err := h.jobs.ListAnalyses(c.Request().Context(), userID(c), id, page)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, pageBody(p))
}
