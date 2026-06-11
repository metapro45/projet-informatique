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

type ProjetHandler struct {
	repo *postgres.ProjetRepository
}

func NewProjetHandler(db *sql.DB) *ProjetHandler {
	return &ProjetHandler{repo: postgres.NewProjetRepository(db)}
}

// ── Structures de requête ─────────────────────────────────────────────────────

type createProjetRequest struct {
	Titre       string  `json:"titre"`
	Description *string `json:"description"`
	Statut      string  `json:"statut"`
	DateDebut   *string `json:"date_debut"`
	DateFin     *string `json:"date_fin"`
	GroupID     *string `json:"group_id"`   // T40 — groupe académique cible
}

type updateProjetRequest struct {
	Titre       string  `json:"titre"`
	Description *string `json:"description"`
	Statut      string  `json:"statut"`
	DateDebut   *string `json:"date_debut"`
	DateFin     *string `json:"date_fin"`
	GroupID     *string `json:"group_id"`
}

// statutsValides regroupe les valeurs acceptées pour le champ statut.
var statutsValides = map[string]bool{
	"actif": true, "archive": true, "termine": true,
}

// ── Helpers de validation ─────────────────────────────────────────────────────

// parseUUIDParam parse un paramètre de route en UUID et retourne une erreur 400 si invalide.
func parseUUIDParam(c *fiber.Ctx, param string) (uuid.UUID, error) {
	id, err := uuid.Parse(c.Params(param))
	if err != nil {
		return uuid.Nil, response.BadRequest(c,
			"L'identifiant '"+c.Params(param)+"' est invalide — UUID attendu")
	}
	return id, nil
}

// parseDate parse une date YYYY-MM-DD depuis une string.
func parseDate(c *fiber.Ctx, s *string, champ string) (*domain.Date, error) {
	if s == nil || *s == "" {
		return nil, nil
	}
	// Valider le format basique YYYY-MM-DD
	if len(*s) != 10 || (*s)[4] != '-' || (*s)[7] != '-' {
		return nil, response.BadRequest(c,
			"Format invalide pour '"+champ+"' — attendu : YYYY-MM-DD")
	}
	d := &domain.Date{}
	if err := d.UnmarshalJSON([]byte(`"` + *s + `"`)); err != nil {
		return nil, response.BadRequest(c,
			"Date invalide pour '"+champ+"' : "+*s)
	}
	return d, nil
}

// validateProjetRequest valide les champs communs à Create et Update.
func validateProjetRequest(c *fiber.Ctx, titre, statut string) error {
	titre = strings.TrimSpace(titre)
	if titre == "" {
		return response.BadRequest(c, "Le champ 'titre' est obligatoire et ne peut pas être vide")
	}
	if len(titre) > 200 {
		return response.BadRequest(c, "Le titre ne peut pas dépasser 200 caractères")
	}
	if statut == "" {
		return response.BadRequest(c, "Le champ 'statut' est obligatoire")
	}
	if !statutsValides[statut] {
		return response.BadRequest(c,
			"Statut '"+statut+"' invalide — valeurs acceptées : actif | archive | termine")
	}
	return nil
}

// ── 2.1 — GET /api/projets ────────────────────────────────────────────────────

func (h *ProjetHandler) GetAll(c *fiber.Ctx) error {
	projets, err := h.repo.GetAll(c.Context())
	if err != nil {
		return response.ServerError(c, err, "ProjetHandler.GetAll")
	}
	if projets == nil {
		projets = []*domain.Projet{}
	}
	return response.OK(c, projets)
}

// ── 2.2 — POST /api/projets ───────────────────────────────────────────────────

func (h *ProjetHandler) Create(c *fiber.Ctx) error {
	var req createProjetRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Body JSON invalide — vérifiez le format de la requête")
	}

	req.Titre = strings.TrimSpace(req.Titre)
	if err := validateProjetRequest(c, req.Titre, req.Statut); err != nil {
		return err
	}

	// Extraire l'enseignant_id depuis le JWT
	enseignantIDStr := middleware.GetUserID(c)
	enseignantID, err := uuid.Parse(enseignantIDStr)
	if err != nil {
		return response.Unauthorized(c, "Impossible d'extraire votre identifiant du token")
	}

	projet := &domain.Projet{
		Titre:        req.Titre,
		Description:  req.Description,
		Statut:       domain.StatutProjet(req.Statut),
		EnseignantID: &enseignantID,
	}

	if projet.DateDebut, err = parseDate(c, req.DateDebut, "date_debut"); err != nil {
		return err
	}
	if projet.DateFin, err = parseDate(c, req.DateFin, "date_fin"); err != nil {
		return err
	}

	// Valider cohérence des dates
	if projet.DateDebut != nil && projet.DateFin != nil {
		if projet.DateFin.Time.Before(projet.DateDebut.Time) {
			return response.BadRequest(c, "La date_fin ne peut pas être antérieure à date_debut")
		}
	}

	// Parser group_id optionnel (T40)
	if req.GroupID != nil && *req.GroupID != "" {
		gid, err := uuid.Parse(*req.GroupID)
		if err != nil {
			return response.BadRequest(c, "group_id invalide — UUID attendu")
		}
		projet.GroupID = &gid
	}

	if err := h.repo.Create(c.Context(), projet); err != nil {
		return response.ServerError(c, err, "ProjetHandler.Create")
	}
	return response.Created(c, projet)
}

// ── 2.3 — GET /api/projets/:id ────────────────────────────────────────────────

func (h *ProjetHandler) GetByID(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return err
	}

	projet, err := h.repo.GetByID(c.Context(), id)
	if err != nil {
		return response.ServerError(c, err, "ProjetHandler.GetByID")
	}
	if projet == nil {
		return response.NotFound(c, "Projet")
	}
	return response.OK(c, projet)
}

// ── 2.4 — PUT /api/projets/:id ────────────────────────────────────────────────

func (h *ProjetHandler) Update(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return err
	}

	// Vérifier existence + propriété
	projet, err := h.repo.GetByID(c.Context(), id)
	if err != nil {
		return response.ServerError(c, err, "ProjetHandler.Update — GetByID")
	}
	if projet == nil {
		return response.NotFound(c, "Projet")
	}
	userID := middleware.GetUserID(c)
	if projet.EnseignantID == nil || projet.EnseignantID.String() != userID {
		return response.Forbidden(c, "Vous n'êtes pas le propriétaire de ce projet")
	}

	var req updateProjetRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Body JSON invalide")
	}

	req.Titre = strings.TrimSpace(req.Titre)
	if err := validateProjetRequest(c, req.Titre, req.Statut); err != nil {
		return err
	}

	projet.Titre       = req.Titre
	projet.Description = req.Description
	projet.Statut      = domain.StatutProjet(req.Statut)
	projet.DateDebut   = nil
	projet.DateFin     = nil
	projet.GroupID     = nil

	if projet.DateDebut, err = parseDate(c, req.DateDebut, "date_debut"); err != nil {
		return err
	}
	if projet.DateFin, err = parseDate(c, req.DateFin, "date_fin"); err != nil {
		return err
	}

	if projet.DateDebut != nil && projet.DateFin != nil {
		if projet.DateFin.Time.Before(projet.DateDebut.Time) {
			return response.BadRequest(c, "La date_fin ne peut pas être antérieure à date_debut")
		}
	}

	// Parser group_id optionnel (T40)
	if req.GroupID != nil && *req.GroupID != "" {
		gid, err := uuid.Parse(*req.GroupID)
		if err != nil {
			return response.BadRequest(c, "group_id invalide — UUID attendu")
		}
		projet.GroupID = &gid
	}

	if err := h.repo.Update(c.Context(), projet); err != nil {
		return response.ServerError(c, err, "ProjetHandler.Update")
	}
	return response.OK(c, projet)
}

// ── 2.5 — DELETE /api/projets/:id ─────────────────────────────────────────────

func (h *ProjetHandler) Delete(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return err
	}

	projet, err := h.repo.GetByID(c.Context(), id)
	if err != nil {
		return response.ServerError(c, err, "ProjetHandler.Delete — GetByID")
	}
	if projet == nil {
		return response.NotFound(c, "Projet")
	}
	userID := middleware.GetUserID(c)
	if projet.EnseignantID == nil || projet.EnseignantID.String() != userID {
		return response.Forbidden(c, "Vous n'êtes pas le propriétaire de ce projet")
	}

	if err := h.repo.Delete(c.Context(), id); err != nil {
		return response.ServerError(c, err, "ProjetHandler.Delete")
	}
	return response.NoContent(c)
}
