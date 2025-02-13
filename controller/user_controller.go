package controller

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/viswals_backend_test/core/models"
	database "github.com/viswals_backend_test/core/postgres"
	"go.uber.org/zap"
)

func (c *Controller) GetAllUsers(ctx *gin.Context) {
	ctxWithTimeout, cancel := context.WithTimeout(ctx.Request.Context(), defaultTimeout)
	defer cancel()

	users, err := c.UserService.GetAllUsers(ctxWithTimeout)
	if err != nil {
		c.sendResponse(ctx, http.StatusInternalServerError, "failed to get all users", nil)
		return
	}

	c.sendResponse(ctx, http.StatusOK, "all users", users)
}

func (c *Controller) GetUser(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		c.sendResponse(ctx, http.StatusBadRequest, "user id is not provided in req or empty id, please check url. it should be /users/:id", nil)
		return
	}

	ctxWithTimeout, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	user, err := c.UserService.GetUser(ctxWithTimeout, id)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			c.sendResponse(ctx, http.StatusRequestTimeout, "deadline exceed please try again after some time.", nil)
			return
		}
		if errors.Is(err, database.ErrNoData) {
			c.sendResponse(ctx, http.StatusNotFound, "requested data not found", nil)
			return
		}
		c.logger.Error("failed to get user", zap.Error(err), zap.String("id", id))
		c.sendResponse(ctx, http.StatusInternalServerError, "internal server error", nil)
		return
	}

	c.sendResponse(ctx, http.StatusOK, "success", user)
}

func (c *Controller) CreateUser(ctx *gin.Context) {
	var user models.UserDetails

	if err := ctx.ShouldBindJSON(&user); err != nil {
		c.sendResponse(ctx, http.StatusBadRequest, "failed to parse request body", nil)
		return
	}

	ctxWithTimeout, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	err := c.UserService.CreateUser(ctxWithTimeout, &user)
	if err != nil {
		if errors.Is(err, database.ErrDuplicate) {
			c.sendResponse(ctx, http.StatusConflict, "user already exist in database", nil)
			return
		}
		if errors.Is(err, context.DeadlineExceeded) {
			c.sendResponse(ctx, http.StatusRequestTimeout, "request time out please try again later", nil)
			return
		}
		c.sendResponse(ctx, http.StatusInternalServerError, "internal server error can't fetch data from database", nil)
		return
	}

	c.sendResponse(ctx, http.StatusCreated, "data created successfully", nil)
}

func (c *Controller) DeleteUser(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		c.sendResponse(ctx, http.StatusBadRequest, "request does not contains any id for user", nil)
		return
	}

	ctxWithTimeout, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	err := c.UserService.DeleteUser(ctxWithTimeout, id)
	if err != nil {
		if errors.Is(err, database.ErrNoData) {
			c.sendResponse(ctx, http.StatusNoContent, "requested data not found or already deleted", nil)
			return
		}
		if errors.Is(err, context.DeadlineExceeded) {
			c.sendResponse(ctx, http.StatusRequestTimeout, "request time out please try again later", nil)
			return
		}
		c.sendResponse(ctx, http.StatusInternalServerError, "internal server error can't fetch data from database", nil)
		return
	}

	c.sendResponse(ctx, http.StatusNoContent, "success", nil)
}

func (c *Controller) GetAllUsersSSE(ctx *gin.Context) {
	var limit int64 = 10
	var offset int64 = 1
	var err error

	if ctx.Query("limit") != "" {
		limit, err = strconv.ParseInt(ctx.Query("limit"), 10, 64)
		if err != nil {
			c.sendResponse(ctx, http.StatusBadRequest, "failed to parse limit parameter", nil)
			return
		}
	}

	ctx.Writer.Header().Set("Content-Type", "text/event-stream")
	ctx.Writer.Header().Set("Cache-Control", "no-cache")
	ctx.Writer.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := ctx.Writer.(gin.ResponseWriter)
	if !ok {
		c.sendResponse(ctx, http.StatusInternalServerError, "internal server error", nil)
		return
	}

	var isLastData bool
	for {
		data, err := c.UserService.GetAllUsersSSE(context.Background(), limit, offset*limit)
		if err != nil {
			if errors.Is(err, database.ErrNoData) {
				isLastData = true
			}
		}
		_, err = fmt.Fprintf(ctx.Writer, "data: %s\n\n", string(data))
		if err != nil {
			c.logger.Error("failed to write response", zap.Error(err))
			c.sendResponse(ctx, http.StatusInternalServerError, "fail to send data to client", nil)
			return
		}
		flusher.Flush()
		time.Sleep(2 * time.Second)

		if isLastData {
			break
		}

		offset++
	}

	_, err = fmt.Fprint(ctx.Writer, "data: END\n\n")
	if err != nil {
		c.logger.Error("failed to write response", zap.Error(err))
		c.sendResponse(ctx, http.StatusInternalServerError, "fail to send data to client", nil)
		return
	}
	flusher.Flush()
}
