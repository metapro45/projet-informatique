package handler

import (
	"database/sql"

	"github.com/gofiber/fiber/v2"

	"suivi-projets-backend/internal/domain"
	"suivi-projets-backend/pkg/middleware"
	"suivi-projets-backend/pkg/response"
)

// DashboardHandler gère les endpoints filtrés par rôle.
// T98 : GET /api/encadrant/projets
// T99 : GET /api/etudiant/projet
type DashboardHandler struct {
	db *sql.DB
}

func NewDashboardHandler(db *sql.DB) *DashboardHandler {
	return &DashboardHandler{db: db}
}

// ── T99 — GET /api/etudiant/projet ────────────────────────────────────────────
// Retourne le projet du groupe de l'étudiant connecté.
// Jointure : profiles.group_id → academic_groups → projets.group_id

func (h *DashboardHandler) GetProjetEtudiant(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	row := h.db.QueryRowContext(c.Context(), `
		SELECT p.id, p.titre, p.description, p.statut,
		       p.date_debut, p.date_fin, p.enseignant_id,
		       p.group_id, p.created_at
		FROM projets p
		JOIN profiles pr ON pr.group_id = p.group_id
		WHERE pr.id = $1
		LIMIT 1
	`, userID)

	var projet domain.Projet
	var description sql.NullString
	var dateDebut, dateFin sql.NullTime
	var enseignantID, groupID sql.NullString
	var createdAt sql.NullTime

	err := row.Scan(
		&projet.ID, &projet.Titre, &description, &projet.Statut,
		&dateDebut, &dateFin, &enseignantID, &groupID, &createdAt,
	)
	if err == sql.ErrNoRows {
		return response.NotFound(c, "Projet de votre groupe")
	}
	if err != nil {
		return response.ServerError(c, err, "DashboardHandler.GetProjetEtudiant")
	}

	if description.Valid { projet.Description = &description.String }
	if createdAt.Valid   { t := createdAt.Time.UTC(); projet.CreatedAt = &t }

	return response.OK(c, projet)
}

// ── T98 — GET /api/encadrant/projets ─────────────────────────────────────────
// Retourne les projets des groupes liés aux filières de l'encadrant.
// Jointure : teacher_programs → programs → cohorts → academic_groups → projets

func (h *DashboardHandler) GetProjetsEncadrant(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	rows, err := h.db.QueryContext(c.Context(), `
		SELECT DISTINCT p.id, p.titre, p.description, p.statut,
		       p.date_debut, p.date_fin, p.enseignant_id,
		       p.group_id, p.created_at
		FROM projets p
		JOIN academic_groups ag ON ag.id = p.group_id
		JOIN cohorts co         ON co.id = ag.cohort_id
		JOIN programs pr        ON pr.id = co.program_id
		JOIN teacher_programs tp ON tp.program_id = pr.id
		WHERE tp.user_id = $1
		ORDER BY p.created_at DESC
	`, userID)

	if err != nil {
		return response.ServerError(c, err, "DashboardHandler.GetProjetsEncadrant")
	}
	defer rows.Close()

	projets := []*domain.Projet{}
	for rows.Next() {
		var p domain.Projet
		var description sql.NullString
		var dateDebut, dateFin sql.NullTime
		var enseignantID, groupID sql.NullString
		var createdAt sql.NullTime

		if err := rows.Scan(
			&p.ID, &p.Titre, &description, &p.Statut,
			&dateDebut, &dateFin, &enseignantID, &groupID, &createdAt,
		); err != nil {
			return response.ServerError(c, err, "DashboardHandler.GetProjetsEncadrant scan")
		}
		if description.Valid { p.Description = &description.String }
		if createdAt.Valid   { t := createdAt.Time.UTC(); p.CreatedAt = &t }
		projets = append(projets, &p)
	}

	return response.OK(c, projets)
}
