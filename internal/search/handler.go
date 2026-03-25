package search

import (
	"errors"
	"net/http"
	"time"

	"github.com/homegrown-academy/homegrown-academy/internal/middleware"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
	"github.com/labstack/echo/v4"
)

// rateLimitDeps matches the unexported middleware.rateLimitDeps interface.
// Go structural typing lets *AppState satisfy both this and the middleware interface.
type rateLimitDeps interface {
	GetCache() shared.Cache
}

// Handler holds the search HTTP handler dependencies.
type Handler struct {
	svc SearchService
	rl  rateLimitDeps
}

// NewHandler creates a new search Handler.
func NewHandler(svc SearchService, rl rateLimitDeps) *Handler {
	return &Handler{svc: svc, rl: rl}
}

// Register registers all search routes on the authenticated route group.
// All search endpoints require authentication. [12-search §4]
// Rate limit: 60 req/min per authenticated user [ARCH §10.6]
func (h *Handler) Register(authGroup *echo.Group) {
	search := authGroup.Group("/search")
	search.Use(middleware.RateLimit(h.rl, 60, 60*time.Second))
	search.GET("", h.search)
	search.GET("/autocomplete", h.autocomplete)
	search.GET("/suggestions", h.suggestions)
}

// GET /v1/search
func (h *Handler) search(c echo.Context) error {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return err
	}
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}

	var params SearchParams
	if bindErr := c.Bind(&params); bindErr != nil {
		return shared.ErrBadRequest("invalid query parameters")
	}

	if params.Q == "" {
		return shared.ErrValidation("q is required")
	}

	resp, err := h.svc.Search(c.Request().Context(), auth, &scope, &params)
	if err != nil {
		return mapSearchError(err)
	}

	return c.JSON(http.StatusOK, resp)
}

// GET /v1/search/autocomplete
func (h *Handler) autocomplete(c echo.Context) error {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return err
	}
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}

	var params AutocompleteParams
	if bindErr := c.Bind(&params); bindErr != nil {
		return shared.ErrBadRequest("invalid query parameters")
	}

	if params.Q == "" {
		return shared.ErrBadRequest("q is required")
	}

	resp, err := h.svc.Autocomplete(c.Request().Context(), auth, &scope, &params)
	if err != nil {
		return mapSearchError(err)
	}

	return c.JSON(http.StatusOK, resp)
}

// GET /v1/search/suggestions — Phase 3 stub [12-search §4.3]
// Returns 501 until the recs:: domain is implemented.
func (h *Handler) suggestions(c echo.Context) error {
	return c.JSON(http.StatusNotImplemented, map[string]string{
		"error": "search suggestions are not yet available",
	})
}

// mapSearchError maps any error to an Echo-compatible HTTP error.
func mapSearchError(err error) error {
	var se *SearchError
	if errors.As(err, &se) {
		return se.toAppError()
	}
	return err
}
