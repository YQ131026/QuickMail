package server

import (
	"errors"
	"net/http"

	"QuickMail/internal/config"
	"QuickMail/internal/email"

	"github.com/gin-gonic/gin"
)

const apiKeyHeader = "X-API-Key"

// Server wraps HTTP handlers for the QuickMail service.
type Server struct {
	engine *gin.Engine
	store  *config.Store
	sender *email.Sender
	apiKey string
}

func New(store *config.Store, sender *email.Sender, apiKey string) *Server {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	srv := &Server{
		engine: r,
		store:  store,
		sender: sender,
		apiKey: apiKey,
	}

	r.Use(srv.authMiddleware)

	r.GET("/health", srv.handleHealth)
	r.GET("/health/providers/:name", srv.handleProviderHealth)

	r.GET("/providers", srv.handleListProviders)
	r.POST("/providers", srv.handleUpsertProvider)
	r.DELETE("/providers/:name", srv.handleDeleteProvider)

	r.POST("/send", srv.handleSendEmail)

	return srv
}

func (s *Server) Engine() *gin.Engine {
	return s.engine
}

func (s *Server) Run(addr string) error {
	return s.engine.Run(addr)
}

func (s *Server) authMiddleware(c *gin.Context) {
	if s.apiKey == "" {
		c.Next()
		return
	}

	if c.GetHeader(apiKeyHeader) != s.apiKey {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	c.Next()
}

func (s *Server) handleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (s *Server) handleProviderHealth(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "provider name required"})
		return
	}

	if err := s.sender.CheckProvider(c.Request.Context(), name); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unhealthy", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "healthy"})
}

func (s *Server) handleListProviders(c *gin.Context) {
	providers, err := s.store.ListProviders()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, providers)
}

func (s *Server) handleUpsertProvider(c *gin.Context) {
	var input config.ProviderInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := s.store.UpsertProvider(input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	provider := config.ProviderResponse{
		Name:     input.Name,
		Host:     input.Host,
		Port:     input.Port,
		Username: input.Username,
		From:     input.From,
		UseTLS:   input.UseTLS,
	}

	c.JSON(http.StatusCreated, provider)
}

func (s *Server) handleDeleteProvider(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "provider name required"})
		return
	}

	if err := s.store.DeleteProvider(name); err != nil {
		if err == config.ErrProviderNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "provider not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

func (s *Server) handleSendEmail(c *gin.Context) {
	var req email.SendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := s.sender.Send(c.Request.Context(), req); err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, email.ErrAllProvidersFailed) {
			status = http.StatusServiceUnavailable
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusAccepted)
}
