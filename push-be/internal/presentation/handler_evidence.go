package presentation

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"push-be/internal/domain"
	"push-be/internal/domain/usecase"
)

func (h *Handler) listEvidence(c echo.Context) error {
	page, err := pageRequest(c)
	if err != nil {
		return err
	}
	f := domain.EvidenceFilter{Page: page, Query: c.QueryParam("query")}
	if k := c.QueryParam("kind"); k != "" {
		kind := domain.EvidenceKind(k)
		if !domain.ValidEvidenceKind(kind) {
			return domain.ValidationField("kind", "unsupported kind")
		}
		f.Kind = &kind
	}
	p, err := h.evidence.List(c.Request().Context(), userID(c), f)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, pageBody(p))
}

func (h *Handler) createEvidence(c echo.Context) error {
	var req createEvidenceReq
	if err := bindJSON(c, &req); err != nil {
		return err
	}
	e, err := h.evidence.Create(c.Request().Context(), userID(c), usecase.CreateEvidenceInput{
		Kind: domain.EvidenceKind(req.Kind), Title: req.Title, SourceText: req.SourceText,
		SourceURL: req.SourceURL, Skills: req.Skills, SupersedesID: req.SupersedesID,
	})
	if err != nil {
		return err
	}
	return data(c, http.StatusCreated, e)
}

func (h *Handler) getEvidence(c echo.Context) error {
	id, err := paramID(c, "id")
	if err != nil {
		return err
	}
	e, err := h.evidence.Get(c.Request().Context(), userID(c), id)
	if err != nil {
		return err
	}
	return data(c, http.StatusOK, e)
}

func (h *Handler) importEvidence(c echo.Context) error {
	var req importEvidenceReq
	if err := bindJSON(c, &req); err != nil {
		return err
	}
	op, err := h.evidence.Import(c.Request().Context(), userID(c), usecase.ImportInput{
		SourceID: req.SourceID, Text: req.Text, SourceURL: req.SourceURL,
		ContentHash: req.ContentHash, Format: req.Format,
	})
	if err != nil {
		return err
	}
	return data(c, http.StatusAccepted, op)
}

func (h *Handler) archiveEvidence(c echo.Context) error {
	id, err := paramID(c, "id")
	if err != nil {
		return err
	}
	var req expectedRevisionReq
	if err := bindJSON(c, &req); err != nil {
		return err
	}
	e, err := h.evidence.Archive(c.Request().Context(), userID(c), id, req.ExpectedRevision)
	if err != nil {
		return err
	}
	return data(c, http.StatusOK, e)
}

var allowedSourceMIME = map[string]bool{
	"application/pdf": true,
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
	"text/plain":               true,
	"text/markdown":            true,
	"application/octet-stream": true,
}

const maxSourceBytes = 20 << 20

func (h *Handler) createSource(c echo.Context) error {
	kind := c.FormValue("kind")
	if kind == "" {
		return domain.ValidationField("kind", "required")
	}
	fh, err := c.FormFile("file")
	if err != nil {
		return domain.ValidationField("file", "required")
	}
	if fh.Size > maxSourceBytes {
		return domain.PayloadTooLarge()
	}
	f, err := fh.Open()
	if err != nil {
		return domain.ValidationField("file", "cannot read file")
	}
	defer f.Close()
	buf, err := io.ReadAll(io.LimitReader(f, maxSourceBytes+1))
	if err != nil || int64(len(buf)) > maxSourceBytes {
		return domain.PayloadTooLarge()
	}
	mime := fh.Header.Get("Content-Type")
	if mime == "" {
		mime = http.DetectContentType(buf)
	}
	if !allowedSourceMIME[mime] {
		return domain.ValidationField("file", "unsupported file type "+mime)
	}
	sum := sha256.Sum256(buf)
	id := uuid.New()
	path := filepath.Join(h.storageDir, userID(c).String(), id.String())
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return domain.Internal()
	}
	if err := os.WriteFile(path, buf, 0o600); err != nil {
		return domain.Internal()
	}
	src := &domain.Source{
		ID: id, UserID: userID(c), FileName: fh.Filename, MimeType: mime,
		Size: int64(len(buf)), SHA256: hex.EncodeToString(sum[:]),
		Path: path, Status: "STORED", CreatedAt: time.Now().UTC(),
	}
	if err := h.sources.Create(c.Request().Context(), src); err != nil {
		return domain.Internal()
	}
	return data(c, http.StatusCreated, map[string]any{
		"id": src.ID, "fileName": src.FileName, "mimeType": src.MimeType,
		"size": src.Size, "sha256": src.SHA256, "status": src.Status,
	})
}

func (h *Handler) getSource(c echo.Context) error {
	id, err := paramID(c, "id")
	if err != nil {
		return err
	}
	s, err := h.sources.Get(c.Request().Context(), userID(c), id)
	if err != nil {
		return domain.NotFound()
	}
	return data(c, http.StatusOK, map[string]any{
		"id": s.ID, "fileName": s.FileName, "mimeType": s.MimeType,
		"size": s.Size, "sha256": s.SHA256, "status": s.Status,
		"createdAt": s.CreatedAt.UTC().Format(time.RFC3339),
	})
}

func (h *Handler) getSourceContent(c echo.Context) error {
	id, err := paramID(c, "id")
	if err != nil {
		return err
	}
	s, err := h.sources.Get(c.Request().Context(), userID(c), id)
	if err != nil {
		return domain.NotFound()
	}
	return c.File(s.Path)
}

func (h *Handler) deleteSource(c echo.Context) error {
	id, err := paramID(c, "id")
	if err != nil {
		return err
	}
	ref, err := h.sources.Referenced(c.Request().Context(), userID(c), id)
	if err != nil {
		return domain.Internal()
	}
	if ref {
		return domain.InvalidTransition("source is referenced by evidence")
	}
	if err := h.sources.Delete(c.Request().Context(), userID(c), id); err != nil {
		return domain.NotFound()
	}
	return c.NoContent(http.StatusNoContent)
}
