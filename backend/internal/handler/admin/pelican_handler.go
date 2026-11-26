package admin

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// PelicanHandler exposes the periodic pelican degradation benchmark.
type PelicanHandler struct {
	svc *service.PelicanCheckService
}

func NewPelicanHandler(svc *service.PelicanCheckService) *PelicanHandler {
	return &PelicanHandler{svc: svc}
}

// Overview GET /admin/pelican
func (h *PelicanHandler) Overview(c *gin.Context) {
	overview, err := h.svc.Overview(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, overview)
}

// Run POST /admin/pelican/accounts/:id/run
func (h *PelicanHandler) Run(c *gin.Context) {
	accountID, ok := parsePelicanAccountID(c)
	if !ok {
		return
	}
	var req struct {
		Model string `json:"model"`
	}
	_ = c.ShouldBindJSON(&req)
	if err := h.svc.StartCheck(accountID, strings.TrimSpace(req.Model)); err != nil {
		if errors.Is(err, service.ErrPelicanCheckRunning) {
			response.Error(c, http.StatusConflict, err.Error())
			return
		}
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"status": "queued", "account_id": accountID})
}

// Checks GET /admin/pelican/accounts/:id/checks
func (h *PelicanHandler) Checks(c *gin.Context) {
	accountID, ok := parsePelicanAccountID(c)
	if !ok {
		return
	}
	limit := 20
	if raw := strings.TrimSpace(c.Query("limit")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			limit = parsed
		}
	}
	checks, err := h.svc.ListChecks(c.Request.Context(), accountID, limit)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, checks)
}

func parsePelicanAccountID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "invalid account id")
		return 0, false
	}
	return id, true
}
