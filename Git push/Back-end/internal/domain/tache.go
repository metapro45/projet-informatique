package domain

import (
	"time"

	"github.com/google/uuid"
)

// TacheOrdre représente l'ordre d'une tâche dans une colonne Kanban.
// Utilisé par l'endpoint de réordonnancement (T53).
type TacheOrdre struct {
	ID       uuid.UUID   `json:"id"`
	Position int         `json:"position"`
	Statut   StatutTache `json:"statut"`
	EquipeID uuid.UUID   `json:"-"` // validation interne uniquement
}

// StatutTache représente la colonne Kanban d'une tâche.
// Valeurs attendues en BDD : "todo" | "en_cours" | "termine"
type StatutTache string

const (
	StatutTacheTodo    StatutTache = "todo"
	StatutTacheEnCours StatutTache = "en_cours"
	StatutTacheTermine StatutTache = "termine"
)

// PrioriteTache représente le niveau d'urgence d'une tâche.
// Valeurs attendues en BDD : "basse" | "moyenne" | "haute"
type PrioriteTache string

const (
	PrioriteBasse   PrioriteTache = "basse"
	PrioriteMoyenne PrioriteTache = "moyenne"
	PrioriteHaute   PrioriteTache = "haute"
)

// Tache mappe la table `public.taches`.
// Représente une carte sur le tableau Kanban de l'équipe.
// Étape 5 : le changement de StatutTache est l'endpoint central du Kanban.
//
// Note : DateEcheance est de type *Date (pas *time.Time) car la colonne
// SQL est de type `date` — sérialisation JSON au format "YYYY-MM-DD".
type Tache struct {
	ID           uuid.UUID      `json:"id" db:"id"`
	Titre        string         `json:"titre" db:"titre"`
	Description  *string        `json:"description,omitempty" db:"description"`
	Statut       StatutTache    `json:"statut" db:"statut"`
	Priorite     *PrioriteTache `json:"priorite,omitempty" db:"priorite"`
	DateEcheance *Date          `json:"date_echeance,omitempty" db:"date_echeance"`
	EquipeID     *uuid.UUID     `json:"equipe_id,omitempty" db:"equipe_id"`
	AssigneeID   *uuid.UUID     `json:"assignee_id,omitempty" db:"assignee_id"`
	Position     int            `json:"position" db:"position"`
	CreatedAt    *time.Time     `json:"created_at,omitempty" db:"created_at"`

	// T116 — Champs calculés (non stockés en BDD)
	IsOverdue     bool `json:"is_overdue"`                          // true si date_echeance < aujourd'hui et statut != termine
	JoursRestants *int `json:"jours_restants,omitempty"`            // jours avant l'échéance (négatif si en retard)
}

// ComputeOverdue calcule les champs is_overdue et jours_restants.
// Appelé après chaque scan depuis la BDD.
func (t *Tache) ComputeOverdue() {
	if t.DateEcheance == nil {
		t.IsOverdue = false
		t.JoursRestants = nil
		return
	}
	if t.Statut == StatutTacheTermine {
		t.IsOverdue = false
		t.JoursRestants = nil
		return
	}
	now := time.Now().Truncate(24 * time.Hour)
	echeance := t.DateEcheance.Time.Truncate(24 * time.Hour)
	diff := int(echeance.Sub(now).Hours() / 24)
	t.JoursRestants = &diff
	t.IsOverdue = diff < 0
}
