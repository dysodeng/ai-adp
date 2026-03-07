package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	chatdto "github.com/dysodeng/ai-adp/internal/application/chat/dto"
	chatservice "github.com/dysodeng/ai-adp/internal/application/chat/service"
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
func (h *ChatHandler) Chat(c *gin.Context) {
	var req request.ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Fail(c.Request.Context(), err.Error(), response.CodeFail))
		return
	}

	appID := c.Param("app_id")
	if appID == "" {
		c.JSON(http.StatusBadRequest, response.Fail(c.Request.Context(), "app_id is required", response.CodeFail))
		return
	}

	// 默认 streaming 模式
	if req.ResponseMode == "" {
		req.ResponseMode = "streaming"
	}
	responseMode := chatdto.ResponseMode(req.ResponseMode)
	if !responseMode.IsValid() {
		c.JSON(http.StatusBadRequest, response.Fail(c.Request.Context(), "invalid response_mode", response.CodeFail))
		return
	}

	// 创建协议适配器
	var adapter protocol.Adapter
	if responseMode == chatdto.ResponseModeBlocking {
		adapter = protocol.NewBlockingAdapter(c.Writer)
	} else {
		sseAdapter, err := protocol.NewSSEAdapter(c.Writer)
		if err != nil {
			c.JSON(http.StatusInternalServerError, response.Fail(c.Request.Context(), "streaming not supported", response.CodeInternalServerError))
			return
		}
		adapter = sseAdapter
	}
	defer func() { _ = adapter.Close() }()

	// 调用应用服务
	agentExecutor, err := h.chatService.Chat(
		c.Request.Context(),
		appID,
		chatdto.ChatCommand{
			ConversationID: req.ConversationID,
			Query:          req.Query,
			Input:          req.Input,
			ResponseMode:   responseMode,
		},
	)
	if err != nil {
		if responseMode == chatdto.ResponseModeBlocking {
			c.JSON(http.StatusOK, response.Fail(c.Request.Context(), err.Error(), response.CodeFail))
		} else {
			_ = adapter.SendError(err)
		}
		return
	}

	// 使用适配器订阅并转发事件
	_ = adapter.HandleExecution(c.Request.Context(), agentExecutor)
}

// RegisterRoutes 注册 Chat 路由
func (h *ChatHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/apps/:app_id/chat", h.Chat)
}
