package repository

import (
	"context"

	"github.com/google/uuid"
	"suivi-projets-backend/internal/domain"
)

// MembreRepository définit le contrat de persistance pour l'entité Membre.
// Gère la table de jointure N-N entre users et equipes.
// L'implémentation concrète se trouve dans internal/repository/postgres/membre_repository.go
//
// TODO Étape 4 : Implémenter l'ensemble de ces méthodes.
type MembreRepository interface {

	// --- Étape 4 : Gestion des membres ---

	// AddMembre ajoute un utilisateur à une équipe avec un rôle défini.
	AddMembre(ctx context.Context, membre *domain.Membre) error

	// RemoveMembre retire un utilisateur d'une équipe.
	RemoveMembre(ctx context.Context, equipeID uuid.UUID, userID uuid.UUID) error

	// GetByEquipeID liste tous les membres d'une équipe.
	GetByEquipeID(ctx context.Context, equipeID uuid.UUID) ([]*domain.Membre, error)

	// GetByUserID liste toutes les équipes auxquelles appartient un utilisateur.
	// TODO Étape 8 : Utilisé pour construire la vue "mes équipes" de l'étudiant.
	// GetByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.Membre, error)
}
