package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	chatdto "github.com/dysodeng/ai-adp/internal/application/chat/dto"
	chatservice "github.com/dysodeng/ai-adp/internal/application/chat/service"
	"github.com/dysodeng/ai-adp/internal/infrastructure/logger"
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
	var req request.ChatRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusOK, response.Fail(ctx, err.Error(), response.CodeFail))
		return
	}

	apiKey := ctx.GetString("api_key")

	// 默认 streaming 模式
	if req.ResponseMode == "" {
		req.ResponseMode = "streaming"
	}
	responseMode := chatdto.ResponseMode(req.ResponseMode)
	if !responseMode.IsValid() {
		ctx.JSON(http.StatusOK, response.Fail(ctx, "invalid response_mode", response.CodeFail))
		return
	}

	// 创建协议适配器
	var adapter protocol.Adapter
	if responseMode == chatdto.ResponseModeBlocking {
		adapter = protocol.NewBlockingAdapter(ctx.Writer)
	} else {
		sseAdapter, err := protocol.NewSSEAdapter(ctx.Writer)
		if err != nil {
			ctx.JSON(http.StatusOK, response.Fail(ctx, "streaming not supported", response.CodeInternalServerError))
			return
		}
		adapter = sseAdapter
	}
	defer func() { _ = adapter.Close() }()

	logger.Info(ctx, "[handler] calling chatService.Chat",
		logger.AddField("api_key", apiKey),
		logger.AddField("query", req.Query),
		logger.AddField("response_mode", req.ResponseMode),
	)

	// 调用应用服务
	agentExecutor, err := h.chatService.Chat(
		ctx,
		apiKey,
		chatdto.ChatCommand{
			ConversationID: req.ConversationID,
			Query:          req.Query,
			Input:          req.Input,
			ResponseMode:   responseMode,
		},
	)
	if err != nil {
		logger.Error(ctx, "[handler] chatService.Chat failed", logger.ErrorField(err))
		if responseMode == chatdto.ResponseModeBlocking {
			ctx.JSON(http.StatusOK, response.Fail(ctx, err.Error(), response.CodeFail))
		} else {
			_ = adapter.SendError(err)
		}
		return
	}

	logger.Info(ctx, "[handler] chatService.Chat returned, calling HandleExecution")

	// 使用适配器订阅并转发事件
	if err = adapter.HandleExecution(ctx, agentExecutor); err != nil {
		logger.Error(ctx, "[handler] HandleExecution error", logger.ErrorField(err))
	}
	logger.Info(ctx, "[handler] HandleExecution done")
}
