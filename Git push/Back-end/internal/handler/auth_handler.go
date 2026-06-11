package handler

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gofiber/fiber/v2"
)

// AuthHandler gère les endpoints d'authentification.
// L'inscription et la connexion sont déléguées à Supabase Auth via son API REST.
// Notre Backend crée/synchronise ensuite le profil dans public.users.
type AuthHandler struct {
	db          *sql.DB
	supabaseURL string
	anonKey     string
}

// NewAuthHandler crée un AuthHandler avec les dépendances nécessaires.
func NewAuthHandler(db *sql.DB, supabaseURL, anonKey string) *AuthHandler {
	return &AuthHandler{
		db:          db,
		supabaseURL: supabaseURL,
		anonKey:     anonKey,
	}
}

// ── Structures de requête ─────────────────────────────────────────────────────

// RegisterRequest est le body attendu pour POST /api/auth/register
type RegisterRequest struct {
	Email  string `json:"email"`
	Password string `json:"password"`
	Nom    string `json:"nom"`
	Prenom string `json:"prenom"`
	Role   string `json:"role"` // "etudiant" | "encadrant"
}

// LoginRequest est le body attendu pour POST /api/auth/login
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// ── T28 — POST /api/auth/register ────────────────────────────────────────────

// Register godoc
// @Summary  Inscrire un nouvel utilisateur
// @Route    POST /api/auth/register
// @Roles    Public (sans JWT)
//
// Flux :
//  1. Valider le body (email, password, nom, role)
//  2. Appeler Supabase Auth API pour créer le compte
//  3. Insérer le profil étendu dans public.users
//  4. Retourner le token JWT Supabase
func (h *AuthHandler) Register(c *fiber.Ctx) error {
	// ── 1. Parser et valider le body ──────────────────────────────────────────
	var req RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Body JSON invalide",
		})
	}

	// Validation des champs obligatoires
	if req.Email == "" || req.Password == "" || req.Nom == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Champs obligatoires manquants : email, password, nom",
		})
	}
	if req.Role != "etudiant" && req.Role != "encadrant" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Role invalide — valeurs acceptées : etudiant | encadrant",
		})
	}
	if len(req.Password) < 6 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Le mot de passe doit contenir au moins 6 caractères",
		})
	}

	// ── 2. Appeler Supabase Auth pour créer le compte ─────────────────────────
	supabaseBody := map[string]interface{}{
		"email":    req.Email,
		"password": req.Password,
		"data": map[string]interface{}{
			"nom":    req.Nom,
			"prenom": req.Prenom,
			"role":   req.Role,
		},
	}

	supabaseResp, err := callSupabaseAuth(
		h.supabaseURL+"/auth/v1/signup",
		h.anonKey,
		supabaseBody,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   fmt.Sprintf("Erreur Supabase Auth : %v", err),
		})
	}

	// Vérifier si Supabase a retourné une erreur
	if errMsg, ok := supabaseResp["error_description"].(string); ok && errMsg != "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   errMsg,
		})
	}
	if errMsg, ok := supabaseResp["msg"].(string); ok && errMsg != "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   errMsg,
		})
	}

	// Extraire l'UUID Supabase
	userMap, _ := supabaseResp["user"].(map[string]interface{})
	userID, _ := userMap["id"].(string)
	if userID == "" {
		// Certaines configs Supabase retournent l'user directement
		userID, _ = supabaseResp["id"].(string)
	}

	if userID == "" {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Impossible de récupérer l'UUID utilisateur depuis Supabase",
		})
	}

	// ── 3. Insérer le profil dans public.users ────────────────────────────────
	_, err = h.db.ExecContext(c.Context(), `
		INSERT INTO profiles (id, email, nom, prenom, role, is_active, must_change_password)
		VALUES ($1, $2, $3, $4, $5, true, false)
		ON CONFLICT (id) DO UPDATE
		SET email  = EXCLUDED.email,
		    nom    = EXCLUDED.nom,
		    prenom = EXCLUDED.prenom,
		    role   = EXCLUDED.role
	`, userID, req.Email, req.Nom, req.Prenom, req.Role)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   fmt.Sprintf("Erreur insertion profil : %v", err),
		})
	}

	// ── 4. Retourner la réponse ───────────────────────────────────────────────
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"message":      "Compte créé avec succès",
			"user_id":      userID,
			"email":        req.Email,
			"role":         req.Role,
			"access_token": supabaseResp["access_token"],
		},
	})
}

// ── T29 — POST /api/auth/login ────────────────────────────────────────────────

// Login godoc
// @Summary  Connecter un utilisateur
// @Route    POST /api/auth/login
// @Roles    Public (sans JWT)
//
// Flux :
//  1. Valider le body (email, password)
//  2. Appeler Supabase Auth pour vérifier les identifiants
//  3. Retourner le token JWT + infos utilisateur
func (h *AuthHandler) Login(c *fiber.Ctx) error {
	// ── 1. Parser et valider ──────────────────────────────────────────────────
	var req LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Body JSON invalide",
		})
	}

	if req.Email == "" || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Email et mot de passe obligatoires",
		})
	}

	// ── 2. Appeler Supabase Auth ──────────────────────────────────────────────
	supabaseBody := map[string]interface{}{
		"email":    req.Email,
		"password": req.Password,
	}

	supabaseResp, err := callSupabaseAuth(
		h.supabaseURL+"/auth/v1/token?grant_type=password",
		h.anonKey,
		supabaseBody,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   fmt.Sprintf("Erreur Supabase Auth : %v", err),
		})
	}

	// Vérifier si Supabase retourne une erreur (mauvais identifiants)
	if errMsg, ok := supabaseResp["error_description"].(string); ok && errMsg != "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"error":   "Email ou mot de passe incorrect",
		})
	}

	// Extraire le token et les infos utilisateur
	accessToken, _ := supabaseResp["access_token"].(string)
	if accessToken == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"error":   "Email ou mot de passe incorrect",
		})
	}

	userMap, _ := supabaseResp["user"].(map[string]interface{})
	userID, _ := userMap["id"].(string)
	userEmail, _ := userMap["email"].(string)

	// Récupérer le rôle et le profil depuis profiles (v2)
	// Fallback sur user_metadata.role si le profil n'existe pas encore
	var userRole, userNom string
	var userPrenom *string

	err = h.db.QueryRowContext(c.Context(),
		"SELECT role, nom, COALESCE(prenom, '') FROM profiles WHERE id = $1", userID,
	).Scan(&userRole, &userNom, &userPrenom)

	// Si pas dans profiles, lire le rôle depuis user_metadata Supabase
	if err != nil || userRole == "" {
		if userMap != nil {
			if meta, ok := userMap["user_metadata"].(map[string]interface{}); ok {
				if r, ok := meta["role"].(string); ok && r != "" {
					userRole = r
				}
			}
		}
		if userNom == "" {
			userNom = userEmail // fallback : email comme nom
		}
	}

	// ── 3. Retourner la réponse ───────────────────────────────────────────────
	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"access_token":  accessToken,
			"token_type":    "Bearer",
			"expires_in":    supabaseResp["expires_in"],
			"refresh_token": supabaseResp["refresh_token"],
			"user": fiber.Map{
				"id":     userID,
				"email":  userEmail,
				"nom":    userNom,
				"prenom": userPrenom,
				"role":   userRole,
			},
		},
	})
}

// ── Helper : appel Supabase Auth API ─────────────────────────────────────────

// callSupabaseAuth envoie une requête POST à l'API Supabase Auth.
func callSupabaseAuth(url, anonKey string, body map[string]interface{}) (map[string]interface{}, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal body : %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("créer requête : %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apikey", anonKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("appel HTTP : %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("lire réponse : %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("parser réponse JSON : %w", err)
	}

	return result, nil
}
