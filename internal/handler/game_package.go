package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	"control-panel/internal/application/services"
	"control-panel/internal/domain/tenant"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type GamePackageHandler struct{ service *services.GamePackageService }

func NewGamePackageHandler(service *services.GamePackageService) *GamePackageHandler {
	return &GamePackageHandler{service: service}
}

func (h *GamePackageHandler) Register(c *gin.Context) {
	raw, err := io.ReadAll(c.Request.Body)
	if err != nil || len(strings.TrimSpace(string(raw))) == 0 {
		respondError(c, http.StatusBadRequest, "request body must be a GamePackageManifest")
		return
	}
	detail, err := h.service.Register(tenant.GetTenantID(c), actorID(c), json.RawMessage(raw))
	if err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	respondCreated(c, detail)
}

// EnsureAcceptanceFixtures installs the small, domain-neutral M0 examples into
// the caller's tenant. It is explicit (never a hidden GET side effect) and
// idempotent, so product acceptance works in builtin and Casdoor deployments.
func (h *GamePackageHandler) EnsureAcceptanceFixtures(c *gin.Context) {
	tenantID := tenant.GetTenantID(c)
	if err := h.service.EnsureM0AcceptancePackages(tenantID); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	result, err := h.service.List(tenantID, nil)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, result)
}

func (h *GamePackageHandler) List(c *gin.Context) {
	var enabled *bool
	if raw := strings.TrimSpace(c.Query("enabled")); raw != "" {
		v, err := strconv.ParseBool(raw)
		if err != nil {
			respondError(c, http.StatusBadRequest, "enabled must be true or false")
			return
		}
		enabled = &v
	}
	result, err := h.service.List(tenant.GetTenantID(c), enabled)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, result)
}
func (h *GamePackageHandler) Get(c *gin.Context) {
	detail, err := h.service.Get(tenant.GetTenantID(c), c.Param("id"))
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			respondError(c, http.StatusNotFound, "game package not found")
		} else {
			respondError(c, http.StatusInternalServerError, err.Error())
		}
		return
	}
	respondSuccess(c, detail)
}
func (h *GamePackageHandler) SetEnabled(c *gin.Context) {
	var req struct {
		Enabled *bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Enabled == nil {
		respondError(c, http.StatusBadRequest, "enabled is required")
		return
	}
	item, err := h.service.SetEnabled(tenant.GetTenantID(c), c.Param("id"), *req.Enabled)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			respondError(c, http.StatusNotFound, "game package not found")
		} else {
			respondError(c, http.StatusInternalServerError, err.Error())
		}
		return
	}
	respondSuccess(c, item)
}
func (h *GamePackageHandler) TestRun(c *gin.Context) {
	var req services.GameTestRunInput
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.service.TestRun(tenant.GetTenantID(c), c.Param("id"), req)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			respondError(c, http.StatusNotFound, "game package not found")
		} else {
			respondError(c, http.StatusBadRequest, err.Error())
		}
		return
	}
	respondSuccess(c, result)
}
