package response

import (
	"github.com/gofiber/fiber/v2"
)

// OK retourne une réponse 200 avec données.
func OK(c *fiber.Ctx, data interface{}) error {
	return c.JSON(fiber.Map{"success": true, "data": data})
}

// Created retourne une réponse 201 avec données.
func Created(c *fiber.Ctx, data interface{}) error {
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"success": true, "data": data})
}

// NoContent retourne une réponse 204 sans corps.
func NoContent(c *fiber.Ctx) error {
	return c.Status(fiber.StatusNoContent).Send(nil)
}

// BadRequest retourne 400 avec message d'erreur.
func BadRequest(c *fiber.Ctx, msg string) error {
	return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
		"success": false, "error": msg,
	})
}

// Unauthorized retourne 401.
func Unauthorized(c *fiber.Ctx, msg string) error {
	if msg == "" {
		msg = "Authentification requise"
	}
	return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
		"success": false, "error": msg,
	})
}

// Forbidden retourne 403.
func Forbidden(c *fiber.Ctx, msg string) error {
	if msg == "" {
		msg = "Accès refusé — droits insuffisants"
	}
	return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
		"success": false, "error": msg,
	})
}

// NotFound retourne 404.
func NotFound(c *fiber.Ctx, ressource string) error {
	if ressource == "" {
		ressource = "Ressource"
	}
	return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
		"success": false, "error": ressource + " introuvable",
	})
}

// Conflict retourne 409 — doublon ou contrainte violée.
func Conflict(c *fiber.Ctx, msg string) error {
	return c.Status(fiber.StatusConflict).JSON(fiber.Map{
		"success": false, "error": msg,
	})
}

// ServerError retourne 500 avec log de l'erreur interne.
func ServerError(c *fiber.Ctx, err error, context string) error {
	// On logue l'erreur technique côté serveur mais on ne l'expose pas au client
	if err != nil {
		c.Context().Logger().Printf("[ERROR] %s : %v", context, err)
	}
	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
		"success": false,
		"error":   "Erreur serveur — veuillez réessayer",
	})
}

// Unavailable retourne 503 — service ou BDD indisponible.
func Unavailable(c *fiber.Ctx) error {
	return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
		"success": false, "error": "Service temporairement indisponible",
	})
}
