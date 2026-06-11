package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"suivi-projets-backend/internal/domain"
)

type TacheRepository struct{ db *sql.DB }

func NewTacheRepository(db *sql.DB) *TacheRepository {
	return &TacheRepository{db: db}
}

const sqlGetTachesByEquipe = `
	SELECT id, titre, description, statut, priorite,
	       date_echeance, equipe_id, assignee_id, position, created_at
	FROM taches
	WHERE equipe_id = $1
	ORDER BY statut, position ASC, created_at ASC
`
const sqlGetTacheByID = `
	SELECT id, titre, description, statut, priorite,
	       date_echeance, equipe_id, assignee_id, position, created_at
	FROM taches WHERE id = $1
`
const sqlCreateTache = `
	INSERT INTO taches (id, titre, description, statut, priorite,
	                    date_echeance, equipe_id, assignee_id, position)
	VALUES ($1,$2,$3,$4,$5,$6,$7,$8,
	        COALESCE((SELECT MAX(position) FROM taches WHERE equipe_id=$7 AND statut=$4), 0) + 1)
	RETURNING id, titre, description, statut, priorite,
	          date_echeance, equipe_id, assignee_id, position, created_at
`
const sqlUpdateTache = `
	UPDATE taches
	SET titre=$2, description=$3, priorite=$4,
	    date_echeance=$5, assignee_id=$6
	WHERE id=$1
	RETURNING id, titre, description, statut, priorite,
	          date_echeance, equipe_id, assignee_id, position, created_at
`
const sqlUpdateStatut = `
	UPDATE taches SET statut=$2,
	    position = COALESCE((SELECT MAX(position) FROM taches WHERE equipe_id=taches.equipe_id AND statut=$2), 0) + 1
	WHERE id=$1
	RETURNING id, titre, description, statut, priorite,
	          date_echeance, equipe_id, assignee_id, position, created_at
`
const sqlDeleteTache = `DELETE FROM taches WHERE id=$1`

func (r *TacheRepository) GetByEquipeID(ctx context.Context, equipeID uuid.UUID) ([]*domain.Tache, error) {
	rows, err := r.db.QueryContext(ctx, sqlGetTachesByEquipe, equipeID)
	if err != nil {
		return nil, fmt.Errorf("tache.GetByEquipeID: %w", err)
	}
	defer rows.Close()
	var taches []*domain.Tache
	for rows.Next() {
		t, err := scanTache(rows)
		if err != nil {
			return nil, fmt.Errorf("tache.GetByEquipeID scan: %w", err)
		}
		taches = append(taches, t)
	}
	return taches, nil
}

func (r *TacheRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Tache, error) {
	row := r.db.QueryRowContext(ctx, sqlGetTacheByID, id)
	t, err := scanTacheRow(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("tache.GetByID: %w", err)
	}
	return t, nil
}

func (r *TacheRepository) Create(ctx context.Context, t *domain.Tache) error {
	t.ID = uuid.New()
	row := r.db.QueryRowContext(ctx, sqlCreateTache,
		t.ID, t.Titre, t.Description, t.Statut, t.Priorite,
		t.DateEcheance, t.EquipeID, t.AssigneeID,
	)
	created, err := scanTacheRow(row)
	if err != nil {
		return fmt.Errorf("tache.Create: %w", err)
	}
	*t = *created
	return nil
}

func (r *TacheRepository) Update(ctx context.Context, t *domain.Tache) error {
	row := r.db.QueryRowContext(ctx, sqlUpdateTache,
		t.ID, t.Titre, t.Description, t.Priorite,
		t.DateEcheance, t.AssigneeID,
	)
	updated, err := scanTacheRow(row)
	if err == sql.ErrNoRows {
		return fmt.Errorf("tache.Update: tache %s introuvable", t.ID)
	}
	if err != nil {
		return fmt.Errorf("tache.Update: %w", err)
	}
	*t = *updated
	return nil
}

func (r *TacheRepository) UpdateStatut(ctx context.Context, id uuid.UUID, statut domain.StatutTache) (*domain.Tache, error) {
	row := r.db.QueryRowContext(ctx, sqlUpdateStatut, id, statut)
	t, err := scanTacheRow(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("tache.UpdateStatut: %w", err)
	}
	return t, nil
}

func (r *TacheRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.ExecContext(ctx, sqlDeleteTache, id)
	if err != nil {
		return fmt.Errorf("tache.Delete: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("tache.Delete: tache %s introuvable", id)
	}
	return nil
}


// ReorderValidated met à jour les positions avec validation d'appartenance.
// Format M4 : position = index du tableau task_ids envoyé par M2.
// Vérifie que chaque tâche appartient bien à equipe_id + statut avant de mettre à jour.
func (r *TacheRepository) ReorderValidated(
	ctx context.Context,
	equipeID uuid.UUID,
	statut domain.StatutTache,
	ordres []domain.TacheOrdre,
) error {
	// 1. Récupérer les IDs valides pour cette équipe + statut
	rows, err := r.db.QueryContext(ctx,
		"SELECT id FROM taches WHERE equipe_id=$1 AND statut=$2",
		equipeID, statut,
	)
	if err != nil {
		return fmt.Errorf("tache.ReorderValidated query: %w", err)
	}
	defer rows.Close()

	valides := make(map[uuid.UUID]bool)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err == nil {
			valides[id] = true
		}
	}

	// 2. Valider que tous les IDs envoyés sont valides
	for _, o := range ordres {
		if !valides[o.ID] {
			return fmt.Errorf("validation")
		}
	}

	// 3. Mettre à jour les positions (position = index du tableau)
	for _, o := range ordres {
		_, err := r.db.ExecContext(ctx,
			"UPDATE taches SET position=$2 WHERE id=$1",
			o.ID, o.Position,
		)
		if err != nil {
			return fmt.Errorf("tache.ReorderValidated update id=%s: %w", o.ID, err)
		}
	}
	return nil
}
// ── Scan helpers ──────────────────────────────────────────────────────────────

func scanTache(rows *sql.Rows) (*domain.Tache, error) { return scanTacheFields(rows) }
func scanTacheRow(row *sql.Row) (*domain.Tache, error) { return scanTacheFields(row) }

func scanTacheFields(s scanRow) (*domain.Tache, error) {
	t := &domain.Tache{}
	var description sql.NullString
	var priorite    sql.NullString
	var dateEch     sql.NullTime
	var equipeID    uuid.NullUUID
	var assigneeID  uuid.NullUUID
	var createdAt   sql.NullTime

	err := s.Scan(
		&t.ID, &t.Titre, &description, &t.Statut, &priorite,
		&dateEch, &equipeID, &assigneeID, &t.Position, &createdAt,
	)
	if err != nil {
		return nil, err
	}
	if description.Valid { t.Description = &description.String }
	if priorite.Valid {
		p := domain.PrioriteTache(priorite.String)
		t.Priorite = &p
	}
	if dateEch.Valid {
		d := domain.Date{Time: dateEch.Time}
		t.DateEcheance = &d
	}
	if equipeID.Valid  { t.EquipeID   = &equipeID.UUID }
	if assigneeID.Valid { t.AssigneeID = &assigneeID.UUID }
	if createdAt.Valid  {
		ct := createdAt.Time.UTC()
		t.CreatedAt = &ct
	}

	// T116 — Calculer is_overdue et jours_restants
	t.ComputeOverdue()

	return t, nil
}
