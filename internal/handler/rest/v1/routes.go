package v1

func (h *Handler) RegisterRoutes() {
	v1 := h.Engine.Group("/api/v1")

	v1.GET("/langs", h.getSupportedLanguages)
	v1.POST("/run", h.executeCode)
}
