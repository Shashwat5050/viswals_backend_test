package controller

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

var (
	defaultTimeout  = 5 * time.Second
	defaultHttpPort = "8080"
)

type Controller struct {
	UserService UserService
	logger      *zap.Logger
	httpMux     *http.ServeMux
	HttpPort    string
}

type Option func(*Controller)

func WithHttpMux(httpMux *http.ServeMux) func(*Controller) {
	return func(c *Controller) {
		c.httpMux = httpMux
	}
}

func WithHttpPort(port string) func(*Controller) {
	return func(c *Controller) {
		c.HttpPort = port
	}
}

func New(userService UserService, logger *zap.Logger, opts ...Option) *Controller {
	ctrl := &Controller{
		UserService: userService,
		logger:      logger,
	}
	for _, opt := range opts {
		opt(ctrl)
	}
	return ctrl
}

func (c *Controller) sendResponse(ctx *gin.Context, statusCode int, message string, data interface{}) {
	ctx.JSON(statusCode, gin.H{
		"status_code": statusCode,
		"message":     message,
		"data":        data,
	})
}

func (c *Controller) Start() error {
	c.registerRoutes()

	server := &http.Server{
		Handler:           c.httpMux,
		Addr:              fmt.Sprintf(":%v", c.HttpPort),
		ReadHeaderTimeout: 3 * time.Second,
	}

	return server.ListenAndServe()

}

func (c *Controller) Ping(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{
		"message": "pong",
	})
}

func (ctrl *Controller) registerRoutes() {
	router := gin.Default()

	router.GET("/users", ctrl.GetAllUsers)
	router.GET("/users/:id", ctrl.GetUser)
	router.POST("/users", ctrl.CreateUser)
	router.DELETE("/users/:id", ctrl.DeleteUser)
	router.GET("/users/sse", ctrl.GetAllUsersSSE)
	router.Static("/static", "./client")

	ctrl.httpMux.Handle("/", router)
}
