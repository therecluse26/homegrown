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

// search godoc
//
// @Summary     Search across social, marketplace, or learning content
// @Tags        search
// @Produce     json
// @Security    BearerAuth
// @Param       q                query string  true  "Search query (min 2 chars)"
// @Param       scope            query string  true  "Search scope" Enums(social,marketplace,learning)
// @Param       cursor           query string  false "Pagination cursor"
// @Param       limit            query int     false "Results per page (default 20, max 50)"
// @Param       sort             query string  false "Sort order (marketplace only)" Enums(relevance,price_asc,price_desc,rating,recency)
// @Param       sub_scope        query string  false "Social sub-scope" Enums(families,groups,events)
// @Param       methodology_slug query string  false "Methodology slug filter (social scope)"
// @Success     200 {object} SearchResponse
// @Failure     400 {object} shared.AppError
// @Failure     401 {object} shared.AppError
// @Failure     422 {object} shared.AppError
// @Router      /search [get]
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
	if err := c.Validate(&params); err != nil {
		return shared.ValidationError(err)
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

// autocomplete godoc
//
// @Summary     Type-ahead suggestions for search
// @Tags        search
// @Produce     json
// @Security    BearerAuth
// @Param       q     query string  true  "Prefix query (min 1 char)"
// @Param       scope query string  false "Search scope" Enums(social,marketplace,learning)
// @Param       limit query int     false "Max suggestions (default 5, max 10)"
// @Success     200 {object} AutocompleteResponse
// @Failure     400 {object} shared.AppError
// @Failure     401 {object} shared.AppError
// @Router      /search/autocomplete [get]
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
	if err := c.Validate(&params); err != nil {
		return shared.ValidationError(err)
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

// suggestions godoc
//
// @Summary     AI-powered search suggestions (Phase 3 — not yet available)
// @Tags        search
// @Produce     json
// @Security    BearerAuth
// @Success     200 {object} map[string][]any
// @Router      /search/suggestions [get]
// GET /v1/search/suggestions — returns empty list until Typesense is wired. [12-search §4.3]
func (h *Handler) suggestions(c echo.Context) error {
	// TODO(phase-3): wire Typesense suggestions query. Returning empty list for now.
	return c.JSON(http.StatusOK, map[string]any{"suggestions": []any{}})
}

// mapSearchError maps any error to an Echo-compatible HTTP error.
func mapSearchError(err error) error {
	var se *SearchError
	if errors.As(err, &se) {
		return se.toAppError()
	}
	return err
}
