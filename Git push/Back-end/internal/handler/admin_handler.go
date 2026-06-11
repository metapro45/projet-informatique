package handler

import (
	"bytes"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"suivi-projets-backend/pkg/response"
)

// AdminHandler gère les endpoints réservés au rôle admin.
// T95 : POST /api/admin/users
// T96 : GET + PATCH /api/admin/users
type AdminHandler struct {
	db          *sql.DB
	supabaseURL string
	serviceRole string
}

func NewAdminHandler(db *sql.DB, supabaseURL, serviceRole string) *AdminHandler {
	return &AdminHandler{db: db, supabaseURL: supabaseURL, serviceRole: serviceRole}
}

// ── T96 — GET /api/admin/users ────────────────────────────────────────────────

func (h *AdminHandler) GetAllUsers(c *fiber.Ctx) error {
	rows, err := h.db.QueryContext(c.Context(), `
		SELECT id, email, nom, COALESCE(prenom,''), role,
		       is_active, must_change_password, group_id::text
		FROM profiles
		ORDER BY role, nom
	`)
	if err != nil {
		return response.ServerError(c, err, "AdminHandler.GetAllUsers")
	}
	defer rows.Close()

	type UserRow struct {
		ID                 string  `json:"id"`
		Email              string  `json:"email"`
		Nom                string  `json:"nom"`
		Prenom             string  `json:"prenom"`
		Role               string  `json:"role"`
		IsActive           bool    `json:"is_active"`
		MustChangePwd      bool    `json:"must_change_password"`
		GroupID            *string `json:"group_id"`
	}

	users := []UserRow{}
	for rows.Next() {
		var u UserRow
		if err := rows.Scan(&u.ID, &u.Email, &u.Nom, &u.Prenom,
			&u.Role, &u.IsActive, &u.MustChangePwd, &u.GroupID); err != nil {
			return response.ServerError(c, err, "AdminHandler.GetAllUsers scan")
		}
		users = append(users, u)
	}
	return response.OK(c, users)
}

// ── T95 — POST /api/admin/users ───────────────────────────────────────────────

func (h *AdminHandler) CreateUser(c *fiber.Ctx) error {
	var req struct {
		Nom     string  `json:"nom"`
		Prenom  *string `json:"prenom"`
		Email   string  `json:"email"`
		Role    string  `json:"role"`
		GroupID *string `json:"group_id"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Body JSON invalide")
	}
	if req.Email == "" || req.Nom == "" {
		return response.BadRequest(c, "email et nom sont obligatoires")
	}
	if req.Role != "etudiant" && req.Role != "encadrant" {
		return response.BadRequest(c, "role invalide — valeurs : etudiant | encadrant")
	}
	if h.serviceRole == "" {
		return response.Unavailable(c)
	}

	// Créer le compte via Supabase Admin API
	payload := map[string]interface{}{
		"email":    req.Email,
		"password": "Password123!", // mot de passe temporaire
		"email_confirm": true,
		"user_metadata": map[string]interface{}{
			"nom":    req.Nom,
			"prenom": req.Prenom,
			"role":   req.Role,
		},
		"app_metadata": map[string]interface{}{
			"role": req.Role,
		},
	}

	body, _ := json.Marshal(payload)
	httpReq, _ := http.NewRequest("POST",
		h.supabaseURL+"/auth/v1/admin/users", bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("apikey", h.serviceRole)
	httpReq.Header.Set("Authorization", "Bearer "+h.serviceRole)

	resp, err := (&http.Client{}).Do(httpReq)
	if err != nil {
		return response.ServerError(c, err, "AdminHandler.CreateUser — Supabase API")
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	var supaResp map[string]interface{}
	json.Unmarshal(respBody, &supaResp)

	if resp.StatusCode >= 400 {
		errMsg, _ := supaResp["msg"].(string)
		if errMsg == "" {
			errMsg = fmt.Sprintf("Erreur Supabase Admin API (status %d)", resp.StatusCode)
		}
		return response.BadRequest(c, errMsg)
	}

	userID, _ := supaResp["id"].(string)
	if userID == "" {
		return response.ServerError(c, nil, "AdminHandler.CreateUser — UUID vide")
	}

	// Insérer dans profiles
	prenom := ""
	if req.Prenom != nil {
		prenom = *req.Prenom
	}

	var groupIDVal interface{}
	if req.GroupID != nil && *req.GroupID != "" {
		gid, err := uuid.Parse(*req.GroupID)
		if err != nil {
			return response.BadRequest(c, "group_id invalide — UUID attendu")
		}
		groupIDVal = gid
	}

	_, err = h.db.ExecContext(c.Context(), `
		INSERT INTO profiles (id, email, nom, prenom, role, is_active, must_change_password, group_id)
		VALUES ($1, $2, $3, $4, $5, true, true, $6)
		ON CONFLICT (id) DO UPDATE SET
			nom=EXCLUDED.nom, prenom=EXCLUDED.prenom,
			role=EXCLUDED.role, group_id=EXCLUDED.group_id
	`, userID, req.Email, req.Nom, prenom, req.Role, groupIDVal)

	if err != nil {
		return response.ServerError(c, err, "AdminHandler.CreateUser — INSERT profiles")
	}

	return response.Created(c, fiber.Map{
		"id":     userID,
		"email":  req.Email,
		"nom":    req.Nom,
		"prenom": prenom,
		"role":   req.Role,
		"must_change_password": true,
	})
}

// ── T96 — PATCH /api/admin/users/:id ─────────────────────────────────────────

func (h *AdminHandler) UpdateUser(c *fiber.Ctx) error {
	id := c.Params("id")
	if _, err := uuid.Parse(id); err != nil {
		return response.BadRequest(c, "Identifiant invalide — UUID attendu")
	}

	var req struct {
		IsActive *bool   `json:"is_active"`
		Role     *string `json:"role"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Body JSON invalide")
	}

	if req.Role != nil && *req.Role != "etudiant" && *req.Role != "encadrant" && *req.Role != "admin" {
		return response.BadRequest(c, "role invalide — valeurs : etudiant | encadrant | admin")
	}

	// Mettre à jour is_active si fourni
	if req.IsActive != nil {
		_, err := h.db.ExecContext(c.Context(),
			"UPDATE profiles SET is_active=$2 WHERE id=$1", id, *req.IsActive)
		if err != nil {
			return response.ServerError(c, err, "AdminHandler.UpdateUser — is_active")
		}
	}

	// Mettre à jour le rôle si fourni
	if req.Role != nil {
		_, err := h.db.ExecContext(c.Context(),
			"UPDATE profiles SET role=$2 WHERE id=$1", id, *req.Role)
		if err != nil {
			return response.ServerError(c, err, "AdminHandler.UpdateUser — role")
		}
	}

	// Retourner le profil mis à jour
	var user struct {
		ID       string `json:"id"`
		Email    string `json:"email"`
		Nom      string `json:"nom"`
		Role     string `json:"role"`
		IsActive bool   `json:"is_active"`
	}
	err := h.db.QueryRowContext(c.Context(),
		"SELECT id::text, email, nom, role, is_active FROM profiles WHERE id=$1", id,
	).Scan(&user.ID, &user.Email, &user.Nom, &user.Role, &user.IsActive)
	if err == sql.ErrNoRows {
		return response.NotFound(c, "Utilisateur")
	}
	if err != nil {
		return response.ServerError(c, err, "AdminHandler.UpdateUser — SELECT")
	}

	return response.OK(c, user)
}

// ── T114 — POST /api/admin/users/bulk ────────────────────────────────────────
// Import CSV d'étudiants en masse.
// Format CSV attendu : email,nom,prenom,group_id
// Crée chaque compte via Supabase Admin API + insère dans profiles.

func (h *AdminHandler) BulkImport(c *fiber.Ctx) error {
	// Récupérer le fichier CSV
	fileHeader, err := c.FormFile("fichier")
	if err != nil {
		return response.BadRequest(c, "Fichier CSV manquant — champ 'fichier' requis")
	}
	if fileHeader.Size > 5*1024*1024 {
		return response.BadRequest(c, "Fichier trop volumineux — maximum 5MB")
	}

	file, err := fileHeader.Open()
	if err != nil {
		return response.ServerError(c, err, "BulkImport — open")
	}
	defer file.Close()

	// Parser le CSV
	reader := csv.NewReader(file)
	reader.TrimLeadingSpace = true

	records, err := reader.ReadAll()
	if err != nil {
		return response.BadRequest(c, "Format CSV invalide : "+err.Error())
	}
	if len(records) < 2 {
		return response.BadRequest(c, "CSV vide ou sans données (header + au moins 1 ligne requise)")
	}

	// Valider l'en-tête
	header := records[0]
	expected := []string{"email", "nom", "prenom", "group_id"}
	for i, col := range expected {
		if i >= len(header) || strings.ToLower(strings.TrimSpace(header[i])) != col {
			return response.BadRequest(c,
				fmt.Sprintf("Colonne %d invalide — attendu '%s', reçu '%s'", i+1, col, header[i]))
		}
	}

	type ImportResult struct {
		Email   string `json:"email"`
		Succes  bool   `json:"succes"`
		Erreur  string `json:"erreur,omitempty"`
	}

	resultats := []ImportResult{}
	crees := 0
	erreurs := 0

	for lineNum, record := range records[1:] {
		if len(record) < 4 {
			resultats = append(resultats, ImportResult{
				Email:  fmt.Sprintf("ligne %d", lineNum+2),
				Succes: false,
				Erreur: "Ligne incomplète — 4 colonnes requises",
			})
			erreurs++
			continue
		}

		email   := strings.TrimSpace(record[0])
		nom     := strings.TrimSpace(record[1])
		prenom  := strings.TrimSpace(record[2])
		groupID := strings.TrimSpace(record[3])

		// Validation basique
		if email == "" || nom == "" {
			resultats = append(resultats, ImportResult{
				Email: email, Succes: false, Erreur: "email et nom obligatoires",
			})
			erreurs++
			continue
		}

		// Créer le compte via Supabase Admin API
		payload := map[string]interface{}{
			"email":         email,
			"password":      "Password123!",
			"email_confirm": true,
			"user_metadata": map[string]interface{}{
				"nom": nom, "prenom": prenom, "role": "etudiant",
			},
			"app_metadata": map[string]interface{}{"role": "etudiant"},
		}

		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest("POST",
			h.supabaseURL+"/auth/v1/admin/users", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("apikey", h.serviceRole)
		req.Header.Set("Authorization", "Bearer "+h.serviceRole)

		resp, err := (&http.Client{}).Do(req)
		if err != nil {
			resultats = append(resultats, ImportResult{
				Email: email, Succes: false, Erreur: "Erreur Supabase API",
			})
			erreurs++
			continue
		}
		defer resp.Body.Close()
		respBody, _ := io.ReadAll(resp.Body)

		var supaResp map[string]interface{}
		json.Unmarshal(respBody, &supaResp)

		if resp.StatusCode >= 400 {
			msg, _ := supaResp["msg"].(string)
			if msg == "" { msg = fmt.Sprintf("Erreur %d", resp.StatusCode) }
			resultats = append(resultats, ImportResult{
				Email: email, Succes: false, Erreur: msg,
			})
			erreurs++
			continue
		}

		userID, _ := supaResp["id"].(string)
		if userID == "" {
			resultats = append(resultats, ImportResult{
				Email: email, Succes: false, Erreur: "UUID manquant dans réponse",
			})
			erreurs++
			continue
		}

		// Insérer dans profiles
		_, dbErr := h.db.ExecContext(c.Context(), `
			INSERT INTO profiles (id, email, nom, prenom, role, is_active, must_change_password, group_id)
			VALUES ($1, $2, $3, $4, 'etudiant', true, true, $5)
			ON CONFLICT (id) DO NOTHING
		`, userID, email, nom, prenom, nullableUUID(groupID))

		if dbErr != nil {
			resultats = append(resultats, ImportResult{
				Email: email, Succes: false, Erreur: "Compte créé mais profil échoué : " + dbErr.Error(),
			})
			erreurs++
			continue
		}

		resultats = append(resultats, ImportResult{Email: email, Succes: true})
		crees++
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": erreurs == 0,
		"data": fiber.Map{
			"total":    len(records) - 1,
			"crees":    crees,
			"erreurs":  erreurs,
			"resultats": resultats,
		},
	})
}

// nullableUUID retourne nil si la string est vide, sinon la string.
func nullableUUID(s string) interface{} {
	if s == "" { return nil }
	return s
}
