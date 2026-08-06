package app

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

var modelEndpoint = RelayV1URL + "/models"
var messagesEndpoint = RelayV1URL + "/messages"

func validateModels(token string, tools []string) ([]string, error) {
	var required []string
	if containsTool(tools, "claude") || containsTool(tools, desktopTool) {
		required = append(required, ClaudeModel, ClaudeOpusModel, ClaudeHaikuModel)
	}
	if containsTool(tools, "codex") {
		required = append(required, CodexModel)
	}
	return validateRequiredModels(token, required)
}

func validateRequiredModels(token string, required []string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, modelEndpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("创建模型验证请求: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "aiengine-setup")
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("连接 AiEngine API 失败: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("AiEngine API 拒绝了该密钥（HTTP %d）", response.StatusCode)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("AiEngine 模型接口返回 HTTP %d", response.StatusCode)
	}
	var payload struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	decoder := json.NewDecoder(io.LimitReader(response.Body, 4*1024*1024))
	if err := decoder.Decode(&payload); err != nil {
		return nil, fmt.Errorf("AiEngine 模型接口响应无效: %w", err)
	}
	available := make(map[string]bool)
	for _, model := range payload.Data {
		available[model.ID] = true
	}
	var missing []string
	for _, model := range required {
		if !available[model] {
			missing = append(missing, model)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		return nil, fmt.Errorf("该密钥缺少所需模型: %v", missing)
	}
	return required, nil
}

func validateDesktopMessages(token string) error {
	payload, err := json.Marshal(map[string]any{
		"model":      ClaudeModel,
		"max_tokens": 1,
		"stream":     true,
		"messages": []map[string]string{{
			"role": "user", "content": "Reply with OK.",
		}},
	})
	if err != nil {
		return fmt.Errorf("生成 Claude Desktop 验证请求: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, messagesEndpoint, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("创建 Claude Desktop 验证请求: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Anthropic-Version", "2023-06-01")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("User-Agent", "aiengine-setup")
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("连接 AiEngine Messages API 失败: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden {
		return fmt.Errorf("AiEngine Messages API 拒绝了该密钥（HTTP %d）", response.StatusCode)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("AiEngine Messages API 返回 HTTP %d", response.StatusCode)
	}
	scanner := bufio.NewScanner(io.LimitReader(response.Body, 1024*1024))
	for scanner.Scan() {
		if strings.HasPrefix(strings.TrimSpace(scanner.Text()), "data:") {
			return nil
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("读取 AiEngine 流式响应: %w", err)
	}
	return fmt.Errorf("AiEngine Messages API 未返回 Claude Desktop 所需的流式 SSE 响应")
}
