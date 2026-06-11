package repository

import (
	"context"

	"github.com/google/uuid"
	"suivi-projets-backend/internal/domain"
)

// TacheRepository définit le contrat de persistance pour l'entité Tache.
// L'implémentation concrète se trouve dans internal/repository/postgres/tache_repository.go
type TacheRepository interface {

	// --- Étape 5 : Moteur Kanban ---

	// Create insère une nouvelle tâche dans le Kanban d'une équipe.
	Create(ctx context.Context, tache *domain.Tache) error

	// GetByID récupère une tâche par son UUID.
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Tache, error)

	// GetByEquipeID liste toutes les tâches d'une équipe (toutes colonnes confondues).
	// Utilisé pour le rendu initial du tableau Kanban.
	GetByEquipeID(ctx context.Context, equipeID uuid.UUID) ([]*domain.Tache, error)

	// UpdateStatut met à jour uniquement le champ `statut` d'une tâche.
	// C'est l'endpoint critique interceptant le drag-and-drop Kanban (Étape 5 & 9).
	UpdateStatut(ctx context.Context, id uuid.UUID, statut domain.StatutTache) error

	// Update met à jour l'ensemble des champs d'une tâche (titre, description, priorité...).
	Update(ctx context.Context, tache *domain.Tache) error

	// Delete supprime une tâche.
	Delete(ctx context.Context, id uuid.UUID) error

	// --- Étape 8 : Dashboard étudiant ---

	// GetByAssigneeID liste les tâches assignées à un utilisateur précis.
	// TODO Étape 8 : Utilisé pour la vue "mes tâches" de l'étudiant.
	// GetByAssigneeID(ctx context.Context, assigneeID uuid.UUID) ([]*domain.Tache, error)
}
