package presentation

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"

	"push-be/internal/domain"
)

func (h *Handler) getOperation(c echo.Context) error {
	id, err := paramID(c, "id")
	if err != nil {
		return err
	}
	op, err := h.operations.Get(c.Request().Context(), userID(c), id)
	if err != nil {
		return err
	}
	return data(c, http.StatusOK, op)
}

func (h *Handler) operationInput(c echo.Context) error {
	id, err := paramID(c, "id")
	if err != nil {
		return err
	}
	var req opInputReq
	if err := bindJSON(c, &req); err != nil {
		return err
	}
	op, err := h.operations.SubmitInput(c.Request().Context(), userID(c), id, req.Fields, req.SourceID)
	if err != nil {
		return err
	}
	return data(c, http.StatusOK, op)
}

func (h *Handler) cancelOperation(c echo.Context) error {
	id, err := paramID(c, "id")
	if err != nil {
		return err
	}
	op, err := h.operations.Cancel(c.Request().Context(), userID(c), id)
	if err != nil {
		return err
	}
	return data(c, http.StatusOK, op)
}

func (h *Handler) operationEvents(c echo.Context) error {
	id, err := paramID(c, "id")
	if err != nil {
		return err
	}
	op, err := h.operations.Get(c.Request().Context(), userID(c), id)
	if err != nil {
		return err
	}
	w := c.Response()
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	flusher, ok := w.Writer.(http.Flusher)
	if !ok {
		return domain.Internal()
	}
	seq := 0
	emit := func(kind string, payload any) {
		seq++
		b, _ := json.Marshal(map[string]any{
			"operationId": op.ID, "sequence": seq, "payload": payload,
		})
		fmt.Fprintf(w, "id: %s:%d\nevent: %s\ndata: %s\n\n", op.ID, seq, kind, b)
		flusher.Flush()
	}
	emit("progress", map[string]any{"status": op.Status, "progress": op.Progress, "step": string(op.Status)})
	switch op.Status {
	case domain.OpSucceeded:
		if op.Result != nil {
			emit("result", json.RawMessage(op.Result.Value))
		}
	case domain.OpFailed:
		emit("error", op.Error)
	case domain.OpCancelled:
		emit("error", domain.OperationError{Code: "CANCELLED", Message: "operation cancelled", Retryable: false})
	}
	return nil
}
