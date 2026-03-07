package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/dysodeng/ai-adp/internal/domain/agent/executor"
	"github.com/dysodeng/ai-adp/internal/infrastructure/logger"
	"github.com/dysodeng/ai-adp/internal/interfaces/http/dto/response"
)

// CancelHandler 任务取消 Handler
type CancelHandler struct {
	taskRegistry      executor.TaskRegistry
	cancelBroadcaster executor.CancelBroadcaster
}

// NewCancelHandler 创建 Cancel Handler
func NewCancelHandler(
	taskRegistry executor.TaskRegistry,
	cancelBroadcaster executor.CancelBroadcaster,
) *CancelHandler {
	return &CancelHandler{
		taskRegistry:      taskRegistry,
		cancelBroadcaster: cancelBroadcaster,
	}
}

// Cancel 取消指定任务
func (h *CancelHandler) Cancel(ctx *gin.Context) {
	taskID := ctx.Param("task_id")
	if taskID == "" {
		ctx.JSON(http.StatusOK, response.Fail(ctx, "task_id is required", response.CodeFail))
		return
	}

	// 本地取消
	h.taskRegistry.Cancel(taskID)

	// 广播到其他实例
	if err := h.cancelBroadcaster.Broadcast(ctx, taskID); err != nil {
		logger.Warn(ctx, "[CancelHandler] broadcast cancel failed", logger.ErrorField(err))
	}

	ctx.JSON(http.StatusOK, response.Success(ctx, map[string]string{"result": "success"}))
}
