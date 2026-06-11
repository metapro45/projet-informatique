package handler

import (
	"database/sql"
	"strings"

	"github.com/gofiber/fiber/v2"

	"suivi-projets-backend/internal/domain"
	"suivi-projets-backend/internal/repository/postgres"
	"suivi-projets-backend/pkg/response"
)

type JalonHandler struct {
	repo *postgres.JalonRepository
}

func NewJalonHandler(db *sql.DB) *JalonHandler {
	return &JalonHandler{repo: postgres.NewJalonRepository(db)}
}

// GET /api/projets/:id/jalons
func (h *JalonHandler) GetByProjet(c *fiber.Ctx) error {
	projetID, err := parseUUIDParam(c, "id")
	if err != nil {
		return err
	}
	jalons, err := h.repo.GetByProjetID(c.Context(), projetID)
	if err != nil {
		return response.ServerError(c, err, "JalonHandler.GetByProjet")
	}
	if jalons == nil {
		jalons = []*domain.Jalon{}
	}
	return response.OK(c, jalons)
}

// POST /api/projets/:id/jalons
func (h *JalonHandler) Create(c *fiber.Ctx) error {
	projetID, err := parseUUIDParam(c, "id")
	if err != nil {
		return err
	}
	var req struct {
		Titre       string  `json:"titre"`
		Description *string `json:"description"`
		DateLimite  *string `json:"date_limite"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Body JSON invalide")
	}
	req.Titre = strings.TrimSpace(req.Titre)
	if req.Titre == "" {
		return response.BadRequest(c, "Le champ 'titre' est obligatoire")
	}

	jalon := &domain.Jalon{
		Titre:       req.Titre,
		Description: req.Description,
		ProjetID:    &projetID,
	}

	if req.DateLimite != nil && *req.DateLimite != "" {
		d, err := parseDate(c, req.DateLimite, "date_limite")
		if err != nil {
			return err
		}
		if d != nil {
			jalon.DateLimite = *d
		}
	}

	if err := h.repo.Create(c.Context(), jalon); err != nil {
		return response.ServerError(c, err, "JalonHandler.Create")
	}
	return response.Created(c, jalon)
}

// PUT /api/jalons/:id
func (h *JalonHandler) Update(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return err
	}
	existing, err := h.repo.GetByID(c.Context(), id)
	if err != nil {
		return response.ServerError(c, err, "JalonHandler.Update — GetByID")
	}
	if existing == nil {
		return response.NotFound(c, "Jalon")
	}

	var req struct {
		Titre       string  `json:"titre"`
		Description *string `json:"description"`
		DateLimite  *string `json:"date_limite"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Body JSON invalide")
	}
	req.Titre = strings.TrimSpace(req.Titre)
	if req.Titre == "" {
		return response.BadRequest(c, "Le champ 'titre' est obligatoire")
	}

	existing.Titre = req.Titre
	existing.Description = req.Description
	if req.DateLimite != nil && *req.DateLimite != "" {
		d, err := parseDate(c, req.DateLimite, "date_limite")
		if err != nil {
			return err
		}
		if d != nil {
			existing.DateLimite = *d
		}
	}

	if err := h.repo.Update(c.Context(), existing); err != nil {
		return response.ServerError(c, err, "JalonHandler.Update")
	}
	return response.OK(c, existing)
}

// DELETE /api/jalons/:id
func (h *JalonHandler) Delete(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return err
	}
	existing, err := h.repo.GetByID(c.Context(), id)
	if err != nil {
		return response.ServerError(c, err, "JalonHandler.Delete — GetByID")
	}
	if existing == nil {
		return response.NotFound(c, "Jalon")
	}
	if err := h.repo.Delete(c.Context(), id); err != nil {
		return response.ServerError(c, err, "JalonHandler.Delete")
	}
	return response.NoContent(c)
}
