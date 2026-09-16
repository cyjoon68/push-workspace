package presentation

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"push-be/internal/domain"
	"push-be/internal/domain/usecase"
)

func (h *Handler) listProjects(c echo.Context) error {
	page, err := pageRequest(c)
	if err != nil {
		return err
	}
	f := domain.ProjectFilter{Page: page}
	if a := c.QueryParam("applicationId"); a != "" {
		id, err := uuid.Parse(a)
		if err != nil {
			return domain.ValidationField("applicationId", "must be a UUID")
		}
		f.ApplicationID = &id
	}
	p, err := h.projects.ListBlueprints(c.Request().Context(), userID(c), f)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, pageBody(p))
}

func (h *Handler) generateBlueprints(c echo.Context) error {
	var req blueprintsReq
	if err := bindJSON(c, &req); err != nil {
		return err
	}
	op, err := h.projects.GenerateBlueprints(c.Request().Context(), userID(c),
		req.ApplicationID, req.GapAnalysisID, req.AI)
	if err != nil {
		return err
	}
	return data(c, http.StatusAccepted, op)
}

func (h *Handler) getProject(c echo.Context) error {
	id, err := paramID(c, "id")
	if err != nil {
		return err
	}
	b, err := h.projects.GetBlueprint(c.Request().Context(), userID(c), id)
	if err != nil {
		return err
	}
	return data(c, http.StatusOK, b)
}

func (h *Handler) selectProject(c echo.Context) error {
	id, err := paramID(c, "id")
	if err != nil {
		return err
	}
	var req expectedRevisionReq
	if err := bindJSON(c, &req); err != nil {
		return err
	}
	b, manifest, err := h.projects.Select(c.Request().Context(), userID(c), id, req.ExpectedRevision)
	if err != nil {
		return err
	}
	return data(c, http.StatusOK, map[string]any{"blueprint": b, "manifest": manifest})
}

func (h *Handler) listRuns(c echo.Context) error {
	id, err := paramID(c, "id")
	if err != nil {
		return err
	}
	page, err := pageRequest(c)
	if err != nil {
		return err
	}
	p, err := h.projects.ListRuns(c.Request().Context(), userID(c), id, page)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, pageBody(p))
}

func (h *Handler) createRun(c echo.Context) error {
	id, err := paramID(c, "id")
	if err != nil {
		return err
	}
	var req createRunReq
	if err := bindJSON(c, &req); err != nil {
		return err
	}
	r, err := h.projects.CreateRun(c.Request().Context(), userID(c), id,
		domain.CliProvider(req.Provider), req.WorkingDirectory, req.Prompt)
	if err != nil {
		return err
	}
	return data(c, http.StatusCreated, r)
}

func (h *Handler) runIDs(c echo.Context) (uuid.UUID, uuid.UUID, error) {
	pid, err := paramID(c, "id")
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	rid, err := paramID(c, "runId")
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	return pid, rid, nil
}

func (h *Handler) startRun(c echo.Context) error {
	pid, rid, err := h.runIDs(c)
	if err != nil {
		return err
	}
	var req startRunReq
	if err := bindJSON(c, &req); err != nil {
		return err
	}
	r, err := h.projects.StartRun(c.Request().Context(), userID(c), pid, rid, usecase.StartRunInput{
		ExpectedRevision: req.ExpectedRevision, ApprovalID: req.ApprovalID,
		DetectedVersion: req.DetectedVersion, DeviceID: req.DeviceID,
	})
	if err != nil {
		return err
	}
	return data(c, http.StatusOK, r)
}

func (h *Handler) launchRun(c echo.Context) error {
	pid, rid, err := h.runIDs(c)
	if err != nil {
		return err
	}
	var req launchReq
	if err := bindJSON(c, &req); err != nil {
		return err
	}
	r, err := h.projects.ReportLaunch(c.Request().Context(), userID(c), pid, rid, usecase.LaunchInput{
		ExpectedRevision: req.ExpectedRevision, DeviceID: req.DeviceID,
		PayloadHash: req.PayloadHash, LaunchStatus: domain.LaunchStatus(req.LaunchStatus),
		Process: req.Process,
	})
	if err != nil {
		return err
	}
	return data(c, http.StatusOK, r)
}

func (h *Handler) recoverRun(c echo.Context) error {
	pid, rid, err := h.runIDs(c)
	if err != nil {
		return err
	}
	var req recoverReq
	if err := bindJSON(c, &req); err != nil {
		return err
	}
	r, err := h.projects.Recover(c.Request().Context(), userID(c), pid, rid, usecase.RecoverInput{
		ExpectedRevision: req.ExpectedRevision, DeviceID: req.DeviceID,
		Decision: req.Decision, Process: req.Process, FailureReason: req.FailureReason,
	})
	if err != nil {
		return err
	}
	return data(c, http.StatusOK, r)
}

func (h *Handler) resultRun(c echo.Context) error {
	pid, rid, err := h.runIDs(c)
	if err != nil {
		return err
	}
	var req runResultReq
	if err := bindJSON(c, &req); err != nil {
		return err
	}
	r, err := h.projects.ReportResult(c.Request().Context(), userID(c), pid, rid, usecase.RunResultInput{
		ExpectedRevision: req.ExpectedRevision, ExitCode: req.ExitCode,
		CommitSHA: req.CommitSHA, StdoutHash: req.StdoutHash, StderrHash: req.StderrHash,
	})
	if err != nil {
		return err
	}
	return data(c, http.StatusOK, r)
}

func (h *Handler) createProjectEvidence(c echo.Context) error {
	id, err := paramID(c, "id")
	if err != nil {
		return err
	}
	var req createProjectEvidenceReq
	if err := bindJSON(c, &req); err != nil {
		return err
	}
	e, err := h.projects.CreateEvidence(c.Request().Context(), userID(c), id, usecase.CreateProjectEvidenceInput{
		RunID: req.RunID, CommitURL: req.CommitURL, TestCommand: req.TestCommand,
		TestOutput: req.TestOutput, ExitCode: req.ExitCode, Metrics: req.Metrics,
		Summary: req.Summary,
	})
	if err != nil {
		return err
	}
	return data(c, http.StatusCreated, e)
}

func (h *Handler) verifyProjectEvidence(c echo.Context) error {
	pid, err := paramID(c, "id")
	if err != nil {
		return err
	}
	eid, err := paramID(c, "evidenceId")
	if err != nil {
		return err
	}
	var req expectedRevisionReq
	if err := bindJSON(c, &req); err != nil {
		return err
	}
	op, err := h.projects.VerifyEvidence(c.Request().Context(), userID(c), pid, eid, req.ExpectedRevision)
	if err != nil {
		return err
	}
	return data(c, http.StatusAccepted, op)
}

func (h *Handler) listProjectEvidence(c echo.Context) error {
	id, err := paramID(c, "id")
	if err != nil {
		return err
	}
	page, err := pageRequest(c)
	if err != nil {
		return err
	}
	p, err := h.projects.ListEvidence(c.Request().Context(), userID(c), id, page)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, pageBody(p))
}
