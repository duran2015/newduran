package handler

import (
	"errors"
	"net/http"

	"control-panel/internal/application/services"
	"control-panel/internal/domain/run"
	"control-panel/internal/domain/tenant"
	"github.com/gin-gonic/gin"
)

type GameSessionHandler struct{ service *services.GameSessionService }

func NewGameSessionHandler(service *services.GameSessionService) *GameSessionHandler {
	return &GameSessionHandler{service: service}
}

func writeGameSessionError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, services.ErrGameSessionNotFound), errors.Is(err, run.ErrNotFound):
		respondError(c, http.StatusNotFound, err.Error())
	case errors.Is(err, services.ErrGameDirectorRequired):
		respondError(c, http.StatusForbidden, err.Error())
	case errors.Is(err, run.ErrConflict):
		respondError(c, http.StatusConflict, err.Error())
	case errors.Is(err, run.ErrInvalidTransition):
		respondError(c, http.StatusUnprocessableEntity, err.Error())
	default:
		respondError(c, http.StatusBadRequest, err.Error())
	}
}

func (h *GameSessionHandler) Create(c *gin.Context) {
	d, err := h.service.Create(tenant.GetTenantID(c), c.Param("id"), services.CreateGameSessionInput{CreatedBy: actorID(c)})
	if err != nil {
		writeGameSessionError(c, err)
		return
	}
	respondCreated(c, d)
}
func (h *GameSessionHandler) List(c *gin.Context) {
	rows, err := h.service.List(tenant.GetTenantID(c), c.Param("id"))
	if err != nil {
		writeGameSessionError(c, err)
		return
	}
	respondSuccess(c, rows)
}
func (h *GameSessionHandler) Get(c *gin.Context) {
	d, err := h.service.Get(tenant.GetTenantID(c), c.Param("id"), c.Param("sessionId"))
	if err != nil {
		writeGameSessionError(c, err)
		return
	}
	respondSuccess(c, d)
}
func (h *GameSessionHandler) Advance(c *gin.Context) {
	var in services.AdvanceGameSessionInput
	if err := c.ShouldBindJSON(&in); err != nil {
		respondError(c, 400, err.Error())
		return
	}
	d, err := h.service.Advance(tenant.GetTenantID(c), c.Param("id"), c.Param("sessionId"), in)
	if err != nil {
		writeGameSessionError(c, err)
		return
	}
	respondSuccess(c, d)
}
func (h *GameSessionHandler) Pause(c *gin.Context)  { h.lifecycle(c, "pause") }
func (h *GameSessionHandler) Resume(c *gin.Context) { h.lifecycle(c, "resume") }
func (h *GameSessionHandler) End(c *gin.Context)    { h.lifecycle(c, "end") }
func (h *GameSessionHandler) lifecycle(c *gin.Context, action string) {
	var in struct {
		ActorCharacterID string `json:"actorCharacterId"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		respondError(c, 400, err.Error())
		return
	}
	var d *services.GameSessionDetail
	var err error
	switch action {
	case "pause":
		d, err = h.service.Pause(tenant.GetTenantID(c), c.Param("id"), c.Param("sessionId"), in.ActorCharacterID)
	case "resume":
		d, err = h.service.Resume(tenant.GetTenantID(c), c.Param("id"), c.Param("sessionId"), in.ActorCharacterID)
	default:
		d, err = h.service.End(tenant.GetTenantID(c), c.Param("id"), c.Param("sessionId"), in.ActorCharacterID)
	}
	if err != nil {
		writeGameSessionError(c, err)
		return
	}
	respondSuccess(c, d)
}
