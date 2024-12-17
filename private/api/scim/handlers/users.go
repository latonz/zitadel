package handlers

import (
	"context"
	"github.com/zitadel/zitadel/internal/command"
	"github.com/zitadel/zitadel/internal/query"
	"github.com/zitadel/zitadel/internal/zerrors"
)

type UsersHandler struct {
	command *command.Commands
	query   *query.Queries
}

func NewUsersHandler(command *command.Commands, query *query.Queries) *UsersHandler {
	return &UsersHandler{command, query}
}

func (h *UsersHandler) Get(_ context.Context, id string) error {
	return zerrors.ThrowNotFound(nil, "HANDLER-M007", "Errors.Users.NotFound")
}

func (h *UsersHandler) Delete(ctx context.Context, id string) error {
	_, err := h.command.RemoveUserV2(ctx, id, nil)
	return err
}
