package handler

import (
	"errors"
	"net/http"
	"strconv"

	"control-panel/internal/application/services"
	"control-panel/internal/domain/tenant"
	"github.com/gin-gonic/gin"
)

type GameProjectHandler struct{ service *services.GameProjectService }

func NewGameProjectHandler(service *services.GameProjectService) *GameProjectHandler {
	return &GameProjectHandler{service: service}
}

func gameProjectError(c *gin.Context, err error) {
	status := http.StatusBadRequest
	if errors.Is(err, services.ErrGameProjectNotFound) {
		status = http.StatusNotFound
	} else if errors.Is(err, services.ErrGameProjectConflict) {
		status = http.StatusConflict
	}
	respondError(c, status, err.Error())
}

func (h *GameProjectHandler) Create(c *gin.Context) {
	var input services.CreateGameProjectInput
	if err := c.ShouldBindJSON(&input); err != nil {
		respondError(c, http.StatusBadRequest, "invalid game details")
		return
	}
	result, err := h.service.Create(tenant.GetTenantID(c), actorID(c), input)
	if err != nil {
		gameProjectError(c, err)
		return
	}
	respondCreated(c, result)
}

func (h *GameProjectHandler) List(c *gin.Context) {
	result, err := h.service.List(tenant.GetTenantID(c))
	if err != nil {
		gameProjectError(c, err)
		return
	}
	respondSuccess(c, gin.H{"items": result, "total": len(result)})
}

func (h *GameProjectHandler) Get(c *gin.Context) {
	result, err := h.service.Get(tenant.GetTenantID(c), c.Param("id"))
	if err != nil {
		gameProjectError(c, err)
		return
	}
	respondSuccess(c, result)
}

func (h *GameProjectHandler) Update(c *gin.Context) {
	var input services.UpdateGameProjectInput
	if err := c.ShouldBindJSON(&input); err != nil {
		respondError(c, http.StatusBadRequest, "invalid game content")
		return
	}
	result, err := h.service.Update(tenant.GetTenantID(c), c.Param("id"), actorID(c), input)
	if err != nil {
		gameProjectError(c, err)
		return
	}
	respondSuccess(c, result)
}

func (h *GameProjectHandler) Publish(c *gin.Context) {
	var input services.PublishGameProjectInput
	if err := c.ShouldBindJSON(&input); err != nil {
		respondError(c, http.StatusBadRequest, "version is required")
		return
	}
	check, err := h.service.PublishCheck(tenant.GetTenantID(c), c.Param("id"))
	if err != nil {
		gameProjectError(c, err)
		return
	}
	if !check.CanPublish {
		respondError(c, http.StatusUnprocessableEntity, "发布检查未通过，请先修正必须完成的内容")
		return
	}
	result, err := h.service.Publish(tenant.GetTenantID(c), c.Param("id"), actorID(c), input)
	if err != nil {
		gameProjectError(c, err)
		return
	}
	respondCreated(c, result)
}

func (h *GameProjectHandler) PublishCheck(c *gin.Context) {
	result, err := h.service.PublishCheck(tenant.GetTenantID(c), c.Param("id"))
	if err != nil {
		gameProjectError(c, err)
		return
	}
	respondSuccess(c, result)
}

func (h *GameProjectHandler) GetRelease(c *gin.Context) {
	result, err := h.service.GetRelease(tenant.GetTenantID(c), c.Param("id"), c.Param("releaseId"))
	if err != nil {
		gameProjectError(c, err)
		return
	}
	respondSuccess(c, result)
}
func (h *GameProjectHandler) RestoreRelease(c *gin.Context) {
	var in services.RestoreGameReleaseInput
	if c.ShouldBindJSON(&in) != nil {
		respondError(c, http.StatusBadRequest, "请选择如何使用这个历史版本")
		return
	}
	result, err := h.service.RestoreRelease(tenant.GetTenantID(c), c.Param("id"), c.Param("releaseId"), actorID(c), in)
	if err != nil {
		gameProjectError(c, err)
		return
	}
	respondCreated(c, result)
}

func (h *GameProjectHandler) CreateScene(c *gin.Context) {
	var input services.SaveGameSceneInput
	if err := c.ShouldBindJSON(&input); err != nil {
		respondError(c, http.StatusBadRequest, "请完整填写场景信息")
		return
	}
	result, err := h.service.CreateScene(tenant.GetTenantID(c), c.Param("id"), actorID(c), input)
	if err != nil {
		gameProjectError(c, err)
		return
	}
	respondCreated(c, result)
}

func (h *GameProjectHandler) UpdateScene(c *gin.Context) {
	var input services.SaveGameSceneInput
	if err := c.ShouldBindJSON(&input); err != nil {
		respondError(c, http.StatusBadRequest, "请完整填写场景信息")
		return
	}
	result, err := h.service.UpdateScene(tenant.GetTenantID(c), c.Param("id"), c.Param("sceneId"), actorID(c), input)
	if err != nil {
		gameProjectError(c, err)
		return
	}
	respondSuccess(c, result)
}

func (h *GameProjectHandler) DeleteScene(c *gin.Context) {
	revision, err := strconv.Atoi(c.Query("revision"))
	if err != nil || revision <= 0 {
		respondError(c, http.StatusBadRequest, "缺少当前版本号")
		return
	}
	result, err := h.service.DeleteScene(tenant.GetTenantID(c), c.Param("id"), c.Param("sceneId"), actorID(c), revision)
	if err != nil {
		gameProjectError(c, err)
		return
	}
	respondSuccess(c, result)
}

func (h *GameProjectHandler) PreviewAction(c *gin.Context) {
	var input services.PreviewGameActionInput
	if err := c.ShouldBindJSON(&input); err != nil {
		respondError(c, http.StatusBadRequest, "请先选择要试算的行动")
		return
	}
	result, err := h.service.PreviewAction(tenant.GetTenantID(c), c.Param("id"), input)
	if err != nil {
		gameProjectError(c, err)
		return
	}
	respondSuccess(c, result)
}

func (h *GameProjectHandler) PreviewCharacterContext(c *gin.Context) {
	result, err := h.service.PreviewCharacterContext(tenant.GetTenantID(c), c.Param("id"), c.Param("characterId"))
	if err != nil {
		gameProjectError(c, err)
		return
	}
	respondSuccess(c, result)
}

func (h *GameProjectHandler) PreviewEvent(c *gin.Context) {
	var input services.PreviewGameEventsInput
	if err := c.ShouldBindJSON(&input); err != nil {
		respondError(c, http.StatusBadRequest, "请填写用于试跑的当前状态")
		return
	}
	result, err := h.service.PreviewEvents(tenant.GetTenantID(c), c.Param("id"), c.Param("eventId"), input)
	if err != nil {
		gameProjectError(c, err)
		return
	}
	respondSuccess(c, result)
}

func (h *GameProjectHandler) PreviewEventSchedule(c *gin.Context) {
	var input services.PreviewGameEventsInput
	if err := c.ShouldBindJSON(&input); err != nil {
		respondError(c, http.StatusBadRequest, "请填写用于试跑的当前状态")
		return
	}
	result, err := h.service.PreviewEvents(tenant.GetTenantID(c), c.Param("id"), "", input)
	if err != nil {
		gameProjectError(c, err)
		return
	}
	respondSuccess(c, result)
}
