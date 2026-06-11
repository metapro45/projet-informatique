package handler

import (
	"bytes"
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"suivi-projets-backend/internal/domain"
	"suivi-projets-backend/internal/repository/postgres"
	"suivi-projets-backend/pkg/middleware"
	"suivi-projets-backend/pkg/response"
)

// LivrableHandler gère les endpoints HTTP pour les livrables.
// T62 : Configuration bucket Supabase Storage
// T63 : POST /api/equipes/:id/livrables — upload fichier
// T64 : GET  /api/jalons/:id/livrables
type LivrableHandler struct {
	repo          *postgres.LivrableRepository
	supabaseURL   string
	serviceRole   string
	storageBucket string
}

func NewLivrableHandler(db *sql.DB, supabaseURL, serviceRole, bucket string) *LivrableHandler {
	return &LivrableHandler{
		repo:          postgres.NewLivrableRepository(db),
		supabaseURL:   supabaseURL,
		serviceRole:   serviceRole,
		storageBucket: bucket,
	}
}

// ── Extensions autorisées ─────────────────────────────────────────────────────
var extensionsAutorisees = map[string]bool{
	".pdf": true, ".doc": true, ".docx": true,
	".zip": true, ".tar": true, ".gz": true,
	".png": true, ".jpg": true, ".jpeg": true,
	".mp4": true, ".mov": true,
}

const tailleMaxFichier = 50 * 1024 * 1024 // 50 MB

// ── T64 — GET /api/jalons/:id/livrables ──────────────────────────────────────

func (h *LivrableHandler) GetByJalon(c *fiber.Ctx) error {
	jalonID, err := parseUUIDParam(c, "id")
	if err != nil {
		return err
	}
	livrables, err := h.repo.GetByJalonID(c.Context(), jalonID)
	if err != nil {
		return response.ServerError(c, err, "LivrableHandler.GetByJalon")
	}
	if livrables == nil {
		livrables = []*domain.Livrable{}
	}
	return response.OK(c, livrables)
}

// ── T63 — POST /api/equipes/:id/livrables ────────────────────────────────────
// Flux :
//  1. Valider le fichier (extension, taille)
//  2. Uploader vers Supabase Storage
//  3. Insérer le livrable dans public.livrables
//  4. Retourner le livrable créé avec sa fichier_url

func (h *LivrableHandler) Create(c *fiber.Ctx) error {
	equipeID, err := parseUUIDParam(c, "id")
	if err != nil {
		return err
	}

	// ── 1. Récupérer le fichier multipart ─────────────────────────────────────
	fileHeader, err := c.FormFile("fichier")
	if err != nil {
		return response.BadRequest(c,
			"Fichier manquant — champ 'fichier' requis (multipart/form-data)")
	}

	// Valider la taille
	if fileHeader.Size > tailleMaxFichier {
		return response.BadRequest(c,
			fmt.Sprintf("Fichier trop volumineux — maximum %dMB", tailleMaxFichier/1024/1024))
	}

	// Valider l'extension
	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if !extensionsAutorisees[ext] {
		return response.BadRequest(c,
			"Extension non autorisée — acceptées : pdf, doc, docx, zip, png, jpg, mp4...")
	}

	// Récupérer le nom personnalisé (optionnel) et le jalon_id (optionnel)
	nom := c.FormValue("nom")
	if nom == "" {
		nom = fileHeader.Filename
	}
	jalonIDStr := c.FormValue("jalon_id")

	// ── 2. Ouvrir le fichier ──────────────────────────────────────────────────
	file, err := fileHeader.Open()
	if err != nil {
		return response.ServerError(c, err, "LivrableHandler.Create — open file")
	}
	defer file.Close()

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		return response.ServerError(c, err, "LivrableHandler.Create — read file")
	}

	// ── 3. Construire le chemin de stockage ───────────────────────────────────
	// Format : equipes/{equipe_id}/{timestamp}_{filename}
	userID := middleware.GetUserID(c)
	fileName := fmt.Sprintf("%d_%s", time.Now().Unix(), fileHeader.Filename)
	storagePath := fmt.Sprintf("equipes/%s/%s", equipeID.String(), fileName)

	// ── 4. Uploader vers Supabase Storage ────────────────────────────────────
	fichierURL, err := h.uploadToStorage(storagePath, fileBytes, fileHeader.Header.Get("Content-Type"))
	if err != nil {
		log.Printf("[STORAGE ERROR] %v", err)
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   "Upload Storage echoue : " + err.Error(),
		})
	}

	// ── 5. Insérer dans public.livrables ─────────────────────────────────────
	userUUID, _ := uuid.Parse(userID)
	livrable := &domain.Livrable{
		Nom:        nom,
		FichierURL: fichierURL,
		EquipeID:   &equipeID,
		DeposePar:  &userUUID,
	}

	// Rattacher au jalon si fourni
	if jalonIDStr != "" {
		jid, err := uuid.Parse(jalonIDStr)
		if err != nil {
			return response.BadRequest(c, "jalon_id invalide — UUID attendu")
		}
		livrable.JalonID = &jid
	}

	if err := h.repo.Create(c.Context(), livrable); err != nil {
		log.Printf("[INSERT ERROR] %v", err)
		h.deleteFromStorage(storagePath)
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   "INSERT echoue : " + err.Error(),
		})
	}

	return response.Created(c, livrable)
}

// ── PATCH /api/livrables/:id/note ────────────────────────────────────────────

func (h *LivrableHandler) Noter(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return err
	}

	var req struct {
		Note        float64 `json:"note"`
		Commentaire *string `json:"commentaire"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Body JSON invalide")
	}
	if req.Note < 0 || req.Note > 20 {
		return response.BadRequest(c, "La note doit être entre 0 et 20")
	}

	livrable, err := h.repo.Noter(c.Context(), id, req.Note, req.Commentaire)
	if err != nil {
		return response.ServerError(c, err, "LivrableHandler.Noter")
	}
	if livrable == nil {
		return response.NotFound(c, "Livrable")
	}
	return response.OK(c, livrable)
}

// ── DELETE /api/livrables/:id ────────────────────────────────────────────────

func (h *LivrableHandler) Delete(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return err
	}

	livrable, err := h.repo.GetByID(c.Context(), id)
	if err != nil {
		return response.ServerError(c, err, "LivrableHandler.Delete — GetByID")
	}
	if livrable == nil {
		return response.NotFound(c, "Livrable")
	}

	// Supprimer de la BDD
	if err := h.repo.Delete(c.Context(), id); err != nil {
		return response.ServerError(c, err, "LivrableHandler.Delete")
	}

	// Supprimer le fichier du Storage (best-effort, pas bloquant)
	// Extraire le chemin depuis l'URL
	if livrable.FichierURL != "" {
		pathInBucket := extractStoragePath(livrable.FichierURL, h.storageBucket)
		if pathInBucket != "" {
			h.deleteFromStorage(pathInBucket)
		}
	}

	return response.NoContent(c)
}

// ── Helpers Supabase Storage ──────────────────────────────────────────────────

// uploadToStorage envoie un fichier vers Supabase Storage via l'API REST.
// Retourne l'URL publique signée du fichier.
func (h *LivrableHandler) uploadToStorage(path string, data []byte, contentType string) (string, error) {
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	url := fmt.Sprintf("%s/storage/v1/object/%s/%s",
		h.supabaseURL, h.storageBucket, path)

	req, err := http.NewRequest("POST", url, bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("upload: créer requête : %w", err)
	}
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("apikey", h.serviceRole)
	req.Header.Set("Authorization", "Bearer "+h.serviceRole)

	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return "", fmt.Errorf("upload: appel HTTP : %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("upload: Supabase Storage erreur %d : %s",
			resp.StatusCode, string(body))
	}

	// Construire l'URL de téléchargement
	// Format : {supabaseURL}/storage/v1/object/public/{bucket}/{path}
	fichierURL := fmt.Sprintf("%s/storage/v1/object/public/%s/%s",
		h.supabaseURL, h.storageBucket, path)

	return fichierURL, nil
}

// deleteFromStorage supprime un fichier du Storage (best-effort).
func (h *LivrableHandler) deleteFromStorage(path string) {
	url := fmt.Sprintf("%s/storage/v1/object/%s/%s",
		h.supabaseURL, h.storageBucket, path)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return
	}
	req.Header.Set("apikey", h.serviceRole)
	req.Header.Set("Authorization", "Bearer "+h.serviceRole)
	(&http.Client{}).Do(req)
}

// extractStoragePath extrait le chemin relatif depuis une URL Supabase Storage.
func extractStoragePath(url, bucket string) string {
	marker := "/object/public/" + bucket + "/"
	idx := strings.Index(url, marker)
	if idx == -1 {
		return ""
	}
	return url[idx+len(marker):]
}
