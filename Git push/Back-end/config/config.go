package config

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Config regroupe toutes les variables d'environnement nécessaires au serveur.
type Config struct {
	Port                string
	DBHost              string
	DBPort              string
	DBUser              string
	DBPassword          string
	DBName              string
	SupabaseURL         string
	SupabaseAnonKey     string
	SupabaseServiceRole string // T110 — jamais exposée au Frontend
	JWTSecret           string
	CORSOrigins         string // T109 — origines autorisées séparées par virgules
	StorageBucket       string // T62  — nom du bucket Supabase Storage
}

// LoadConfig charge la configuration depuis .env puis les variables système.
func LoadConfig() (*Config, error) {
	loadDotEnv(".env")

	cfg := &Config{
		Port:                getEnv("PORT", "8080"),
		DBHost:              getEnv("DB_HOST", ""),
		DBPort:              getEnv("DB_PORT", "6543"),
		DBUser:              getEnv("DB_USER", ""),
		DBPassword:          getEnv("DB_PASSWORD", ""),
		DBName:              getEnv("DB_NAME", "postgres"),
		SupabaseURL:         getEnv("SUPABASE_URL", ""),
		SupabaseAnonKey:     getEnv("SUPABASE_ANON_KEY", ""),
		SupabaseServiceRole: getEnv("SUPABASE_SERVICE_ROLE_KEY", ""),
		JWTSecret:           getEnv("SUPABASE_JWT_SECRET", ""),
		// T109 — défaut : localhost Vite (dev) + localhost React (alt)
		// En prod : remplacer par l'URL Vercel/Netlify dans .env
		CORSOrigins:   getEnv("CORS_ORIGINS", "http://localhost:5173,http://localhost:3000"),
		StorageBucket: getEnv("SUPABASE_STORAGE_BUCKET", "livrables"),
	}

	// Variables obligatoires au démarrage
	required := map[string]string{
		"DB_HOST":             cfg.DBHost,
		"DB_USER":             cfg.DBUser,
		"DB_PASSWORD":         cfg.DBPassword,
		"SUPABASE_URL":        cfg.SupabaseURL,
		"SUPABASE_ANON_KEY":   cfg.SupabaseAnonKey,
		"SUPABASE_JWT_SECRET": cfg.JWTSecret,
	}
	for key, val := range required {
		if val == "" {
			return nil, fmt.Errorf("config: variable d'environnement obligatoire manquante : %s", key)
		}
	}

	// T110 — Avertissement si service_role manquante (pas bloquant en dev)
	if cfg.SupabaseServiceRole == "" {
		fmt.Println("⚠️  SUPABASE_SERVICE_ROLE_KEY manquante — endpoints /admin/users désactivés")
	} else {
		fmt.Println("✅ Clé service_role chargée")
	}

	return cfg, nil
}

// loadDotEnv lit le .env ligne par ligne — robuste BOM/CRLF Windows.
func loadDotEnv(filename string) {
	f, err := os.Open(filename)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimPrefix(line, "\xef\xbb\xbf")
		line = strings.TrimRight(line, "\r")
		line = strings.TrimSpace(line)

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		if os.Getenv(key) == "" {
			os.Setenv(key, val)
		}
	}
}

// DSN construit la chaîne de connexion PostgreSQL.
func (c *Config) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=require TimeZone=UTC",
		c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName,
	)
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
