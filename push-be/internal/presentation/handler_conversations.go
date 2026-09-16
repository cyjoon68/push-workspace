package presentation

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"push-be/internal/domain"
	"push-be/internal/domain/usecase"
)

func (h *Handler) listConversations(c echo.Context) error {
	page, err := pageRequest(c)
	if err != nil {
		return err
	}
	f := domain.ConversationFilter{Page: page}
	if a := c.QueryParam("applicationId"); a != "" {
		id, err := uuid.Parse(a)
		if err != nil {
			return domain.ValidationField("applicationId", "must be a UUID")
		}
		f.ApplicationID = &id
	}
	p, err := h.conversations.List(c.Request().Context(), userID(c), f)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, pageBody(p))
}

func (h *Handler) createConversation(c echo.Context) error {
	var req createConversationReq
	if err := bindJSON(c, &req); err != nil {
		return err
	}
	v, err := h.conversations.Create(c.Request().Context(), userID(c), usecase.CreateConversationInput{
		ApplicationID: req.ApplicationID, Title: req.Title,
	})
	if err != nil {
		return err
	}
	return data(c, http.StatusCreated, v)
}

func (h *Handler) patchConversation(c echo.Context) error {
	id, err := paramID(c, "id")
	if err != nil {
		return err
	}
	var req patchConversationReq
	if err := bindJSON(c, &req); err != nil {
		return err
	}
	v, err := h.conversations.Patch(c.Request().Context(), userID(c), id, usecase.PatchConversationInput{
		ExpectedRevision: req.ExpectedRevision, Title: req.Title, Pinned: req.Pinned,
	})
	if err != nil {
		return err
	}
	return data(c, http.StatusOK, v)
}

func (h *Handler) archiveConversation(c echo.Context) error {
	id, err := paramID(c, "id")
	if err != nil {
		return err
	}
	var req expectedRevisionReq
	if err := bindJSON(c, &req); err != nil {
		return err
	}
	v, err := h.conversations.Archive(c.Request().Context(), userID(c), id, req.ExpectedRevision)
	if err != nil {
		return err
	}
	return data(c, http.StatusOK, v)
}

func (h *Handler) listMessages(c echo.Context) error {
	id, err := paramID(c, "id")
	if err != nil {
		return err
	}
	page, err := pageRequest(c)
	if err != nil {
		return err
	}
	p, err := h.conversations.ListMessages(c.Request().Context(), userID(c), id, page)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, pageBody(p))
}

func (h *Handler) postMessage(c echo.Context) error {
	id, err := paramID(c, "id")
	if err != nil {
		return err
	}
	var req postMessageReq
	if err := bindJSON(c, &req); err != nil {
		return err
	}
	if req.Context == nil {
		return domain.ValidationField("context", "required")
	}
	if req.Context.EvidenceIDs == nil {
		return domain.ValidationField("context.evidenceIds", "required")
	}
	op, err := h.conversations.PostMessage(c.Request().Context(), userID(c), id, usecase.PostMessageInput{
		Text: req.Text,
		Context: usecase.MessageContext{
			DocumentID: req.Context.DocumentID, VersionID: req.Context.VersionID,
			EvidenceIDs: req.Context.EvidenceIDs,
		},
		AI: req.AI, AccessMode: domain.AccessMode(req.AccessMode),
	})
	if err != nil {
		return err
	}
	return data(c, http.StatusAccepted, op)
}
