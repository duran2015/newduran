package handler

import (
	"net/http"

	"control-panel/internal/application/services"
	"control-panel/internal/domain/tenant"
	"github.com/gin-gonic/gin"
)

type GamePlayerSessionHandler struct {
	service *services.GamePlayerSessionService
}

func NewGamePlayerSessionHandler(service *services.GamePlayerSessionService) *GamePlayerSessionHandler {
	return &GamePlayerSessionHandler{service: service}
}

func (h *GamePlayerSessionHandler) Create(c *gin.Context) {
	var in services.CreateGamePlayerSessionInput
	if err := c.ShouldBindJSON(&in); err != nil {
		respondError(c, http.StatusBadRequest, "请选择试玩内容")
		return
	}
	in.CreatedBy = actorID(c)
	d, err := h.service.Create(tenant.GetTenantID(c), c.Param("id"), in)
	if err != nil {
		writeGameSessionError(c, err)
		return
	}
	respondCreated(c, d)
}

func (h *GamePlayerSessionHandler) CreateRelease(c *gin.Context) {
	projectID, err := h.service.ProjectForRelease(tenant.GetTenantID(c), c.Param("releaseId"))
	if err != nil {
		writeGameSessionError(c, err)
		return
	}
	in := services.CreateGamePlayerSessionInput{Source: "release", ReleaseID: c.Param("releaseId"), CreatedBy: actorID(c)}
	d, err := h.service.Create(tenant.GetTenantID(c), projectID, in)
	if err != nil {
		writeGameSessionError(c, err)
		return
	}
	respondCreated(c, d)
}

func (h *GamePlayerSessionHandler) ReleaseDetail(c *gin.Context) {
	d, err := h.service.ReleaseDetail(tenant.GetTenantID(c), c.Param("releaseId"))
	if err != nil {
		writeGameSessionError(c, err)
		return
	}
	respondSuccess(c, d)
}

func (h *GamePlayerSessionHandler) publicProject(c *gin.Context) (string, bool) {
	id, err := h.service.ProjectForRelease(tenant.GetTenantID(c), c.Param("releaseId"))
	if err != nil {
		writeGameSessionError(c, err)
		return "", false
	}
	if sessionID := c.Param("sessionId"); sessionID != "" {
		if err = h.service.ValidateReleaseSession(tenant.GetTenantID(c), c.Param("releaseId"), sessionID); err != nil {
			writeGameSessionError(c, err)
			return "", false
		}
	}
	return id, true
}
func (h *GamePlayerSessionHandler) GetRelease(c *gin.Context) {
	id, ok := h.publicProject(c)
	if !ok {
		return
	}
	d, err := h.service.Get(tenant.GetTenantID(c), id, c.Param("sessionId"))
	if err != nil {
		writeGameSessionError(c, err)
		return
	}
	respondSuccess(c, d)
}
func (h *GamePlayerSessionHandler) InterpretRelease(c *gin.Context) {
	id, ok := h.publicProject(c)
	if !ok {
		return
	}
	var in services.InterpretGamePlayerInput
	if c.ShouldBindJSON(&in) != nil {
		respondError(c, 400, "请说出你想做什么")
		return
	}
	d, err := h.service.Interpret(tenant.GetTenantID(c), id, c.Param("sessionId"), in)
	if err != nil {
		writeGameSessionError(c, err)
		return
	}
	respondSuccess(c, d)
}
func (h *GamePlayerSessionHandler) ConfirmRelease(c *gin.Context) {
	id, ok := h.publicProject(c)
	if !ok {
		return
	}
	var in services.ConfirmGamePlayerInput
	if c.ShouldBindJSON(&in) != nil {
		respondError(c, 400, "请选择要确认的行动")
		return
	}
	d, err := h.service.Confirm(tenant.GetTenantID(c), id, c.Param("sessionId"), in)
	if err != nil {
		writeGameSessionError(c, err)
		return
	}
	respondSuccess(c, d)
}
func (h *GamePlayerSessionHandler) RejectRelease(c *gin.Context) {
	id, ok := h.publicProject(c)
	if !ok {
		return
	}
	var in services.RejectGamePlayerInput
	if c.ShouldBindJSON(&in) != nil {
		respondError(c, 400, "请选择要放弃的行动")
		return
	}
	d, err := h.service.Reject(tenant.GetTenantID(c), id, c.Param("sessionId"), in)
	if err != nil {
		writeGameSessionError(c, err)
		return
	}
	respondSuccess(c, d)
}

func (h *GamePlayerSessionHandler) AcceptObjectiveRelease(c *gin.Context) {
	id, ok := h.publicProject(c)
	if !ok {
		return
	}
	h.decideObjective(c, id, true)
}

func (h *GamePlayerSessionHandler) RejectObjectiveRelease(c *gin.Context) {
	id, ok := h.publicProject(c)
	if !ok {
		return
	}
	h.decideObjective(c, id, false)
}

func (h *GamePlayerSessionHandler) Get(c *gin.Context) {
	d, err := h.service.Get(tenant.GetTenantID(c), c.Param("id"), c.Param("sessionId"))
	if err != nil {
		writeGameSessionError(c, err)
		return
	}
	respondSuccess(c, d)
}

func (h *GamePlayerSessionHandler) Interpret(c *gin.Context) {
	var in services.InterpretGamePlayerInput
	if err := c.ShouldBindJSON(&in); err != nil {
		respondError(c, http.StatusBadRequest, "请说出你想做什么")
		return
	}
	d, err := h.service.Interpret(tenant.GetTenantID(c), c.Param("id"), c.Param("sessionId"), in)
	if err != nil {
		writeGameSessionError(c, err)
		return
	}
	respondSuccess(c, d)
}

func (h *GamePlayerSessionHandler) Confirm(c *gin.Context) {
	var in services.ConfirmGamePlayerInput
	if err := c.ShouldBindJSON(&in); err != nil {
		respondError(c, http.StatusBadRequest, "请选择要确认的行动")
		return
	}
	d, err := h.service.Confirm(tenant.GetTenantID(c), c.Param("id"), c.Param("sessionId"), in)
	if err != nil {
		writeGameSessionError(c, err)
		return
	}
	respondSuccess(c, d)
}

func (h *GamePlayerSessionHandler) Reject(c *gin.Context) {
	var in services.RejectGamePlayerInput
	if err := c.ShouldBindJSON(&in); err != nil {
		respondError(c, http.StatusBadRequest, "请选择要放弃的行动")
		return
	}
	d, err := h.service.Reject(tenant.GetTenantID(c), c.Param("id"), c.Param("sessionId"), in)
	if err != nil {
		writeGameSessionError(c, err)
		return
	}
	respondSuccess(c, d)
}

func (h *GamePlayerSessionHandler) AcceptObjective(c *gin.Context) {
	h.decideObjective(c, c.Param("id"), true)
}

func (h *GamePlayerSessionHandler) RejectObjective(c *gin.Context) {
	h.decideObjective(c, c.Param("id"), false)
}

func (h *GamePlayerSessionHandler) decideObjective(c *gin.Context, projectID string, accept bool) {
	var in services.DecideGameObjectiveInput
	if err := c.ShouldBindJSON(&in); err != nil {
		respondError(c, http.StatusBadRequest, "请选择要处理的任务")
		return
	}
	var (
		d   *services.GamePlayerSessionDetail
		err error
	)
	if accept {
		d, err = h.service.AcceptObjective(tenant.GetTenantID(c), projectID, c.Param("sessionId"), in)
	} else {
		d, err = h.service.RejectObjective(tenant.GetTenantID(c), projectID, c.Param("sessionId"), in)
	}
	if err != nil {
		writeGameSessionError(c, err)
		return
	}
	respondSuccess(c, d)
}

func (h *GamePlayerSessionHandler) Reset(c *gin.Context) {
	d, err := h.service.Reset(tenant.GetTenantID(c), c.Param("id"), c.Param("sessionId"), actorID(c))
	if err != nil {
		writeGameSessionError(c, err)
		return
	}
	respondCreated(c, d)
}
func (h *GamePlayerSessionHandler) ResetRelease(c *gin.Context) {
	id, ok := h.publicProject(c)
	if !ok {
		return
	}
	d, err := h.service.Reset(tenant.GetTenantID(c), id, c.Param("sessionId"), actorID(c))
	if err != nil {
		writeGameSessionError(c, err)
		return
	}
	respondCreated(c, d)
}
