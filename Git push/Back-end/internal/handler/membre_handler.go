package handler

import (
	"database/sql"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"suivi-projets-backend/internal/repository/postgres"
	"suivi-projets-backend/pkg/response"
)

type MembreHandler struct {
	repo *postgres.MembreRepository
}

func NewMembreHandler(db *sql.DB) *MembreHandler {
	return &MembreHandler{repo: postgres.NewMembreRepository(db)}
}

// GET /api/equipes/:id/membres
func (h *MembreHandler) GetByEquipe(c *fiber.Ctx) error {
	equipeID, err := parseUUIDParam(c, "id")
	if err != nil {
		return err
	}
	membres, err := h.repo.GetByEquipeID(c.Context(), equipeID)
	if err != nil {
		return response.ServerError(c, err, "MembreHandler.GetByEquipe")
	}
	if membres == nil {
		return response.OK(c, []interface{}{})
	}
	return response.OK(c, membres)
}

// POST /api/equipes/:id/membres
// Body : { "user_id": "uuid" }
func (h *MembreHandler) Add(c *fiber.Ctx) error {
	equipeID, err := parseUUIDParam(c, "id")
	if err != nil {
		return err
	}
	var req struct {
		UserID string `json:"user_id"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Body JSON invalide")
	}
	if req.UserID == "" {
		return response.BadRequest(c, "Le champ 'user_id' est obligatoire")
	}
	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		return response.BadRequest(c, "user_id invalide — UUID attendu")
	}
	membre, err := h.repo.Add(c.Context(), equipeID, userID)
	if err != nil {
		return response.ServerError(c, err, "MembreHandler.Add")
	}
	return response.Created(c, membre)
}

// DELETE /api/equipes/:id/membres/:user_id
func (h *MembreHandler) Remove(c *fiber.Ctx) error {
	equipeID, err := parseUUIDParam(c, "id")
	if err != nil {
		return err
	}
	userIDStr := c.Params("user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return response.BadRequest(c, "user_id invalide — UUID attendu")
	}
	if err := h.repo.Remove(c.Context(), equipeID, userID); err != nil {
		return response.NotFound(c, "Membre")
	}
	return response.NoContent(c)
}
