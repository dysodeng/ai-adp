package handler

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/bytedance/sonic"
	"github.com/gin-gonic/gin"

	chatdto "github.com/dysodeng/ai-adp/internal/application/chat/dto"
	chatservice "github.com/dysodeng/ai-adp/internal/application/chat/service"
	"github.com/dysodeng/ai-adp/internal/domain/agent/model"
	"github.com/dysodeng/ai-adp/internal/infrastructure/pkg/logger"
	"github.com/dysodeng/ai-adp/internal/infrastructure/pkg/telemetry/trace"
	"github.com/dysodeng/ai-adp/internal/infrastructure/protocol"
	"github.com/dysodeng/ai-adp/internal/interfaces/http/dto/request"
	"github.com/dysodeng/ai-adp/internal/interfaces/http/dto/response"
)

// ChatHandler Chat 对话 Handler
type ChatHandler struct {
	chatService chatservice.ChatAppService
}

// NewChatHandler 创建 Chat Handler
func NewChatHandler(chatService chatservice.ChatAppService) *ChatHandler {
	return &ChatHandler{chatService: chatService}
}

// Chat Agent 对话接口，支持 SSE 流式和阻塞式响应，也支持 SSE 重连
func (h *ChatHandler) Chat(ctx *gin.Context) {
	spanCtx, span := trace.Tracer().Start(trace.Gin(ctx), "api.Handler.ChatHandler")
	defer span.End()

	// 读取原始 body，支持多次解析
	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		ctx.JSON(http.StatusOK, response.Fail(spanCtx, "read request body failed", response.CodeFail))
		return
	}

	// 先尝试解析为重连请求
	var reconnReq request.ReconnectRequest
	if err := sonic.Unmarshal(body, &reconnReq); err == nil && reconnReq.TaskID != "" && reconnReq.LastEventID != "" {
		h.handleReconnect(ctx, spanCtx, reconnReq)
		return
	}

	// 解析为正常 Chat 请求
	var req request.ChatRequest
	if err := sonic.Unmarshal(body, &req); err != nil {
		ctx.JSON(http.StatusOK, response.Fail(spanCtx, err.Error(), response.CodeFail))
		return
	}
	if req.Query == "" {
		ctx.JSON(http.StatusOK, response.Fail(spanCtx, "query is required", response.CodeFail))
		return
	}

	apiKey := ctx.GetString("api_key")

	// 默认 streaming 模式
	if req.ResponseMode == "" {
		req.ResponseMode = "streaming"
	}
	responseMode := chatdto.ResponseMode(req.ResponseMode)
	if !responseMode.IsValid() {
		ctx.JSON(http.StatusOK, response.Fail(spanCtx, "invalid response_mode", response.CodeFail))
		return
	}

	// 创建协议适配器
	var adapter protocol.Adapter
	if responseMode == chatdto.ResponseModeBlocking {
		adapter = protocol.NewBlockingAdapter(ctx.Writer)
	} else {
		sseAdapter, err := protocol.NewSSEAdapter(ctx.Writer, req.EnableSSEResume)
		if err != nil {
			ctx.JSON(http.StatusOK, response.Fail(spanCtx, "streaming not supported", response.CodeInternalServerError))
			return
		}
		adapter = sseAdapter
	}
	defer func() { _ = adapter.Close() }()

	logger.Info(spanCtx, "[handler] calling chatService.Chat",
		logger.AddField("api_key", apiKey),
		logger.AddField("query", req.Query),
		logger.AddField("response_mode", req.ResponseMode),
	)

	// 调用应用服务
	agentExecutor, err := h.chatService.Chat(
		spanCtx,
		apiKey,
		chatdto.ChatCommand{
			ConversationID:  req.ConversationID,
			Query:           req.Query,
			Input:           req.Input,
			ResponseMode:    responseMode,
			EnableSSEResume: req.EnableSSEResume,
		},
	)
	if err != nil {
		logger.Error(spanCtx, "[handler] chatService.Chat failed", logger.ErrorField(err))
		if responseMode == chatdto.ResponseModeBlocking {
			ctx.JSON(http.StatusOK, response.Fail(spanCtx, err.Error(), response.CodeFail))
		} else {
			_ = adapter.SendError(err)
		}
		return
	}

	// 使用适配器订阅并转发事件
	if err = adapter.HandleExecution(spanCtx, agentExecutor); err != nil {
		logger.Error(spanCtx, "[handler] HandleExecution error", logger.ErrorField(err))
	}
}

// handleReconnect 处理 SSE 重连请求（POST）
func (h *ChatHandler) handleReconnect(ctx *gin.Context, spanCtx context.Context, req request.ReconnectRequest) {
	sseAdapter, err := protocol.NewSSEAdapter(ctx.Writer, true)
	if err != nil {
		ctx.JSON(http.StatusOK, response.Fail(spanCtx, "streaming not supported", response.CodeInternalServerError))
		return
	}
	defer func() { _ = sseAdapter.Close() }()

	cachedEvents, liveExecutor, err := h.chatService.Reconnect(
		spanCtx,
		chatdto.ReconnectCommand{
			TaskID:      req.TaskID,
			LastEventID: req.LastEventID,
		},
	)
	if err != nil {
		logger.Error(spanCtx, "[handler] reconnect failed", logger.ErrorField(err))
		_ = sseAdapter.SendError(err)
		return
	}

	// 事件流已过期
	if cachedEvents == nil && liveExecutor == nil {
		_ = sseAdapter.SendEvent(&model.Event{
			Type:      model.EventTypeExpired,
			TaskID:    req.TaskID,
			Timestamp: time.Now(),
			Data:      map[string]string{"message": "event stream expired, please retry with conversation_id"},
		})
		return
	}

	if err = sseAdapter.HandleReconnection(ctx.Request.Context(), cachedEvents, liveExecutor); err != nil {
		logger.Error(spanCtx, "[handler] HandleReconnection error", logger.ErrorField(err))
	}
}

// StreamReconnect GET 端点，供浏览器 EventSource 自动重连使用
func (h *ChatHandler) StreamReconnect(ctx *gin.Context) {
	spanCtx, span := trace.Tracer().Start(trace.Gin(ctx), "api.Handler.StreamReconnect")
	defer span.End()

	taskID := ctx.Param("task_id")
	if taskID == "" {
		ctx.JSON(http.StatusBadRequest, response.Fail(spanCtx, "task_id is required", response.CodeFail))
		return
	}

	// 优先读取 Last-Event-ID Header，其次读取查询参数
	lastEventID := ctx.GetHeader("Last-Event-ID")
	if lastEventID == "" {
		lastEventID = ctx.Query("last_event_id")
	}
	if lastEventID == "" {
		ctx.JSON(http.StatusBadRequest, response.Fail(spanCtx, "last_event_id is required", response.CodeFail))
		return
	}

	sseAdapter, err := protocol.NewSSEAdapter(ctx.Writer, true)
	if err != nil {
		ctx.JSON(http.StatusOK, response.Fail(spanCtx, "streaming not supported", response.CodeInternalServerError))
		return
	}
	defer func() { _ = sseAdapter.Close() }()

	cachedEvents, liveExecutor, err := h.chatService.Reconnect(
		spanCtx,
		chatdto.ReconnectCommand{
			TaskID:      taskID,
			LastEventID: lastEventID,
		},
	)
	if err != nil {
		logger.Error(spanCtx, "[handler] StreamReconnect failed", logger.ErrorField(err))
		_ = sseAdapter.SendError(err)
		return
	}

	if cachedEvents == nil && liveExecutor == nil {
		_ = sseAdapter.SendEvent(&model.Event{
			Type:      model.EventTypeExpired,
			TaskID:    taskID,
			Timestamp: time.Now(),
			Data:      map[string]string{"message": "event stream expired, please retry with conversation_id"},
		})
		return
	}

	if err = sseAdapter.HandleReconnection(ctx.Request.Context(), cachedEvents, liveExecutor); err != nil {
		logger.Error(spanCtx, "[handler] HandleReconnection error", logger.ErrorField(err))
	}
}
