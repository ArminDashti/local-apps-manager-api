package httpserver

import (
	"errors"
	"net/http"
	"strings"

	"github.com/ArminDashti/local-apps-manager-api/internal/apps"
	"github.com/ArminDashti/local-apps-manager-api/internal/auth"
	"github.com/ArminDashti/local-apps-manager-api/internal/config"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

type Server struct {
	cfg     config.Config
	auth    *auth.Service
	appsSvc *apps.Service
}

func New(cfg config.Config, authSvc *auth.Service, appsSvc *apps.Service) *Server {
	return &Server{cfg: cfg, auth: authSvc, appsSvc: appsSvc}
}

func (s *Server) Router() *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery(), gin.Logger())
	r.Use(cors.New(cors.Config{
		AllowOrigins:     s.cfg.CORSOrigins,
		AllowMethods:     []string{"GET", "POST", "PATCH", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	v1 := r.Group("/api/v1")
	{
		v1.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})
		v1.POST("/auth/login", s.postLogin)
	}

	protected := v1.Group("")
	protected.Use(jwtMiddleware(s.auth))
	{
		protected.GET("/apps", s.getApps)
		protected.PATCH("/apps/:stem", s.patchApp)
	}

	return r
}

type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (s *Server) postLogin(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	resp, err := s.auth.Login(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid username or password"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (s *Server) getApps(c *gin.Context) {
	rows, err := s.appsSvc.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"apps": rows})
}

type patchAppRequest struct {
	Enabled bool `json:"enabled"`
}

func (s *Server) patchApp(c *gin.Context) {
	stem := strings.TrimSpace(c.Param("stem"))
	if stem == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "stem required"})
		return
	}
	var req patchAppRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	if err := s.appsSvc.SetEnabled(c.Request.Context(), stem, req.Enabled); err != nil {
		if strings.Contains(err.Error(), "already in progress") {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	rows, err := s.appsSvc.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"ok": true})
		return
	}
	for _, row := range rows {
		if row.Stem == stem {
			c.JSON(http.StatusOK, row)
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
