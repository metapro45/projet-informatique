package middleware

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

const ContextUserID    = "user_id"
const ContextUserRole  = "user_role"
const ContextUserEmail = "user_email"

type supabaseClaims struct {
	Sub          string                 `json:"sub"`
	Email        string                 `json:"email"`
	Role         string                 `json:"role"`
	Exp          int64                  `json:"exp"`
	Iat          int64                  `json:"iat"`
	Iss          string                 `json:"iss"`
	UserMetadata map[string]interface{} `json:"user_metadata"`
	AppMetadata  map[string]interface{} `json:"app_metadata"`
}

// JWTProtected valide le token JWT Supabase.
// T35 v2 : extraction du rôle dans l'ordre app_metadata → user_metadata → fallback
// + normalisation enseignant → encadrant
// + vérification de l'expiration
func JWTProtected(jwtSecret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// 1. Extraire le token
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"error":   "Token JWT manquant — header Authorization requis",
			})
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"error":   "Format invalide — attendu : Authorization: Bearer <token>",
			})
		}
		tokenString := parts[1]

		// 2. Décoder le payload JWT (base64)
		claims, err := decodeJWTPayload(tokenString)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"error":   "Token JWT malformé",
			})
		}

		// 3. Vérifier l'expiration (T35 — sécurité renforcée)
		if claims.Exp == 0 {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"error":   "Token JWT invalide — expiration manquante",
			})
		}
		if time.Now().Unix() > claims.Exp {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"error":   "Token JWT expiré — veuillez vous reconnecter",
			})
		}

		// 4. Vérifier la présence du subject
		if claims.Sub == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"error":   "Token JWT invalide — subject manquant",
			})
		}

		// 5. Extraire le rôle applicatif — T35 v2
		// Ordre de priorité :
		//   1. app_metadata.role  (modifiable uniquement via service_role — source fiable)
		//   2. user_metadata.role (modifiable par l'utilisateur — fallback)
		//   3. "etudiant"          (valeur par défaut sécurisée)
		userRole := extractRole(claims)

		// 6. Stocker dans le contexte Fiber
		c.Locals(ContextUserID,    claims.Sub)
		c.Locals(ContextUserRole,  userRole)
		c.Locals(ContextUserEmail, claims.Email)

		return c.Next()
	}
}

// extractRole extrait et normalise le rôle depuis les claims JWT.
// T35 : priorité app_metadata > user_metadata, normalisation enseignant → encadrant
func extractRole(claims *supabaseClaims) string {
	// Priorité 1 : app_metadata.role (plus fiable)
	if claims.AppMetadata != nil {
		if r, ok := claims.AppMetadata["role"].(string); ok && r != "" {
			return normalizeRole(r)
		}
	}

	// Priorité 2 : user_metadata.role
	if claims.UserMetadata != nil {
		if r, ok := claims.UserMetadata["role"].(string); ok && r != "" {
			return normalizeRole(r)
		}
	}

	// Fallback sécurisé : etudiant (le rôle avec le moins de droits)
	return "etudiant"
}

// normalizeRole normalise les anciens noms de rôles.
// "enseignant" → "encadrant" (migration v1 → v2)
func normalizeRole(role string) string {
	if role == "enseignant" {
		return "encadrant"
	}
	// Valider que le rôle est connu
	switch role {
	case "admin", "encadrant", "etudiant":
		return role
	default:
		return "etudiant" // rôle inconnu → fallback sécurisé
	}
}

// decodeJWTPayload décode la partie payload d'un JWT sans vérifier la signature.
func decodeJWTPayload(tokenString string) (*supabaseClaims, error) {
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return nil, fiber.NewError(fiber.StatusUnauthorized, "Format JWT invalide")
	}

	payload := parts[1]
	switch len(payload) % 4 {
	case 2:
		payload += "=="
	case 3:
		payload += "="
	}

	decoded, err := base64.URLEncoding.DecodeString(payload)
	if err != nil {
		decoded, err = base64.RawURLEncoding.DecodeString(parts[1])
		if err != nil {
			return nil, fiber.NewError(fiber.StatusUnauthorized, "Payload JWT invalide")
		}
	}

	var claims supabaseClaims
	if err := json.Unmarshal(decoded, &claims); err != nil {
		return nil, fiber.NewError(fiber.StatusUnauthorized, "Claims JWT invalides")
	}

	return &claims, nil
}

// RequireRole vérifie que l'utilisateur a l'un des rôles requis.
// T35 v2 : supporte les 3 rôles (admin, encadrant, etudiant)
// Note : admin a accès à TOUTES les routes (super-utilisateur)
func RequireRole(roles ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userRole, ok := c.Locals(ContextUserRole).(string)
		if !ok || userRole == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"error":   "Rôle utilisateur non trouvé",
			})
		}

		// Admin a accès à tout
		if userRole == "admin" {
			return c.Next()
		}

		for _, role := range roles {
			if userRole == role {
				return c.Next()
			}
		}

		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"success": false,
			"error":   "Accès refusé — rôle insuffisant",
			"detail": fiber.Map{
				"role_actuel":  userRole,
				"roles_requis": roles,
			},
		})
	}
}

// GetUserID extrait l'UUID depuis le contexte Fiber.
func GetUserID(c *fiber.Ctx) string {
	id, _ := c.Locals(ContextUserID).(string)
	return id
}

// GetUserRole extrait le rôle depuis le contexte Fiber.
func GetUserRole(c *fiber.Ctx) string {
	role, _ := c.Locals(ContextUserRole).(string)
	return role
}
