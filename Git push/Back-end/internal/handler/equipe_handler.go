package handler

import (
	"database/sql"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"suivi-projets-backend/internal/domain"
	"suivi-projets-backend/internal/repository/postgres"
	"suivi-projets-backend/pkg/response"
)

type EquipeHandler struct {
	repo *postgres.EquipeRepository
}

func NewEquipeHandler(db *sql.DB) *EquipeHandler {
	return &EquipeHandler{repo: postgres.NewEquipeRepository(db)}
}

// GET /api/projets/:id/equipes
func (h *EquipeHandler) GetByProjet(c *fiber.Ctx) error {
	projetID, err := parseUUIDParam(c, "id")
	if err != nil {
		return err
	}
	equipes, err := h.repo.GetByProjetID(c.Context(), projetID)
	if err != nil {
		return response.ServerError(c, err, "EquipeHandler.GetByProjet")
	}
	if equipes == nil {
		equipes = []*domain.Equipe{}
	}
	return response.OK(c, equipes)
}

// GET /api/equipes/:id
func (h *EquipeHandler) GetByID(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return err
	}
	equipe, err := h.repo.GetByID(c.Context(), id)
	if err != nil {
		return response.ServerError(c, err, "EquipeHandler.GetByID")
	}
	if equipe == nil {
		return response.NotFound(c, "Équipe")
	}
	return response.OK(c, equipe)
}

// POST /api/projets/:id/equipes
func (h *EquipeHandler) Create(c *fiber.Ctx) error {
	projetID, err := parseUUIDParam(c, "id")
	if err != nil {
		return err
	}
	var req struct {
		Nom string `json:"nom"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Body JSON invalide")
	}
	req.Nom = strings.TrimSpace(req.Nom)
	if req.Nom == "" {
		return response.BadRequest(c, "Le champ 'nom' est obligatoire")
	}
	if len(req.Nom) > 100 {
		return response.BadRequest(c, "Le nom ne peut pas dépasser 100 caractères")
	}

	equipe := &domain.Equipe{Nom: req.Nom, ProjetID: &projetID}
	if err := h.repo.Create(c.Context(), equipe); err != nil {
		return response.ServerError(c, err, "EquipeHandler.Create")
	}
	return response.Created(c, equipe)
}

// PUT /api/equipes/:id
func (h *EquipeHandler) Update(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return err
	}

	// Vérifier existence
	equipe, err := h.repo.GetByID(c.Context(), id)
	if err != nil {
		return response.ServerError(c, err, "EquipeHandler.Update — GetByID")
	}
	if equipe == nil {
		return response.NotFound(c, "Équipe")
	}

	var req struct {
		Nom string `json:"nom"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Body JSON invalide")
	}
	req.Nom = strings.TrimSpace(req.Nom)
	if req.Nom == "" {
		return response.BadRequest(c, "Le champ 'nom' est obligatoire")
	}
	if len(req.Nom) > 100 {
		return response.BadRequest(c, "Le nom ne peut pas dépasser 100 caractères")
	}

	equipe.Nom = req.Nom
	if err := h.repo.Update(c.Context(), equipe); err != nil {
		return response.ServerError(c, err, "EquipeHandler.Update")
	}
	return response.OK(c, equipe)
}

// DELETE /api/equipes/:id
func (h *EquipeHandler) Delete(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return err
	}

	// Vérifier existence avant suppression
	equipe, err := h.repo.GetByID(c.Context(), id)
	if err != nil {
		return response.ServerError(c, err, "EquipeHandler.Delete — GetByID")
	}
	if equipe == nil {
		return response.NotFound(c, "Équipe")
	}

	if err := h.repo.Delete(c.Context(), id); err != nil {
		return response.ServerError(c, err, "EquipeHandler.Delete")
	}
	return response.NoContent(c)
}

// parseUUIDFromStr parse un UUID depuis une string — utilisé dans les helpers membres.
func parseUUIDFromStr(c *fiber.Ctx, s, label string) (uuid.UUID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil, response.BadRequest(c,
			"L'identifiant '"+label+"' est invalide — UUID attendu")
	}
	return id, nil
}
