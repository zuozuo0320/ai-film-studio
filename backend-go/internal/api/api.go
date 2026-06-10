// Package api REST + WebSocket 接口层（gin）。
package api

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"github.com/zuozuo0320/ai-film-studio/backend-go/internal/models"
	"github.com/zuozuo0320/ai-film-studio/backend-go/internal/queue"
	"github.com/zuozuo0320/ai-film-studio/backend-go/internal/scheduler"
	"github.com/zuozuo0320/ai-film-studio/backend-go/internal/storage"
	"github.com/zuozuo0320/ai-film-studio/backend-go/internal/ws"
)

type Server struct {
	store storage.Store
	q     queue.Queue
	hub   *ws.Hub
	sched *scheduler.Scheduler
	data  string
}

func NewServer(store storage.Store, q queue.Queue, hub *ws.Hub, sched *scheduler.Scheduler, dataDir string) *Server {
	return &Server{store: store, q: q, hub: hub, sched: sched, data: dataDir}
}

func (s *Server) Router() *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	r.GET("/healthz", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })
	r.Static("/media", s.data)

	v1 := r.Group("/api/v1")
	{
		v1.POST("/projects", s.createProject)
		v1.GET("/projects", s.listProjects)
		v1.GET("/projects/:id", s.getProject)
		v1.PATCH("/projects/:id", s.updateProject)
		v1.DELETE("/projects/:id", s.deleteProject)

		v1.POST("/projects/:id/characters", s.addCharacter)
		v1.POST("/projects/:id/analyze", s.analyze)

		// 把分镜置为 queued，等待下一个批次窗口；或调试用立即组装
		v1.POST("/projects/:id/shots/:shotID/enqueue-image", s.enqueueImage)
		v1.POST("/projects/:id/shots/:shotID/approve", s.approveShot) // 质量关卡一：确认图片
		v1.POST("/projects/:id/shots/:shotID/video", s.generateVideo)
		v1.POST("/debug/assemble-batch", s.assembleNow)

		v1.GET("/projects/:id/batches", s.listBatches)
	}
	r.GET("/ws", s.websocket)
	return r
}

// ---------- projects ----------

type projectCreateReq struct {
	Title  string `json:"title" binding:"required"`
	Script string `json:"script"`
}

func (s *Server) createProject(c *gin.Context) {
	var req projectCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	p := models.NewProject(req.Title, req.Script)
	if err := s.store.CreateProject(p); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, p)
}

func (s *Server) listProjects(c *gin.Context) {
	ps, err := s.store.ListProjects()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, ps)
}

func (s *Server) getProject(c *gin.Context) {
	p, err := s.store.GetProject(c.Param("id"))
	if err != nil {
		s.notFoundOr500(c, err)
		return
	}
	c.JSON(http.StatusOK, p)
}

type projectUpdateReq struct {
	Title  *string `json:"title"`
	Script *string `json:"script"`
}

func (s *Server) updateProject(c *gin.Context) {
	p, err := s.store.GetProject(c.Param("id"))
	if err != nil {
		s.notFoundOr500(c, err)
		return
	}
	var req projectUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Title != nil {
		p.Title = *req.Title
	}
	if req.Script != nil {
		p.Script = *req.Script
	}
	if err := s.store.UpdateProject(p); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, p)
}

func (s *Server) deleteProject(c *gin.Context) {
	if err := s.store.DeleteProject(c.Param("id")); err != nil {
		s.notFoundOr500(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// ---------- characters ----------

type characterReq struct {
	Name        string   `json:"name" binding:"required"`
	Description string   `json:"description"`
	RefImages   []string `json:"ref_images"`
}

func (s *Server) addCharacter(c *gin.Context) {
	p, err := s.store.GetProject(c.Param("id"))
	if err != nil {
		s.notFoundOr500(c, err)
		return
	}
	var req characterReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ch := models.NewCharacter(req.Name, req.Description)
	if req.RefImages != nil {
		ch.RefImages = req.RefImages
	}
	p.Characters = append(p.Characters, ch)
	if err := s.store.UpdateProject(p); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, ch)
}

// ---------- generation ----------

func (s *Server) analyze(c *gin.Context) {
	id := c.Param("id")
	if _, err := s.store.GetProject(id); err != nil {
		s.notFoundOr500(c, err)
		return
	}
	taskID := fmt.Sprintf("analyze_%s_%d", id, time.Now().UnixNano())
	if err := s.q.Publish(queue.TopicScriptAnalyze, queue.Message{TaskID: taskID, ProjectID: id}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"task_id": taskID})
}

// enqueueImage 把分镜标记为 queued，下一个批次窗口（默认 10 分钟）统一出图。
func (s *Server) enqueueImage(c *gin.Context) {
	p, err := s.store.GetProject(c.Param("id"))
	if err != nil {
		s.notFoundOr500(c, err)
		return
	}
	shot := findShot(p, c.Param("shotID"))
	if shot == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "shot not found"})
		return
	}
	shot.Status = models.ShotQueued
	shot.Error = ""
	if err := s.store.UpdateProject(p); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusAccepted, shot)
}

// approveShot 质量关卡一：用户确认图片，分镜方可生成视频。
func (s *Server) approveShot(c *gin.Context) {
	p, err := s.store.GetProject(c.Param("id"))
	if err != nil {
		s.notFoundOr500(c, err)
		return
	}
	shot := findShot(p, c.Param("shotID"))
	if shot == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "shot not found"})
		return
	}
	if shot.Status != models.ShotImageReview {
		c.JSON(http.StatusConflict, gin.H{"error": "shot 不在待确认状态"})
		return
	}
	// 维持 image_review 状态，确认动作即解锁视频生成；
	// 后续可扩展为独立 approved 字段。
	c.JSON(http.StatusOK, shot)
}

func (s *Server) generateVideo(c *gin.Context) {
	p, err := s.store.GetProject(c.Param("id"))
	if err != nil {
		s.notFoundOr500(c, err)
		return
	}
	shot := findShot(p, c.Param("shotID"))
	if shot == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "shot not found"})
		return
	}
	if shot.ImagePath == "" {
		c.JSON(http.StatusConflict, gin.H{"error": "请先出图并确认后再生成视频"})
		return
	}
	taskID := fmt.Sprintf("video_%s_%d", shot.ID, time.Now().UnixNano())
	if err := s.q.Publish(queue.TopicVideoGenerate, queue.Message{TaskID: taskID, ProjectID: p.ID, ShotID: shot.ID}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"task_id": taskID})
}

func (s *Server) assembleNow(c *gin.Context) {
	s.sched.AssembleOnce()
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Server) listBatches(c *gin.Context) {
	bs, err := s.store.ListBatches(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, bs)
}

// ---------- websocket ----------

var upgrader = websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}

func (s *Server) websocket(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()
	events, cancel := s.hub.Subscribe(c.Query("project_id"))
	defer cancel()

	done := make(chan struct{})
	go func() { // 读循环：仅用于感知断开
		defer close(done)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()
	for {
		select {
		case e := <-events:
			if err := conn.WriteJSON(e); err != nil {
				return
			}
		case <-done:
			return
		}
	}
}

// ---------- helpers ----------

func findShot(p *models.Project, shotID string) *models.Shot {
	for i := range p.Shots {
		if p.Shots[i].ID == shotID {
			return &p.Shots[i]
		}
	}
	return nil
}

func (s *Server) notFoundOr500(c *gin.Context, err error) {
	if errors.Is(err, storage.ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
}
