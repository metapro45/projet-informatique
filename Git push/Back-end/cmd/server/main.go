package main

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"suivi-projets-backend/config"
	"suivi-projets-backend/internal/routes"
	"suivi-projets-backend/pkg/database"
)

// main est le point d'entrée du serveur.
//
// Séquence de démarrage :
//  1. Charger la configuration depuis .env
//  2. Connecter la base de données PostgreSQL (Supabase)
//  3. Initialiser Fiber avec ses middlewares globaux
//  4. Configurer CORS (T109)
//  5. Enregistrer toutes les routes
//  6. Démarrer l'écoute HTTP
func main() {
	// ── 1. Configuration ───────────────────────────────────────────────────────
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("❌ Erreur de configuration : %v", err)
	}
	log.Println("✅ Configuration chargée")

	// ── 2. Connexion base de données ───────────────────────────────────────────
	if err := database.Connect(cfg.DSN()); err != nil {
		log.Printf("⚠️  BDD : %v", err)
	}

	// ── 3. Initialisation Fiber ────────────────────────────────────────────────
	app := fiber.New(fiber.Config{
		AppName: "Suivi Projets Étudiants API v1.0",
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{
				"success": false,
				"error":   err.Error(),
			})
		},
	})

	// ── 4. Middleware Recover (anti-crash) ────────────────────────────────────
	app.Use(recover.New())

	// ── 5. Middleware CORS (T109) ─────────────────────────────────────────────
	// Origines autorisées :
	//   - localhost:5173 (Vite dev — M2 Frontend)
	//   - localhost:3000 (alternative React dev)
	//   - URL de production (à renseigner dans .env avant déploiement)
	app.Use(cors.New(cors.Config{
		AllowOrigins: cfg.CORSOrigins,
		AllowMethods: "GET,POST,PUT,PATCH,DELETE,OPTIONS",
		AllowHeaders: "Origin,Content-Type,Accept,Authorization",
		// AllowCredentials : false car on utilise JWT dans le header,
		// pas des cookies — plus simple et plus sécurisé
		AllowCredentials: false,
		MaxAge:           3600,
	}))

	// ── 6. Middleware Logger ───────────────────────────────────────────────────
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${method} ${path} → ${status} (${latency})\n",
	}))

	// ── 7. Enregistrement des routes ──────────────────────────────────────────
	routes.Register(app, database.DB, cfg.JWTSecret, cfg.SupabaseURL, cfg.SupabaseAnonKey, cfg.SupabaseServiceRole)

	// ── 8. Démarrage ──────────────────────────────────────────────────────────
	addr := ":" + cfg.Port
	log.Printf("🚀 Serveur démarré sur http://localhost%s", addr)
	log.Fatal(app.Listen(addr))
}
