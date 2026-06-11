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

// TacheHandler gère les endpoints HTTP du Kanban.
// Contrat d'API : §5 — Étape 5
type TacheHandler struct {
	repo *postgres.TacheRepository
}

func NewTacheHandler(db *sql.DB) *TacheHandler {
	return &TacheHandler{repo: postgres.NewTacheRepository(db)}
}

// ── Structures de requête ─────────────────────────────────────────────────────

type createTacheRequest struct {
	Titre        string  `json:"titre"`
	Description  *string `json:"description"`
	Statut       string  `json:"statut"`
	Priorite     *string `json:"priorite"`
	DateEcheance *string `json:"date_echeance"` // YYYY-MM-DD
	AssigneeID   *string `json:"assignee_id"`
}

type updateTacheRequest struct {
	Titre        string  `json:"titre"`
	Description  *string `json:"description"`
	Priorite     *string `json:"priorite"`
	DateEcheance *string `json:"date_echeance"`
	AssigneeID   *string `json:"assignee_id"`
}

type updateStatutRequest struct {
	Statut string `json:"statut"`
}

// statuts et priorités valides
var statutsTachesValides = map[string]bool{
	"todo": true, "en_cours": true, "termine": true,
}
var prioritesTachesValides = map[string]bool{
	"basse": true, "moyenne": true, "haute": true,
}

// ── T50 — GET /api/equipes/:id/taches ────────────────────────────────────────

func (h *TacheHandler) GetByEquipe(c *fiber.Ctx) error {
	equipeID, err := parseUUIDParam(c, "id")
	if err != nil {
		return err
	}
	taches, err := h.repo.GetByEquipeID(c.Context(), equipeID)
	if err != nil {
		return response.ServerError(c, err, "TacheHandler.GetByEquipe")
	}
	if taches == nil {
		taches = []*domain.Tache{}
	}
	return response.OK(c, taches)
}

// ── T51 — POST /api/equipes/:id/taches ───────────────────────────────────────

func (h *TacheHandler) Create(c *fiber.Ctx) error {
	equipeID, err := parseUUIDParam(c, "id")
	if err != nil {
		return err
	}

	var req createTacheRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Body JSON invalide")
	}

	// Validation titre
	req.Titre = strings.TrimSpace(req.Titre)
	if req.Titre == "" {
		return response.BadRequest(c, "Le champ 'titre' est obligatoire")
	}
	if len(req.Titre) > 200 {
		return response.BadRequest(c, "Le titre ne peut pas dépasser 200 caractères")
	}

	// Statut — défaut "todo" si absent
	if req.Statut == "" {
		req.Statut = "todo"
	}
	if !statutsTachesValides[req.Statut] {
		return response.BadRequest(c,
			"Statut invalide — valeurs acceptées : todo | en_cours | termine")
	}

	// Priorité optionnelle
	var priorite *domain.PrioriteTache
	if req.Priorite != nil && *req.Priorite != "" {
		if !prioritesTachesValides[*req.Priorite] {
			return response.BadRequest(c,
				"Priorité invalide — valeurs acceptées : basse | moyenne | haute")
		}
		p := domain.PrioriteTache(*req.Priorite)
		priorite = &p
	}

	// Date d'échéance optionnelle
	dateEch, err := parseDate(c, req.DateEcheance, "date_echeance")
	if err != nil {
		return err
	}

	// AssigneeID optionnel
	var assigneeID *uuid.UUID
	if req.AssigneeID != nil && *req.AssigneeID != "" {
		aid, err := uuid.Parse(*req.AssigneeID)
		if err != nil {
			return response.BadRequest(c, "assignee_id invalide — UUID attendu")
		}
		assigneeID = &aid
	}

	tache := &domain.Tache{
		Titre:        req.Titre,
		Description:  req.Description,
		Statut:       domain.StatutTache(req.Statut),
		Priorite:     priorite,
		DateEcheance: dateEch,
		EquipeID:     &equipeID,
		AssigneeID:   assigneeID,
	}

	if err := h.repo.Create(c.Context(), tache); err != nil {
		return response.ServerError(c, err, "TacheHandler.Create")
	}
	return response.Created(c, tache)
}

// ── T51 — PATCH /api/taches/:id ──────────────────────────────────────────────

func (h *TacheHandler) Update(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return err
	}

	tache, err := h.repo.GetByID(c.Context(), id)
	if err != nil {
		return response.ServerError(c, err, "TacheHandler.Update — GetByID")
	}
	if tache == nil {
		return response.NotFound(c, "Tâche")
	}

	// Vérifier que l'utilisateur appartient à l'équipe (sécurité basique)
	userID := middleware.GetUserID(c)
	_ = userID // TODO T93 : vérifier appartenance équipe

	var req updateTacheRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Body JSON invalide")
	}

	req.Titre = strings.TrimSpace(req.Titre)
	if req.Titre == "" {
		return response.BadRequest(c, "Le champ 'titre' est obligatoire")
	}

	var priorite *domain.PrioriteTache
	if req.Priorite != nil && *req.Priorite != "" {
		if !prioritesTachesValides[*req.Priorite] {
			return response.BadRequest(c,
				"Priorité invalide — valeurs acceptées : basse | moyenne | haute")
		}
		p := domain.PrioriteTache(*req.Priorite)
		priorite = &p
	}

	dateEch, err := parseDate(c, req.DateEcheance, "date_echeance")
	if err != nil {
		return err
	}

	var assigneeID *uuid.UUID
	if req.AssigneeID != nil && *req.AssigneeID != "" {
		aid, err := uuid.Parse(*req.AssigneeID)
		if err != nil {
			return response.BadRequest(c, "assignee_id invalide — UUID attendu")
		}
		assigneeID = &aid
	}

	tache.Titre        = req.Titre
	tache.Description  = req.Description
	tache.Priorite     = priorite
	tache.DateEcheance = dateEch
	tache.AssigneeID   = assigneeID

	if err := h.repo.Update(c.Context(), tache); err != nil {
		return response.ServerError(c, err, "TacheHandler.Update")
	}
	return response.OK(c, tache)
}

// ── T52 — PATCH /api/taches/:id/statut ───────────────────────────────────────
// Endpoint critique du drag-and-drop Kanban
// Appelé chaque fois qu'une carte est déplacée entre colonnes

func (h *TacheHandler) UpdateStatut(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return err
	}

	var req updateStatutRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Body JSON invalide — attendu : {\"statut\":\"todo|en_cours|termine\"}")
	}

	if req.Statut == "" {
		return response.BadRequest(c, "Le champ 'statut' est obligatoire")
	}
	if !statutsTachesValides[req.Statut] {
		return response.BadRequest(c,
			"Statut invalide — valeurs acceptées : todo | en_cours | termine")
	}

	tache, err := h.repo.UpdateStatut(c.Context(), id, domain.StatutTache(req.Statut))
	if err != nil {
		return response.ServerError(c, err, "TacheHandler.UpdateStatut")
	}
	if tache == nil {
		return response.NotFound(c, "Tâche")
	}
	return response.OK(c, tache)
}

// ── T53 — PATCH /api/taches/reorder ───────────────────────────────────────────
// Endpoint de réordonnancement Kanban — format proposé par M4
// M2 envoie la liste ordonnée des IDs après un drag-and-drop
// Le backend assigne position = index automatiquement

func (h *TacheHandler) Reorder(c *fiber.Ctx) error {
	var req struct {
		EquipeID string   `json:"equipe_id"`
		Statut   string   `json:"statut"`
		TaskIDs  []string `json:"task_ids"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c,
			`Body JSON invalide — attendu : {"equipe_id":"uuid","statut":"todo","task_ids":["uuid1","uuid2"]}`)
	}

	// Validation
	if req.EquipeID == "" {
		return response.BadRequest(c, "Le champ 'equipe_id' est obligatoire")
	}
	equipeID, err := uuid.Parse(req.EquipeID)
	if err != nil {
		return response.BadRequest(c, "equipe_id invalide — UUID attendu")
	}
	if !statutsTachesValides[req.Statut] {
		return response.BadRequest(c, "Statut invalide — valeurs : todo | en_cours | termine")
	}
	if len(req.TaskIDs) == 0 {
		return response.BadRequest(c, "Le tableau 'task_ids' ne peut pas être vide")
	}

	// Convertir task_ids en TacheOrdre avec position = index
	ordres := make([]domain.TacheOrdre, 0, len(req.TaskIDs))
	for i, idStr := range req.TaskIDs {
		id, err := uuid.Parse(idStr)
		if err != nil {
			return response.BadRequest(c, "task_id invalide : "+idStr)
		}
		ordres = append(ordres, domain.TacheOrdre{
			ID:       id,
			Position: i,
			Statut:   domain.StatutTache(req.Statut),
			EquipeID: equipeID,
		})
	}

	// Valider que toutes les tâches appartiennent bien à equipe_id + statut
	if err := h.repo.ReorderValidated(c.Context(), equipeID, domain.StatutTache(req.Statut), ordres); err != nil {
		if err.Error() == "validation" {
			return response.BadRequest(c, "Certaines tâches n'appartiennent pas à cette équipe ou à ce statut")
		}
		return response.ServerError(c, err, "TacheHandler.Reorder")
	}

	return response.OK(c, fiber.Map{
		"updated": len(ordres),
		"statut":  req.Statut,
	})
}

// ── T51 — DELETE /api/taches/:id ─────────────────────────────────────────────

func (h *TacheHandler) Delete(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return err
	}

	tache, err := h.repo.GetByID(c.Context(), id)
	if err != nil {
		return response.ServerError(c, err, "TacheHandler.Delete — GetByID")
	}
	if tache == nil {
		return response.NotFound(c, "Tâche")
	}

	if err := h.repo.Delete(c.Context(), id); err != nil {
		return response.ServerError(c, err, "TacheHandler.Delete")
	}
	return response.NoContent(c)
}
