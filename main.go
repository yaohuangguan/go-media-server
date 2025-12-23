package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	// 简单的健康检查
	r.GET("/", func(c *gin.Context) {
		c.String(200, "Service is running")
	})

	// ---------------------------------------------------------
	// 接口 1: 获取视频信息
	// ---------------------------------------------------------
	r.GET("/info", func(c *gin.Context) {
		videoURL := c.Query("url")
		if videoURL == "" {
			c.JSON(400, gin.H{"error": "请提供 url 参数"})
			return
		}

		cmd := exec.Command("yt-dlp", "--dump-json", "--no-warnings", "--no-check-certificate", videoURL)
		output, err := cmd.Output()
		
		if err != nil {
			fmt.Println("解析错误:", err)
			c.JSON(500, gin.H{"error": "无法解析链接", "details": err.Error()})
			return
		}

		var meta map[string]interface{}
		json.Unmarshal(output, &meta)

		c.JSON(200, gin.H{
			"title":     meta["title"],
			"thumbnail": meta["thumbnail"],
			"duration":  meta["duration_string"],
			"platform":  meta["extractor_key"],
		})
	})

	// ---------------------------------------------------------
	// 接口 2: 万能下载 (支持 视频MP4 和 音频MP3)
	// 用法: 
	//   视频: /download?url=...
	//   音频: /download?url=...&type=audio
	// ---------------------------------------------------------
	r.GET("/download", func(c *gin.Context) {
		videoURL := c.Query("url")
		downloadType := c.Query("type") // 获取 type 参数，如果是 "audio" 则转 mp3

		if videoURL == "" {
			c.String(400, "Missing URL")
			return
		}

		// 基础参数：不输出警告，不校验证书，输出到标准输出(-)
		args := []string{"--no-warnings", "--no-check-certificate", "-o", "-"}

		// 根据 type 参数决定下载逻辑
		contentType := ""
		filenameExt := ""

		if downloadType == "audio" {
			// === 音频模式 (MP3) ===
			// -x: 提取音频
			// --audio-format mp3: 强制转码为 mp3
			// --audio-quality 0: 0 表示最佳音质 (VBR)
			args = append(args, "-x", "--audio-format", "mp3", "--audio-quality", "0")
			
			contentType = "audio/mpeg"
			filenameExt = "mp3"
		} else {
			// === 默认视频模式 (MP4) ===
			// -f best: 下载最佳画质
			// 注意：如果视频本身不是 mp4 封装，yt-dlp 可能会流式传输 mkv 或 webm
			// 如果强行要 mp4，可以加 "--recode-video", "mp4" (但会增加 CPU 负担和延迟)
			args = append(args, "-f", "best")
			
			contentType = "video/mp4"
			filenameExt = "mp4"
		}

		// 最后追加 URL
		args = append(args, videoURL)

		// 创建命令 (关联 Context，支持客户端断开自动停止)
		cmd := exec.CommandContext(c.Request.Context(), "yt-dlp", args...)

		// 获取管道
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			c.String(500, "System Error: Pipe creation failed")
			return
		}

		// 启动进程
		if err := cmd.Start(); err != nil {
			c.String(500, "Download Error: "+err.Error())
			return
		}

		// 设置响应头
		filename := fmt.Sprintf("download_%d.%s", time.Now().Unix(), filenameExt)
		c.Writer.Header().Set("Content-Type", contentType)
		c.Writer.Header().Set("Content-Disposition", "attachment; filename="+filename)

		// 开始流式传输
		io.Copy(c.Writer, stdout)
	})

	// Cloud Run 端口适配
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	fmt.Printf("🚀 服务启动在端口: %s\n", port)
	r.Run(":" + port)
}