package handler

import (
	"errors"
	"net/http"

	"control-panel/internal/application/services"
	"control-panel/internal/domain/tenant"
	"github.com/gin-gonic/gin"
)

type GameAssetHandler struct{ service *services.GameAssetService }

func NewGameAssetHandler(service *services.GameAssetService) *GameAssetHandler {
	return &GameAssetHandler{service: service}
}

func gameAssetError(c *gin.Context, err error) {
	status := http.StatusBadRequest
	if errors.Is(err, services.ErrGameProjectNotFound) || errors.Is(err, services.ErrGameAssetNotFound) {
		status = http.StatusNotFound
	}
	if errors.Is(err, services.ErrGameAssetInUse) {
		status = http.StatusConflict
	}
	respondError(c, status, err.Error())
}

func (h *GameAssetHandler) List(c *gin.Context) {
	items, err := h.service.List(c.Request.Context(), tenant.GetTenantID(c), c.Param("id"))
	if err != nil {
		gameAssetError(c, err)
		return
	}
	respondSuccess(c, gin.H{"items": items, "total": len(items)})
}

func (h *GameAssetHandler) Upload(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		respondError(c, http.StatusBadRequest, "请选择要上传的文件")
		return
	}
	body, err := file.Open()
	if err != nil {
		respondError(c, http.StatusBadRequest, "无法读取上传文件")
		return
	}
	defer body.Close()
	result, err := h.service.Upload(c.Request.Context(), tenant.GetTenantID(c), c.Param("id"), actorID(c), c.PostForm("name"), c.PostForm("kind"), file.Filename, body, file.Size)
	if err != nil {
		gameAssetError(c, err)
		return
	}
	respondCreated(c, result)
}

func (h *GameAssetHandler) Replace(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		respondError(c, http.StatusBadRequest, "请选择新的文件")
		return
	}
	body, err := file.Open()
	if err != nil {
		respondError(c, http.StatusBadRequest, "无法读取新的文件")
		return
	}
	defer body.Close()
	result, err := h.service.Replace(c.Request.Context(), tenant.GetTenantID(c), c.Param("id"), c.Param("assetId"), actorID(c), c.PostForm("name"), file.Filename, body, file.Size)
	if err != nil {
		gameAssetError(c, err)
		return
	}
	respondSuccess(c, result)
}

func (h *GameAssetHandler) Content(c *gin.Context) {
	row, body, err := h.service.Open(c.Request.Context(), tenant.GetTenantID(c), c.Param("id"), c.Param("assetId"))
	if err != nil {
		gameAssetError(c, err)
		return
	}
	defer body.Close()
	c.Header("Content-Type", row.MimeType)
	c.Header("Content-Disposition", "inline")
	c.Header("Cache-Control", "private, max-age=3600")
	c.DataFromReader(http.StatusOK, row.SizeBytes, row.MimeType, body, nil)
}

func (h *GameAssetHandler) ReleaseContent(c *gin.Context) {
	row, body, err := h.service.OpenRelease(c.Request.Context(), tenant.GetTenantID(c), c.Param("releaseId"), c.Param("assetId"))
	if err != nil {
		gameAssetError(c, err)
		return
	}
	defer body.Close()
	c.Header("Content-Type", row.MimeType)
	c.Header("Content-Disposition", "inline")
	c.Header("Cache-Control", "private, max-age=86400, immutable")
	c.DataFromReader(http.StatusOK, row.SizeBytes, row.MimeType, body, nil)
}

func (h *GameAssetHandler) Delete(c *gin.Context) {
	if err := h.service.Delete(c.Request.Context(), tenant.GetTenantID(c), c.Param("id"), c.Param("assetId")); err != nil {
		gameAssetError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
