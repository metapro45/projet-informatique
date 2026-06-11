package repository

import (
	"context"

	"github.com/google/uuid"
	"suivi-projets-backend/internal/domain"
)

// JalonRepository définit le contrat de persistance pour l'entité Jalon.
// L'implémentation concrète se trouve dans internal/repository/postgres/jalon_repository.go
//
// TODO Étape 5 : Implémenter la gestion des jalons dans le contexte du suivi projet.
type JalonRepository interface {

	// --- Étape 5 : Gestion des jalons ---

	// Create insère un nouveau jalon sur un projet. Réservé à l'enseignant.
	Create(ctx context.Context, jalon *domain.Jalon) error

	// GetByID récupère un jalon par son UUID.
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Jalon, error)

	// GetByProjetID liste tous les jalons d'un projet, ordonnés par date_limite ASC.
	GetByProjetID(ctx context.Context, projetID uuid.UUID) ([]*domain.Jalon, error)

	// Update met à jour les informations d'un jalon (titre, description, date).
	Update(ctx context.Context, jalon *domain.Jalon) error

	// Delete supprime un jalon. Attention : cascade sur les livrables associés.
	Delete(ctx context.Context, id uuid.UUID) error
}
