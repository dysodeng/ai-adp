package handler

import (
	"net/http"

	"github.com/dysodeng/ai-adp/internal/infrastructure/pkg/telemetry/trace"
	"github.com/gin-gonic/gin"

	chatdto "github.com/dysodeng/ai-adp/internal/application/chat/dto"
	chatservice "github.com/dysodeng/ai-adp/internal/application/chat/service"
	"github.com/dysodeng/ai-adp/internal/infrastructure/pkg/logger"
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

// Chat Agent 对话接口，支持 SSE 流式和阻塞式响应
func (h *ChatHandler) Chat(ctx *gin.Context) {
	spanCtx, span := trace.Tracer().Start(trace.Gin(ctx), "api.Handler.ChatHandler")
	defer span.End()

	var req request.ChatRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusOK, response.Fail(spanCtx, err.Error(), response.CodeFail))
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
		sseAdapter, err := protocol.NewSSEAdapter(ctx.Writer, false)
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
			ConversationID: req.ConversationID,
			Query:          req.Query,
			Input:          req.Input,
			ResponseMode:   responseMode,
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

	logger.Info(spanCtx, "[handler] chatService.Chat returned, calling HandleExecution")

	// 使用适配器订阅并转发事件
	if err = adapter.HandleExecution(spanCtx, agentExecutor); err != nil {
		logger.Error(spanCtx, "[handler] HandleExecution error", logger.ErrorField(err))
	}
	logger.Info(spanCtx, "[handler] HandleExecution done")
}
