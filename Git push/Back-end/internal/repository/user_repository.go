package repository

import (
	"context"

	"github.com/google/uuid"
	"suivi-projets-backend/internal/domain"
)

// UserRepository définit le contrat de persistance pour l'entité User.
// L'implémentation concrète se trouve dans internal/repository/postgres/user_repository.go
//
// Note : La création de compte est déléguée à Supabase Auth.
// Ce repository gère uniquement le profil applicatif étendu (table public.profiles v2).
type UserRepository interface {

	// --- Étape 3 : Authentification & Middleware ---

	// GetByID récupère un utilisateur par son UUID (identique à l'UUID Supabase Auth).
	// Utilisé par le middleware JWT pour valider le profil et le rôle.
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Profile, error)

	// GetByEmail récupère un utilisateur par son email.
	// Utilisé lors de la synchronisation post-inscription Supabase Auth.
	GetByEmail(ctx context.Context, email string) (*domain.Profile, error)

	// --- Étape 4 : Gestion des membres d'équipe ---

	// Create insère le profil étendu d'un utilisateur après son inscription Supabase.
	// TODO Étape 4 : Déclenché par un webhook Supabase Auth (event: signup).
	Create(ctx context.Context, user *domain.Profile) error

	// --- Étape 8 : Dashboard enseignant ---

	// GetAll récupère la liste de tous les utilisateurs (usage : console enseignant).
	// TODO Étape 8 : Réservé au rôle "enseignant" via le middleware RLS.
	// GetAll(ctx context.Context) ([]*domain.Profile, error)
}
