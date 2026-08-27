package handler

import (
	"golang-payment-ledger/errs"
	"golang-payment-ledger/service"

	"github.com/gofiber/fiber/v2"
)

type transferHandler struct {
	transferSvc service.TransferService
}

func NewTransferHandler(transferSvc service.TransferService) transferHandler {
	return transferHandler{transferSvc: transferSvc}
}

func (h transferHandler) Transfer(c *fiber.Ctx) error {
	var req service.TransferRequest
	if err := c.BodyParser(&req); err != nil {
		return handleError(c, errs.NewValidationError("invalid request body"))
	}
	req.IdempotencyKey = c.Get("Idempotency-Key")

	resp, err := h.transferSvc.Transfer(c.Context(), req)
	if err != nil {
		return handleError(c, err)
	}

	return sendSuccess(c, fiber.StatusOK, "transfer applied", resp)
}
