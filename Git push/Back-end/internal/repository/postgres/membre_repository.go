package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"suivi-projets-backend/internal/domain"
)

type MembreRepository struct{ db *sql.DB }

func NewMembreRepository(db *sql.DB) *MembreRepository {
	return &MembreRepository{db: db}
}

func (r *MembreRepository) GetByEquipeID(ctx context.Context, equipeID uuid.UUID) ([]*domain.Membre, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT m.id, m.equipe_id, m.user_id, m.role_equipe, m.joined_at,
		       COALESCE(p.nom,''), COALESCE(p.prenom,''), p.email, COALESCE(p.role::text,'')
		FROM membres m
		LEFT JOIN profiles p ON p.id = m.user_id
		WHERE m.equipe_id = $1
		ORDER BY m.joined_at ASC
	`, equipeID)
	if err != nil {
		return nil, fmt.Errorf("membre.GetByEquipeID: %w", err)
	}
	defer rows.Close()

	var membres []*domain.Membre
	for rows.Next() {
		m := &domain.Membre{}
		var joinedAt sql.NullTime
		var nom, prenom, email, role string
		err := rows.Scan(
			&m.ID, &m.EquipeID, &m.UserID, &m.RoleEquipe, &joinedAt,
			&nom, &prenom, &email, &role,
		)
		if err != nil {
			return nil, fmt.Errorf("membre.GetByEquipeID scan: %w", err)
		}
		if joinedAt.Valid { t := joinedAt.Time.UTC(); m.JoinedAt = &t }
		// Enrichissement profil
		m.Nom    = nom
		m.Prenom = prenom
		m.Email  = email
		m.Role   = role
		membres = append(membres, m)
	}
	return membres, nil
}

func (r *MembreRepository) Add(ctx context.Context, equipeID, userID uuid.UUID) (*domain.Membre, error) {
	m := &domain.Membre{}
	var joinedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, `
		INSERT INTO membres (id, equipe_id, user_id, role_equipe)
		VALUES (gen_random_uuid(), $1, $2, 'membre')
		ON CONFLICT (equipe_id, user_id) DO UPDATE SET role_equipe = 'membre'
		RETURNING id, equipe_id, user_id, role_equipe, joined_at
	`, equipeID, userID).Scan(
		&m.ID, &m.EquipeID, &m.UserID, &m.RoleEquipe, &joinedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("membre.Add: %w", err)
	}
	if joinedAt.Valid { t := joinedAt.Time.UTC(); m.JoinedAt = &t }
	return m, nil
}

func (r *MembreRepository) Remove(ctx context.Context, equipeID, userID uuid.UUID) error {
	result, err := r.db.ExecContext(ctx,
		"DELETE FROM membres WHERE equipe_id=$1 AND user_id=$2", equipeID, userID)
	if err != nil {
		return fmt.Errorf("membre.Remove: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("membre.Remove: membre introuvable")
	}
	return nil
}
