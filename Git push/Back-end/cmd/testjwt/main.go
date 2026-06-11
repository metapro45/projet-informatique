//go:build ignore

// Outil de test JWT — génère des tokens pour tester le middleware
// Usage : go run ./cmd/testjwt/main.go
package main

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func main() {
	// Remplace par ton SUPABASE_JWT_SECRET du .env
	jwtSecret := "REMPLACE_PAR_TON_JWT_SECRET"

	tokenEtudiant := genToken(jwtSecret, map[string]interface{}{
		"sub":   "b0000002-0000-0000-0000-000000000001",
		"email": "alice.bernard@etudiant.fr",
		"user_metadata": map[string]interface{}{
			"role": "etudiant",
		},
	})

	tokenEnseignant := genToken(jwtSecret, map[string]interface{}{
		"sub":   "a0000001-0000-0000-0000-000000000001",
		"email": "marie.durand@universite.fr",
		"user_metadata": map[string]interface{}{
			"role": "enseignant",
		},
	})

	fmt.Println("TOKEN ETUDIANT (alice.bernard) :")
	fmt.Println(tokenEtudiant)
	fmt.Println()
	fmt.Println("TOKEN ENSEIGNANT (marie.durand) :")
	fmt.Println(tokenEnseignant)
}

func genToken(secret string, claims map[string]interface{}) string {
	mc := jwt.MapClaims{
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(24 * time.Hour).Unix(),
		"iss": "supabase",
	}
	for k, v := range claims {
		mc[k] = v
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, mc)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		return "ERREUR: " + err.Error()
	}
	return signed
}
