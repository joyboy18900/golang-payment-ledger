package handler

import (
	"strconv"

	"golang-payment-ledger/errs"
	"golang-payment-ledger/service"

	"github.com/gofiber/fiber/v2"
)

type accountHandler struct {
	accountSvc service.AccountService
}

func NewAccountHandler(accountSvc service.AccountService) accountHandler {
	return accountHandler{accountSvc: accountSvc}
}

func (h accountHandler) Create(c *fiber.Ctx) error {
	var req service.CreateAccountRequest
	if err := c.BodyParser(&req); err != nil {
		return handleError(c, errs.NewValidationError("invalid request body"))
	}

	resp, err := h.accountSvc.Create(c.Context(), req)
	if err != nil {
		return handleError(c, err)
	}

	return sendSuccess(c, fiber.StatusCreated, "account created", resp)
}

func (h accountHandler) GetBalance(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return handleError(c, errs.NewValidationError("id must be an integer"))
	}

	resp, err := h.accountSvc.GetBalance(c.Context(), id)
	if err != nil {
		return handleError(c, err)
	}

	return sendSuccess(c, fiber.StatusOK, "balance retrieved", resp)
}
