package presentation

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"push-be/internal/domain"
	"push-be/internal/domain/usecase"
)

func (h *Handler) listDocuments(c echo.Context) error {
	page, err := pageRequest(c)
	if err != nil {
		return err
	}
	f := domain.DocumentFilter{Page: page}
	if a := c.QueryParam("applicationId"); a != "" {
		id, err := uuid.Parse(a)
		if err != nil {
			return domain.ValidationField("applicationId", "must be a UUID")
		}
		f.ApplicationID = &id
	}
	if k := c.QueryParam("kind"); k != "" {
		kind := domain.DocumentKind(k)
		if !domain.ValidDocumentKind(kind) {
			return domain.ValidationField("kind", "unsupported kind")
		}
		f.Kind = &kind
	}
	p, err := h.documents.List(c.Request().Context(), userID(c), f)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, pageBody(p))
}

func (h *Handler) createDocument(c echo.Context) error {
	var req createDocumentReq
	if err := bindJSON(c, &req); err != nil {
		return err
	}
	d, err := h.documents.Create(c.Request().Context(), userID(c), usecase.CreateDocumentInput{
		ApplicationID: req.ApplicationID, Title: req.Title,
		Kind: domain.DocumentKind(req.Kind), Template: domain.DocumentTemplate(req.Template),
		Language: req.Language,
	})
	if err != nil {
		return err
	}
	return data(c, http.StatusCreated, d)
}

func (h *Handler) getDocument(c echo.Context) error {
	id, err := paramID(c, "id")
	if err != nil {
		return err
	}
	d, err := h.documents.Get(c.Request().Context(), userID(c), id)
	if err != nil {
		return err
	}
	return data(c, http.StatusOK, d)
}

func (h *Handler) patchDocument(c echo.Context) error {
	id, err := paramID(c, "id")
	if err != nil {
		return err
	}
	var req patchDocumentReq
	if err := bindJSON(c, &req); err != nil {
		return err
	}
	in := usecase.PatchDocumentInput{
		ExpectedRevision: req.ExpectedRevision, Title: req.Title, Language: req.Language,
	}
	if req.Template != nil {
		t := domain.DocumentTemplate(*req.Template)
		in.Template = &t
	}
	d, err := h.documents.Patch(c.Request().Context(), userID(c), id, in)
	if err != nil {
		return err
	}
	return data(c, http.StatusOK, d)
}

func (h *Handler) listVersions(c echo.Context) error {
	id, err := paramID(c, "id")
	if err != nil {
		return err
	}
	page, err := pageRequest(c)
	if err != nil {
		return err
	}
	p, err := h.documents.ListVersions(c.Request().Context(), userID(c), id, page)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, pageBody(p))
}

func (h *Handler) getVersion(c echo.Context) error {
	id, err := paramID(c, "id")
	if err != nil {
		return err
	}
	vid, err := paramID(c, "versionId")
	if err != nil {
		return err
	}
	v, err := h.documents.GetVersion(c.Request().Context(), userID(c), id, vid)
	if err != nil {
		return err
	}
	return data(c, http.StatusOK, v)
}

func (h *Handler) createVersion(c echo.Context) error {
	id, err := paramID(c, "id")
	if err != nil {
		return err
	}
	var req createVersionReq
	if err := bindJSON(c, &req); err != nil {
		return err
	}
	blocks := make([]domain.Block, len(req.Blocks))
	for i, b := range req.Blocks {
		blocks[i] = domain.Block{ID: b.ID, Text: b.Text, EvidenceRefs: b.EvidenceRefs}
	}
	d, v, err := h.documents.CreateVersion(c.Request().Context(), userID(c), id, usecase.CreateVersionInput{
		ExpectedRevision: req.ExpectedRevision, Content: req.Content,
		Blocks: blocks, ChangeNote: req.ChangeNote,
	})
	if err != nil {
		return err
	}
	return data(c, http.StatusCreated, map[string]any{"document": d, "version": v})
}

func (h *Handler) generateDocument(c echo.Context) error {
	id, err := paramID(c, "id")
	if err != nil {
		return err
	}
	var req generateDocReq
	if err := bindJSON(c, &req); err != nil {
		return err
	}
	op, err := h.documents.Generate(c.Request().Context(), userID(c), id, usecase.GenerateInput{
		ExpectedRevision: req.ExpectedRevision, EvidenceIDs: req.EvidenceIDs,
		AnalysisID: req.AnalysisID, AI: req.AI, Language: req.Language,
	})
	if err != nil {
		return err
	}
	return data(c, http.StatusAccepted, op)
}

func (h *Handler) createRevision(c echo.Context) error {
	id, err := paramID(c, "id")
	if err != nil {
		return err
	}
	var req reviseReq
	if err := bindJSON(c, &req); err != nil {
		return err
	}
	op, err := h.documents.CreateRevision(c.Request().Context(), userID(c), id, usecase.ReviseInput{
		ExpectedRevision: req.ExpectedRevision, VersionID: req.VersionID,
		Selection: req.Selection, Action: domain.RevisionAction(req.Action),
		Instruction: req.Instruction, AI: req.AI,
	})
	if err != nil {
		return err
	}
	return data(c, http.StatusAccepted, op)
}

func (h *Handler) applyRevision(c echo.Context) error {
	id, err := paramID(c, "id")
	if err != nil {
		return err
	}
	rid, err := paramID(c, "revisionId")
	if err != nil {
		return err
	}
	var req expectedRevisionReq
	if err := bindJSON(c, &req); err != nil {
		return err
	}
	d, v, err := h.documents.ApplyRevision(c.Request().Context(), userID(c), id, rid, req.ExpectedRevision)
	if err != nil {
		return err
	}
	return data(c, http.StatusOK, map[string]any{"document": d, "version": v})
}

func (h *Handler) reviewDocument(c echo.Context) error {
	id, err := paramID(c, "id")
	if err != nil {
		return err
	}
	var req reviewReq
	if err := bindJSON(c, &req); err != nil {
		return err
	}
	v, err := h.documents.Review(c.Request().Context(), userID(c), id, req.VersionID)
	if err != nil {
		return err
	}
	return data(c, http.StatusOK, map[string]any{"quality": v.Quality, "blocks": v.Blocks})
}

func (h *Handler) finalizeDocument(c echo.Context) error {
	id, err := paramID(c, "id")
	if err != nil {
		return err
	}
	var req finalizeReq
	if err := bindJSON(c, &req); err != nil {
		return err
	}
	d, err := h.documents.Finalize(c.Request().Context(), userID(c), id,
		req.ExpectedRevision, req.VersionID, req.ApprovalID)
	if err != nil {
		return err
	}
	return data(c, http.StatusOK, d)
}

func (h *Handler) archiveDocument(c echo.Context) error {
	id, err := paramID(c, "id")
	if err != nil {
		return err
	}
	var req expectedRevisionReq
	if err := bindJSON(c, &req); err != nil {
		return err
	}
	d, err := h.documents.Archive(c.Request().Context(), userID(c), id, req.ExpectedRevision)
	if err != nil {
		return err
	}
	return data(c, http.StatusOK, d)
}

func (h *Handler) createExport(c echo.Context) error {
	id, err := paramID(c, "id")
	if err != nil {
		return err
	}
	var req createExportReq
	if err := bindJSON(c, &req); err != nil {
		return err
	}
	e, err := h.documents.CreateExport(c.Request().Context(), userID(c), id, usecase.CreateExportInput{
		VersionID: req.VersionID, Format: domain.ExportFormat(req.Format),
		RendererVersion: req.RendererVersion,
	})
	if err != nil {
		return err
	}
	return data(c, http.StatusCreated, e)
}

func (h *Handler) recordExportResult(c echo.Context) error {
	id, err := paramID(c, "id")
	if err != nil {
		return err
	}
	eid, err := paramID(c, "exportId")
	if err != nil {
		return err
	}
	var req exportResultReq
	if err := bindJSON(c, &req); err != nil {
		return err
	}
	e, err := h.documents.RecordExportResult(c.Request().Context(), userID(c), id, eid, usecase.ExportResultInput{
		SHA256: req.SHA256, ByteLength: req.ByteLength, PageCount: req.PageCount,
		Validation: domain.ExportValidation{
			KoreanText: req.Validation.KoreanText, Links: req.Validation.Links, AtsText: req.Validation.AtsText,
		},
		Status: domain.ExportStatus(req.Status), ErrorCode: req.ErrorCode,
	})
	if err != nil {
		return err
	}
	return data(c, http.StatusOK, e)
}

func (h *Handler) listExports(c echo.Context) error {
	id, err := paramID(c, "id")
	if err != nil {
		return err
	}
	page, err := pageRequest(c)
	if err != nil {
		return err
	}
	p, err := h.documents.ListExports(c.Request().Context(), userID(c), id, page)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, pageBody(p))
}
