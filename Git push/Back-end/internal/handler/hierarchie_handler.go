package handler

import (
	"database/sql"

	"github.com/gofiber/fiber/v2"

	"suivi-projets-backend/pkg/response"
)

// HierarchieHandler expose les tables de hiérarchie académique en lecture seule.
// T97 : GET /api/admin/institutions|programs|cohorts|groups
type HierarchieHandler struct {
	db *sql.DB
}

func NewHierarchieHandler(db *sql.DB) *HierarchieHandler {
	return &HierarchieHandler{db: db}
}

// GET /api/admin/institutions
func (h *HierarchieHandler) GetInstitutions(c *fiber.Ctx) error {
	rows, err := h.db.QueryContext(c.Context(),
		`SELECT id::text, nom, code, created_at FROM institutions ORDER BY nom`)
	if err != nil {
		return response.ServerError(c, err, "HierarchieHandler.GetInstitutions")
	}
	defer rows.Close()

	type Row struct {
		ID        string  `json:"id"`
		Nom       string  `json:"nom"`
		Code      string  `json:"code"`
		CreatedAt *string `json:"created_at,omitempty"`
	}
	result := []Row{}
	for rows.Next() {
		var r Row
		if err := rows.Scan(&r.ID, &r.Nom, &r.Code, &r.CreatedAt); err != nil {
			return response.ServerError(c, err, "HierarchieHandler.GetInstitutions scan")
		}
		result = append(result, r)
	}
	return response.OK(c, result)
}

// GET /api/admin/programs
func (h *HierarchieHandler) GetPrograms(c *fiber.Ctx) error {
	rows, err := h.db.QueryContext(c.Context(),
		`SELECT p.id::text, p.nom, COALESCE(p.code,''), p.department_id::text,
		        d.nom AS department_nom
		 FROM programs p
		 JOIN departments d ON d.id = p.department_id
		 ORDER BY p.nom`)
	if err != nil {
		return response.ServerError(c, err, "HierarchieHandler.GetPrograms")
	}
	defer rows.Close()

	type Row struct {
		ID            string `json:"id"`
		Nom           string `json:"nom"`
		Code          string `json:"code"`
		DepartmentID  string `json:"department_id"`
		DepartmentNom string `json:"department_nom"`
	}
	result := []Row{}
	for rows.Next() {
		var r Row
		if err := rows.Scan(&r.ID, &r.Nom, &r.Code, &r.DepartmentID, &r.DepartmentNom); err != nil {
			return response.ServerError(c, err, "HierarchieHandler.GetPrograms scan")
		}
		result = append(result, r)
	}
	return response.OK(c, result)
}

// GET /api/admin/cohorts
func (h *HierarchieHandler) GetCohorts(c *fiber.Ctx) error {
	rows, err := h.db.QueryContext(c.Context(),
		`SELECT c.id::text, c.nom, c.annee_debut, c.annee_fin,
		        c.program_id::text, p.nom AS program_nom
		 FROM cohorts c
		 JOIN programs p ON p.id = c.program_id
		 ORDER BY c.annee_debut DESC, c.nom`)
	if err != nil {
		return response.ServerError(c, err, "HierarchieHandler.GetCohorts")
	}
	defer rows.Close()

	type Row struct {
		ID         string `json:"id"`
		Nom        string `json:"nom"`
		AnneeDebut int    `json:"annee_debut"`
		AnneeFin   int    `json:"annee_fin"`
		ProgramID  string `json:"program_id"`
		ProgramNom string `json:"program_nom"`
	}
	result := []Row{}
	for rows.Next() {
		var r Row
		if err := rows.Scan(&r.ID, &r.Nom, &r.AnneeDebut, &r.AnneeFin,
			&r.ProgramID, &r.ProgramNom); err != nil {
			return response.ServerError(c, err, "HierarchieHandler.GetCohorts scan")
		}
		result = append(result, r)
	}
	return response.OK(c, result)
}

// GET /api/admin/groups
func (h *HierarchieHandler) GetGroups(c *fiber.Ctx) error {
	rows, err := h.db.QueryContext(c.Context(),
		`SELECT ag.id::text, ag.nom, ag.cohort_id::text,
		        c.nom AS cohort_nom
		 FROM academic_groups ag
		 JOIN cohorts c ON c.id = ag.cohort_id
		 ORDER BY ag.nom`)
	if err != nil {
		return response.ServerError(c, err, "HierarchieHandler.GetGroups")
	}
	defer rows.Close()

	type Row struct {
		ID        string `json:"id"`
		Nom       string `json:"nom"`
		CohortID  string `json:"cohort_id"`
		CohortNom string `json:"cohort_nom"`
	}
	result := []Row{}
	for rows.Next() {
		var r Row
		if err := rows.Scan(&r.ID, &r.Nom, &r.CohortID, &r.CohortNom); err != nil {
			return response.ServerError(c, err, "HierarchieHandler.GetGroups scan")
		}
		result = append(result, r)
	}
	return response.OK(c, result)
}
