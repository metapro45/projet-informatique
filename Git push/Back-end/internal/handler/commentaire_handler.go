package handler

import (
	"database/sql"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"suivi-projets-backend/internal/domain"
	"suivi-projets-backend/internal/repository/postgres"
	"suivi-projets-backend/pkg/middleware"
	"suivi-projets-backend/pkg/response"
)

// CommentaireHandler gère les endpoints HTTP pour les commentaires de tâches.
// T118 — Commentaires sur les tâches Kanban
type CommentaireHandler struct {
	repo *postgres.CommentaireRepository
}

func NewCommentaireHandler(db *sql.DB) *CommentaireHandler {
	return &CommentaireHandler{repo: postgres.NewCommentaireRepository(db)}
}

// GET /api/taches/:id/commentaires
func (h *CommentaireHandler) GetByTache(c *fiber.Ctx) error {
	tacheID, err := parseUUIDParam(c, "id")
	if err != nil {
		return err
	}
	commentaires, err := h.repo.GetByTacheID(c.Context(), tacheID)
	if err != nil {
		return response.ServerError(c, err, "CommentaireHandler.GetByTache")
	}
	if commentaires == nil {
		commentaires = []*domain.Commentaire{}
	}
	return response.OK(c, commentaires)
}

// POST /api/taches/:id/commentaires
func (h *CommentaireHandler) Create(c *fiber.Ctx) error {
	tacheID, err := parseUUIDParam(c, "id")
	if err != nil {
		return err
	}

	var req struct {
		Contenu string `json:"contenu"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Body JSON invalide")
	}
	req.Contenu = strings.TrimSpace(req.Contenu)
	if req.Contenu == "" {
		return response.BadRequest(c, "Le champ 'contenu' est obligatoire")
	}
	if len(req.Contenu) > 2000 {
		return response.BadRequest(c, "Le commentaire ne peut pas dépasser 2000 caractères")
	}

	auteurID, err := uuid.Parse(middleware.GetUserID(c))
	if err != nil {
		return response.Unauthorized(c, "Impossible d'extraire votre identifiant")
	}

	commentaire := &domain.Commentaire{
		TacheID:  tacheID,
		AuteurID: auteurID,
		Contenu:  req.Contenu,
	}

	if err := h.repo.Create(c.Context(), commentaire); err != nil {
		return response.ServerError(c, err, "CommentaireHandler.Create")
	}
	return response.Created(c, commentaire)
}

// DELETE /api/taches/commentaires/:id
// Seul l'auteur ou un encadrant peut supprimer un commentaire
func (h *CommentaireHandler) Delete(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return err
	}

	commentaire, err := h.repo.GetByID(c.Context(), id)
	if err != nil {
		return response.ServerError(c, err, "CommentaireHandler.Delete — GetByID")
	}
	if commentaire == nil {
		return response.NotFound(c, "Commentaire")
	}

	// Vérifier droits : auteur ou encadrant
	userID   := middleware.GetUserID(c)
	userRole := middleware.GetUserRole(c)
	if commentaire.AuteurID.String() != userID && userRole != "encadrant" && userRole != "admin" {
		return response.Forbidden(c, "Seul l'auteur ou un encadrant peut supprimer ce commentaire")
	}

	if err := h.repo.Delete(c.Context(), id); err != nil {
		return response.ServerError(c, err, "CommentaireHandler.Delete")
	}
	return response.NoContent(c)
}
