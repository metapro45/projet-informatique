package database

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

// DB est l'instance partagée de la connexion PostgreSQL.
// Initialisée une seule fois au démarrage via Connect().
var DB *sql.DB

// Connect initialise la connexion au pool PostgreSQL (Supabase).
// La chaîne de connexion est construite depuis les variables d'environnement
// chargées par config.LoadConfig().
//
// TODO Étape 2 : Appelée dans cmd/server/main.go après LoadConfig().
func Connect(dsn string) error {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("database: impossible d'ouvrir la connexion : %w", err)
	}

	if err := db.Ping(); err != nil {
		return fmt.Errorf("database: ping échoué (vérifier les credentials Supabase) : %w", err)
	}

	// Configuration du pool de connexions
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)

	DB = db
	log.Println("✅ Connexion PostgreSQL (Supabase) établie")
	return nil
}
