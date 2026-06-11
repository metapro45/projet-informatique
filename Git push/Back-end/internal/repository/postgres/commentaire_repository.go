package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"suivi-projets-backend/internal/domain"
)

type CommentaireRepository struct{ db *sql.DB }

func NewCommentaireRepository(db *sql.DB) *CommentaireRepository {
	return &CommentaireRepository{db: db}
}

// GetByTacheID retourne tous les commentaires d'une tâche avec le nom de l'auteur.
func (r *CommentaireRepository) GetByTacheID(ctx context.Context, tacheID uuid.UUID) ([]*domain.Commentaire, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT c.id, c.tache_id, c.auteur_id, c.contenu,
		       c.created_at, c.updated_at,
		       COALESCE(p.nom, ''), COALESCE(p.prenom, '')
		FROM tache_commentaires c
		LEFT JOIN profiles p ON p.id = c.auteur_id
		WHERE c.tache_id = $1
		ORDER BY c.created_at ASC
	`, tacheID)
	if err != nil {
		return nil, fmt.Errorf("commentaire.GetByTacheID: %w", err)
	}
	defer rows.Close()

	var commentaires []*domain.Commentaire
	for rows.Next() {
		c := &domain.Commentaire{}
		var createdAt, updatedAt sql.NullTime
		err := rows.Scan(
			&c.ID, &c.TacheID, &c.AuteurID, &c.Contenu,
			&createdAt, &updatedAt,
			&c.AuteurNom, &c.AuteurPrenom,
		)
		if err != nil {
			return nil, fmt.Errorf("commentaire.GetByTacheID scan: %w", err)
		}
		if createdAt.Valid { t := createdAt.Time.UTC(); c.CreatedAt = &t }
		if updatedAt.Valid { t := updatedAt.Time.UTC(); c.UpdatedAt = &t }
		commentaires = append(commentaires, c)
	}
	return commentaires, nil
}

// Create insère un nouveau commentaire.
func (r *CommentaireRepository) Create(ctx context.Context, c *domain.Commentaire) error {
	c.ID = uuid.New()
	row := r.db.QueryRowContext(ctx, `
		INSERT INTO tache_commentaires (id, tache_id, auteur_id, contenu)
		VALUES ($1, $2, $3, $4)
		RETURNING id, tache_id, auteur_id, contenu, created_at, updated_at
	`, c.ID, c.TacheID, c.AuteurID, c.Contenu)

	var createdAt, updatedAt sql.NullTime
	err := row.Scan(
		&c.ID, &c.TacheID, &c.AuteurID, &c.Contenu,
		&createdAt, &updatedAt,
	)
	if err != nil {
		return fmt.Errorf("commentaire.Create: %w", err)
	}
	if createdAt.Valid { t := createdAt.Time.UTC(); c.CreatedAt = &t }
	if updatedAt.Valid { t := updatedAt.Time.UTC(); c.UpdatedAt = &t }
	return nil
}

// Delete supprime un commentaire par son ID.
func (r *CommentaireRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.ExecContext(ctx,
		"DELETE FROM tache_commentaires WHERE id=$1", id)
	if err != nil {
		return fmt.Errorf("commentaire.Delete: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("commentaire.Delete: commentaire %s introuvable", id)
	}
	return nil
}

// GetByID retourne un commentaire par son ID (pour vérifier l'auteur avant suppression).
func (r *CommentaireRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Commentaire, error) {
	row := r.db.QueryRowContext(ctx,
		"SELECT id, tache_id, auteur_id, contenu, created_at, updated_at FROM tache_commentaires WHERE id=$1", id)
	c := &domain.Commentaire{}
	var createdAt, updatedAt sql.NullTime
	err := row.Scan(&c.ID, &c.TacheID, &c.AuteurID, &c.Contenu, &createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("commentaire.GetByID: %w", err)
	}
	if createdAt.Valid { t := createdAt.Time.UTC(); c.CreatedAt = &t }
	if updatedAt.Valid { t := updatedAt.Time.UTC(); c.UpdatedAt = &t }
	return c, nil
}
