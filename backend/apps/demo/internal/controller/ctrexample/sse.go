// Package ctrexample 是示例控制器：演示 SSE、响应格式化与外部 client 调用。
package ctrexample

import (
	"fmt"
	"io"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/morehao/golib/glog"
)

// SSECtr 是 SSE 流式推送示例控制器接口。
type SSECtr interface {
	Time(ctx *gin.Context)
	TimeRaw(ctx *gin.Context)
	Process(ctx *gin.Context)
	Chat(ctx *gin.Context)
	Raw(ctx *gin.Context)
}

type sseCtr struct{}

var _ SSECtr = (*sseCtr)(nil)

// NewSSECtr 构造 SSECtr。
func NewSSECtr() SSECtr {
	return &sseCtr{}
}

// Time 实时时间流示例
func (ctr *sseCtr) Time(ctx *gin.Context) {
	// 设置 SSE 响应头
	ctx.Header("Content-Type", "text/event-stream")
	ctx.Header("Cache-Control", "no-cache")
	ctx.Header("Connection", "keep-alive")

	clientGone := ctx.Stream(func(w io.Writer) bool {
		// 必须节流：ctx.Stream 的回调里若没有等待，单连接会以百万级/秒的速度写满管道
		// （实测未节流时 3 秒吐出 ~158 万行，能直接吃满一个核），这里对齐 TimeRaw 的 1s 心跳。
		_, err := fmt.Fprintf(w, "data: %s\n\n", time.Now().Format(time.DateTime))
		time.Sleep(time.Second)
		return err == nil
	})
	if clientGone {
		glog.Infof(ctx, "[sseCtr.Time] Client disconnected during streaming")
	} else {
		glog.Infof(ctx, "[sseCtr.Time] Stream completed normally")
	}
}

// TimeRaw Write写入实时时间流示例
func (ctr *sseCtr) TimeRaw(ctx *gin.Context) {
	// 设置 SSE 响应头
	ctx.Header("Content-Type", "text/event-stream")
	ctx.Header("Cache-Control", "no-cache")
	ctx.Header("Connection", "keep-alive")

	clientGone := ctx.Stream(func(w io.Writer) bool {
		currentTime := time.Now().Format("2006-01-02 15:04:05")
		sseData := fmt.Sprintf("id: %d\nevent: time\ndata: %s\n\n",
			time.Now().Unix(), currentTime)

		// 直接写入到 w
		_, err := w.Write([]byte(sseData))
		if err != nil {
			return false
		}

		time.Sleep(1 * time.Second)
		return true
	})
	if clientGone {
		glog.Infof(ctx, "[sseCtr.TimeRaw] Client disconnected during streaming")
	} else {
		glog.Infof(ctx, "[sseCtr.TimeRaw] Stream completed normally")
	}
}

// Process 模拟数据处理进度示例
func (ctr *sseCtr) Process(ctx *gin.Context) {
	// 设置 SSE 响应头
	ctx.Header("Content-Type", "text/event-stream")
	ctx.Header("Cache-Control", "no-cache")
	ctx.Header("Connection", "keep-alive")

	progress := 0

	clientGone := ctx.Stream(func(_ io.Writer) bool {
		if progress <= 100 {
			// 发送进度更新
			ctx.SSEvent("progress", fmt.Sprintf(`{"progress": %d, "message": "Processing... %d%%"}`, progress, progress))
			progress += 10
			time.Sleep(500 * time.Millisecond)
			return true
		}
		// 完成时发送完成事件
		ctx.SSEvent("complete", `{"progress": 100, "message": "Task completed!"}`)
		return false
	})
	if clientGone {
		glog.Infof(ctx, "[sseCtr.Process] Client disconnected during streaming")
	} else {
		glog.Infof(ctx, "[sseCtr.Process] Stream completed normally")
	}
}

// Chat 聊天消息流示例
func (ctr *sseCtr) Chat(ctx *gin.Context) {
	// 设置 SSE 响应头
	ctx.Header("Content-Type", "text/event-stream")
	ctx.Header("Cache-Control", "no-cache")
	ctx.Header("Connection", "keep-alive")

	messages := []string{
		"Hello! How can I help you today?",
		"I'm an AI assistant powered by SSE streaming.",
		"This demonstrates real-time message delivery.",
		"Each message arrives with a small delay.",
		"This simulates a natural conversation flow.",
		"SSE is perfect for chat applications!",
		"Thanks for trying this demo. Goodbye! 👋",
	}

	messageIndex := 0

	clientGone := ctx.Stream(func(_ io.Writer) bool {
		if messageIndex < len(messages) {
			dataMsg := fmt.Sprintf(`{"id": %d, "text": "%s", "timestamp": "%s"}`,
				messageIndex+1,
				messages[messageIndex],
				time.Now().Format("15:04:05"))

			ctx.SSEvent("message", dataMsg)
			messageIndex++
			time.Sleep(2 * time.Second)
			return true
		}
		// 所有消息发送完毕
		ctx.SSEvent("end", `{"message": "Conversation ended"}`)
		return false
	})
	if clientGone {
		glog.Infof(ctx, "[sseCtr.Chat] Client disconnected during streaming")
	} else {
		glog.Infof(ctx, "[sseCtr.Chat] Stream completed normally")
	}
}

// Raw 自定义格式的 SSE 示例
func (ctr *sseCtr) Raw(ctx *gin.Context) {
	// 设置 SSE 响应头
	ctx.Header("Content-Type", "text/event-stream")
	ctx.Header("Cache-Control", "no-cache")
	ctx.Header("Connection", "keep-alive")

	counter := 0

	clientGone := ctx.Stream(func(_ io.Writer) bool {
		if counter < 10 {
			dataMsg := fmt.Sprintf(`{"count": %d, "message": "This is event #%d"}`, counter, counter)

			ctx.SSEvent("message", dataMsg)

			counter++
			time.Sleep(1 * time.Second)
			return true
		}
		ctx.SSEvent("message", `{"message": "Stream ended"}`)
		return false
	})
	if clientGone {
		glog.Infof(ctx, "[sseCtr.Raw] Client disconnected during streaming")
	} else {
		glog.Infof(ctx, "[sseCtr.Raw] Stream completed normally")
	}
}
