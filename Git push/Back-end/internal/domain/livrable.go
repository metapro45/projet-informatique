package domain

import (
	"time"

	"github.com/google/uuid"
)

// Livrable mappe la table `public.livrables`.
// Représente un fichier déposé par une équipe pour un jalon donné.
// Le fichier physique est stocké dans Supabase Storage (référencé par FichierURL).
//
// Note sur le champ Note : la colonne SQL est de type `numeric`.
// On conserve *float64 en Go — l'arrondi à 2 décimales est appliqué
// au niveau du handler (NoterLivrable) avant toute sérialisation JSON.
// Exemple : math.Round(note*100)/100 → 14.5, jamais 14.499999...
//
// TODO Étape 6 : Connecter la passerelle Supabase Storage pour le téléversement.
// TODO Étape 6 : Implémenter la notation (Note + Commentaire) par l'enseignant.
type Livrable struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	Nom         string     `json:"nom" db:"nom"`
	FichierURL  string     `json:"fichier_url" db:"fichier_url"`
	EquipeID    *uuid.UUID `json:"equipe_id,omitempty" db:"equipe_id"`
	JalonID     *uuid.UUID `json:"jalon_id,omitempty" db:"jalon_id"`
	Note        *float64   `json:"note,omitempty" db:"note"`
	Commentaire *string    `json:"commentaire,omitempty" db:"commentaire"`
	DeposePar   *uuid.UUID `json:"depose_par,omitempty" db:"depose_par"`
	CreatedAt   *time.Time `json:"created_at,omitempty" db:"created_at"`
}
