package routes

import (
	"database/sql"
	"fmt"

	"github.com/gofiber/fiber/v2"

	"suivi-projets-backend/internal/handler"
	"suivi-projets-backend/pkg/middleware"
)

// Register déclare toutes les routes de l'API et les associe à leurs handlers.
// Appelée une seule fois au démarrage depuis cmd/server/main.go.
//
// Architecture de sécurité :
//   - /api/health/*  → publics (sans JWT) — diagnostic uniquement
//   - /api/*         → protégés par JWTProtected()
//   - Routes sensibles → protégées par RequireRole("encadrant")
func Register(app *fiber.App, db *sql.DB, jwtSecret, supabaseURL, anonKey, serviceRole string) {

	// ── Instanciation des handlers ─────────────────────────────────────────────
	authH      := handler.NewAuthHandler(db, supabaseURL, anonKey)
	adminH     := handler.NewAdminHandler(db, supabaseURL, serviceRole)
	dashboardH := handler.NewDashboardHandler(db)
	projetH   := handler.NewProjetHandler(db)
	equipeH   := handler.NewEquipeHandler(db)
	membreH   := handler.NewMembreHandler(db)
	tacheH        := handler.NewTacheHandler(db)
	commentaireH  := handler.NewCommentaireHandler(db)
	jalonH        := handler.NewJalonHandler(db)
	livrableH     := handler.NewLivrableHandler(db, supabaseURL, serviceRole, "livrables")
	hierarchieH   := handler.NewHierarchieHandler(db)

	// ════════════════════════════════════════════════════════════════════════════
	// ROUTES PUBLIQUES — Health checks (sans JWT)
	// ════════════════════════════════════════════════════════════════════════════
	health := app.Group("/api/health")

	// /api/health — vérifie que Fiber tourne
	health.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"success": true,
			"data": fiber.Map{
				"status":  "ok",
				"version": "1.0.0",
				"etape":   "3 — Middleware JWT",
			},
		})
	})

	// /api/health/db — vérifie la connexion PostgreSQL (T24)
	health.Get("/db", func(c *fiber.Ctx) error {
		if db == nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"success": false,
				"error":   "Base de données non connectée",
			})
		}
		var now string
		if err := db.QueryRowContext(c.Context(), "SELECT NOW()::text").Scan(&now); err != nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"success": false,
				"error":   fmt.Sprintf("Requête SQL échouée : %v", err),
			})
		}
		var tableCount int
		db.QueryRowContext(c.Context(), `
			SELECT COUNT(*) FROM information_schema.tables
			WHERE table_schema = 'public' AND table_type = 'BASE TABLE'
		`).Scan(&tableCount)

		rows, _ := db.QueryContext(c.Context(), `
			SELECT table_name FROM information_schema.tables
			WHERE table_schema = 'public' AND table_type = 'BASE TABLE'
			ORDER BY table_name
		`)
		defer rows.Close()
		tables := []string{}
		for rows.Next() {
			var t string
			if rows.Scan(&t) == nil {
				tables = append(tables, t)
			}
		}
		return c.JSON(fiber.Map{
			"success": true,
			"data": fiber.Map{
				"status":      "connecté",
				"server_time": now,
				"tables":      tableCount,
				"schema":      tables,
			},
		})
	})

	// /api/health/indexes — vérifie les index (T13)
	health.Get("/indexes", func(c *fiber.Ctx) error {
		if db == nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"success": false, "error": "Base de données non connectée",
			})
		}
		rows, err := db.QueryContext(c.Context(), `
			SELECT i.relname, t.relname,
				array_to_string(array_agg(a.attname ORDER BY k.n), ', '),
				ix.indisunique, ix.indisprimary
			FROM pg_class t
				JOIN pg_index ix ON t.oid = ix.indrelid
				JOIN pg_class i  ON i.oid = ix.indexrelid
				JOIN pg_namespace n ON n.oid = t.relnamespace
				JOIN LATERAL unnest(ix.indkey) WITH ORDINALITY AS k(attnum, n) ON TRUE
				JOIN pg_attribute a ON a.attrelid = t.oid AND a.attnum = k.attnum
			WHERE n.nspname = 'public' AND t.relkind = 'r'
			GROUP BY i.relname, t.relname, ix.indisunique, ix.indisprimary
			ORDER BY t.relname, ix.indisprimary DESC, i.relname
		`)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false, "error": err.Error(),
			})
		}
		defer rows.Close()
		type Idx struct {
			Nom      string `json:"nom"`
			Table    string `json:"table"`
			Colonnes string `json:"colonnes"`
			Unique   bool   `json:"unique"`
			Primaire bool   `json:"primaire"`
		}
		indexes := []Idx{}
		fkCount := 0
		for rows.Next() {
			var idx Idx
			if rows.Scan(&idx.Nom, &idx.Table, &idx.Colonnes, &idx.Unique, &idx.Primaire) == nil {
				indexes = append(indexes, idx)
				if !idx.Primaire {
					fkCount++
				}
			}
		}
		return c.JSON(fiber.Map{
			"success": true,
			"data": fiber.Map{
				"total_index": len(indexes), "index_fk_manuels": fkCount,
				"t13_valide": fkCount > 0,  "detail": indexes,
			},
		})
	})

	// /api/health/schema — inspecte la structure réelle des tables
	health.Get("/schema", func(c *fiber.Ctx) error {
		if db == nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"success": false, "error": "Base de données non connectée",
			})
		}
		rows, err := db.QueryContext(c.Context(), `
			SELECT table_name, column_name, data_type, is_nullable, column_default
			FROM information_schema.columns
			WHERE table_schema = 'public'
			ORDER BY table_name, ordinal_position
		`)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false, "error": err.Error(),
			})
		}
		defer rows.Close()
		type Col struct {
			Colonne  string  `json:"colonne"`
			Type     string  `json:"type"`
			Nullable string  `json:"nullable"`
			Default  *string `json:"default,omitempty"`
		}
		schema := map[string][]Col{}
		for rows.Next() {
			var tableName string
			var col Col
			if rows.Scan(&tableName, &col.Colonne, &col.Type, &col.Nullable, &col.Default) == nil {
				schema[tableName] = append(schema[tableName], col)
			}
		}
		return c.JSON(fiber.Map{"success": true, "data": schema})
	})

	// ════════════════════════════════════════════════════════════════════════════
	// ROUTES AUTH — Publiques (sans JWT) — T28 & T29
	// ════════════════════════════════════════════════════════════════════════════
	auth := app.Group("/api/auth")
// T28 — Inscription publique désactivée (v2)
	// La création de comptes passe désormais par POST /api/admin/users
	// Garder la route mais retourner 410 Gone
	auth.Post("/register", func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusGone).JSON(fiber.Map{
			"success": false,
			"error":   "Inscription publique désactivée. Contactez un administrateur.",
		})
	}) // T28 — Créer un compte
	auth.Post("/login",    authH.Login)    // T29 — Se connecter

	// ════════════════════════════════════════════════════════════════════════════
	// ROUTES PROTÉGÉES — JWT requis sur toutes les routes /api
	// ════════════════════════════════════════════════════════════════════════════
	api := app.Group("/api", middleware.JWTProtected(jwtSecret))

	// ── Route de test JWT ──────────────────────────────────────────────────────
	// Permet à Postman de vérifier que le token est bien lu (T34)
	api.Get("/me", func(c *fiber.Ctx) error {
		userID := middleware.GetUserID(c)

		// Enrichir avec les données du profil depuis profiles
		var nom, prenom string
		var groupID *string
		var mustChangePwd, isActive bool

		if db != nil {
			db.QueryRowContext(c.Context(), `
				SELECT
					COALESCE(nom, ''),
					COALESCE(prenom, ''),
					group_id::text,
					COALESCE(must_change_password, false),
					COALESCE(is_active, true)
				FROM profiles WHERE id = $1
			`, userID).Scan(&nom, &prenom, &groupID, &mustChangePwd, &isActive)
		}

		return c.JSON(fiber.Map{
			"success": true,
			"data": fiber.Map{
				"user_id":              userID,
				"user_role":            middleware.GetUserRole(c),
				"user_email":           c.Locals(middleware.ContextUserEmail),
				"nom":                  nom,
				"prenom":               prenom,
				"group_id":             groupID,
				"must_change_password": mustChangePwd,
				"is_active":            isActive,
			},
		})
	})

	// ════════════════════════════════════════════════════════════════════════════
	// ADMIN — Routes réservées au rôle admin (T95/T96)
	// ════════════════════════════════════════════════════════════════════════════
	admin := api.Group("/admin", middleware.RequireRole("admin"))
	admin.Get("/users",       adminH.GetAllUsers)
	admin.Post("/users",      adminH.CreateUser)
	admin.Patch("/users/:id",   adminH.UpdateUser)
	admin.Post("/users/bulk",   adminH.BulkImport)  // T114 — Import CSV

	// T97 — Hiérarchie académique (lecture seule)
	admin.Get("/institutions", hierarchieH.GetInstitutions)
	admin.Get("/programs",     hierarchieH.GetPrograms)
	admin.Get("/cohorts",      hierarchieH.GetCohorts)
	admin.Get("/groups",       hierarchieH.GetGroups)

	// ════════════════════════════════════════════════════════════════════════════
	// DASHBOARD — Routes filtrées par rôle (T98/T99)
	// ════════════════════════════════════════════════════════════════════════════
	api.Get("/encadrant/projets", middleware.RequireRole("encadrant"), dashboardH.GetProjetsEncadrant)
	api.Get("/etudiant/projet",   middleware.RequireRole("etudiant"),  dashboardH.GetProjetEtudiant)

	// ════════════════════════════════════════════════════════════════════════════
	// §2 — PROJETS (Étape 4)
	// ════════════════════════════════════════════════════════════════════════════
	projets := api.Group("/projets")
	projets.Get("/",      projetH.GetAll)   // etudiant | enseignant
	projets.Post("/",     middleware.RequireRole("encadrant"), projetH.Create)
	projets.Get("/:id",   projetH.GetByID)
	projets.Put("/:id",   middleware.RequireRole("encadrant"), projetH.Update)
	projets.Delete("/:id",middleware.RequireRole("encadrant"), projetH.Delete)

	// ════════════════════════════════════════════════════════════════════════════
	// §3 — ÉQUIPES (Étape 4)
	// ════════════════════════════════════════════════════════════════════════════
	projets.Get("/:id/equipes",  equipeH.GetByProjet)
	projets.Post("/:id/equipes", middleware.RequireRole("encadrant"), equipeH.Create)

	equipes := api.Group("/equipes")
	equipes.Get("/:id",    equipeH.GetByID)  // §3 — Voir une équipe par ID
	equipes.Put("/:id",    middleware.RequireRole("encadrant"), equipeH.Update)
	equipes.Delete("/:id", middleware.RequireRole("encadrant"), equipeH.Delete)

	// ════════════════════════════════════════════════════════════════════════════
	// §4 — MEMBRES (Étape 4)
	// ════════════════════════════════════════════════════════════════════════════
	equipes.Get("/:id/membres",             membreH.GetByEquipe)
	equipes.Post("/:id/membres",            middleware.RequireRole("encadrant"), membreH.Add)
	equipes.Delete("/:id/membres/:user_id", middleware.RequireRole("encadrant"), membreH.Remove)

	// ════════════════════════════════════════════════════════════════════════════
	// §5 — TÂCHES KANBAN (Étape 5)
	// ════════════════════════════════════════════════════════════════════════════
	equipes.Get("/:id/taches",  tacheH.GetByEquipe)
	equipes.Post("/:id/taches", tacheH.Create) // etudiant | enseignant

	taches := api.Group("/taches")
	taches.Patch("/reorder",    tacheH.Reorder)       // T53 — réordonnancement (AVANT /:id)
	taches.Patch("/:id/statut", tacheH.UpdateStatut) // ⚡ Kanban drag-and-drop
	taches.Patch("/:id",        tacheH.Update)
	taches.Delete("/:id",       middleware.RequireRole("encadrant"), tacheH.Delete)
	taches.Get("/:id/commentaires",  commentaireH.GetByTache)
	taches.Post("/:id/commentaires", commentaireH.Create)

	commentaires := api.Group("/taches/commentaires")
	commentaires.Delete("/:id", commentaireH.Delete)

	// ════════════════════════════════════════════════════════════════════════════
	// §6 — JALONS (Étape 5)
	// ════════════════════════════════════════════════════════════════════════════
	projets.Get("/:id/jalons",  jalonH.GetByProjet)
	projets.Post("/:id/jalons", middleware.RequireRole("encadrant"), jalonH.Create)

	jalons := api.Group("/jalons")
	jalons.Put("/:id",    middleware.RequireRole("encadrant"), jalonH.Update)
	jalons.Delete("/:id", middleware.RequireRole("encadrant"), jalonH.Delete)

	// ════════════════════════════════════════════════════════════════════════════
	// §7 — LIVRABLES (Étape 6)
	// ════════════════════════════════════════════════════════════════════════════
	jalons.Get("/:id/livrables",   livrableH.GetByJalon)
	equipes.Post("/:id/livrables", livrableH.Create) // etudiant | enseignant

	livrables := api.Group("/livrables")
	livrables.Patch("/:id/note", middleware.RequireRole("encadrant"), livrableH.Noter)
	livrables.Delete("/:id",     middleware.RequireRole("encadrant"), livrableH.Delete)
}
