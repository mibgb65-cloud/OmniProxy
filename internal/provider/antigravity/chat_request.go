package antigravity

import (
	"encoding/json"
	"strings"

	"omniproxy/internal/core/proxy"
)

type object = map[string]any

type chatTool struct {
	Type     string          `json:"type"`
	Function json.RawMessage `json:"function"`
}

type chatMessage struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
	ToolID  string          `json:"tool_call_id"`
	Calls   []struct {
		ID       string `json:"id"`
		Type     string `json:"type"`
		Function struct {
			Name      string `json:"name"`
			Arguments string `json:"arguments"`
		} `json:"function"`
		Extra json.RawMessage `json:"extra_content"`
	} `json:"tool_calls"`
}

// payload 是构造好的上游包封，以及发送前的原文 system（未打散）：
// 拒绝判定与「同 prompt 不再撞一次」的指纹都用原文，不能用打散后的文本。
type payload struct {
	body   []byte
	system string
}

// chatPayload 把 Chat Completions 请求整形为 Code Assist 的包封。
// promptAsWritten 为真时按原样发送 system（不打散被拒行）。
func chatPayload(request proxy.Request, cred Credentials, promptAsWritten bool) (payload, error) {
	if request.Protocol != "chat" || !supportedModel(request.Model) {
		return payload{}, proxy.Failure("unsupported_protocol", 400)
	}
	var input struct {
		Messages   []chatMessage   `json:"messages"`
		Tools      []chatTool      `json:"tools"`
		ToolChoice json.RawMessage `json:"tool_choice"`
	}
	if json.Unmarshal(request.Body, &input) != nil || len(input.Messages) == 0 || len(input.Messages) > 1000 || len(input.Tools) > 128 {
		return payload{}, proxy.Failure("invalid_request", 400)
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(request.Body, &fields) != nil {
		return payload{}, proxy.Failure("invalid_request", 400)
	}
	for _, key := range []string{"response_format", "audio", "modalities", "prediction", "reasoning_effort", "extra_body"} {
		if value, ok := fields[key]; ok && string(value) != "null" {
			return payload{}, proxy.Failure("unsupported_content", 400)
		}
	}
	messages, err := repairedMessages(input.Messages)
	if err != nil {
		return payload{}, err
	}
	family := familyOf(request.Model)
	native := object{}
	contents := []object{}
	systems := []object{}
	systemTexts := []string{}
	// 调用参数跟着 tool_call 一起留到最后一条 tool 结果：客户端把读到的图片
	// 换成占位文本时，回填要靠这里的路径找回文件。
	calls := map[string]pendingCall{}
	firstUser := ""
	for _, message := range messages {
		var text string
		parts := []object{}
		switch message.Role {
		case "user":
			// 用户消息可以带内联图片：文本与 inlineData 都要保留。
			text, parts, err = userParts(message.Content)
		case "tool":
			// 工具结果可以带回 agent 读到的图片：文本进 functionResponse，
			// 图片作为同一回合里紧随其后的 inlineData part。
			text, parts, err = toolParts(message.Content)
		default:
			text, err = textContent(message.Content)
			if text != "" {
				parts = append(parts, object{"text": text})
			}
		}
		if err != nil {
			return payload{}, err
		}
		switch message.Role {
		case "system", "developer":
			if len(message.Calls) > 0 {
				return payload{}, proxy.Failure("unsupported_content", 400)
			}
			if text != "" {
				systems = append(systems, object{"text": text})
				systemTexts = append(systemTexts, text)
			}
			continue
		case "assistant":
			callParts, err := assistantParts(message)
			if err != nil {
				return payload{}, err
			}
			for _, call := range message.Calls {
				calls[call.ID] = pendingCall{name: call.Function.Name, arguments: call.Function.Arguments}
			}
			parts = append(parts, callParts...)
		case "tool":
			call := calls[message.ToolID]
			if call.name == "" {
				return payload{}, proxy.Failure("invalid_request", 400)
			}
			parts = append([]object{{"functionResponse": object{"id": message.ToolID, "name": call.name, "response": object{"result": text}}}}, parts...)
			if len(parts) == 1 && family == familyGemini {
				// 内容里没有图片，而结果只是客户端的二进制占位文本：按同一轮调用的
				// 路径回读文件，把客户端丢掉的像素补回同一回合。只对目录里声明了
				// 图片输入的 Gemini 族生效，Claude／GPT-OSS 通道保持纯文本。
				if image := backfillInlineImage(text, call.arguments); image != nil {
					parts = append(parts, image)
				}
			}
			delete(calls, message.ToolID)
		case "user":
			if len(message.Calls) > 0 {
				return payload{}, proxy.Failure("unsupported_content", 400)
			}
			if firstUser == "" {
				firstUser = text
			}
		default:
			return payload{}, proxy.Failure("unsupported_content", 400)
		}
		if len(parts) == 0 {
			return payload{}, proxy.Failure("unsupported_content", 400)
		}
		role := "user"
		if message.Role == "assistant" {
			role = "model"
		}
		contents = append(contents, object{"role": role, "parts": parts})
	}
	if len(contents) == 0 {
		return payload{}, proxy.Failure("invalid_request", 400)
	}
	native["contents"] = mergeTurns(contents)
	system := strings.Join(systemTexts, "\n")
	if len(systems) > 0 {
		if !promptAsWritten {
			// 命中行按词插入零宽空格：模型照读，Google 的行匹配看不到。
			for index := range systems {
				if text, ok := systems[index]["text"].(string); ok {
					systems[index]["text"] = cloaked(text)
				}
			}
		}
		native["systemInstruction"] = object{"parts": systems}
	}
	if len(input.Tools) > 0 {
		declarations, err := toolDeclarations(input.Tools, request.Model)
		if err != nil {
			return payload{}, err
		}
		native["tools"] = []object{{"functionDeclarations": declarations}}
	}
	choice, err := toolChoice(input.ToolChoice, input.Tools, family)
	if err != nil {
		return payload{}, err
	}
	if choice != nil {
		native["toolConfig"] = choice
	}
	config, maxTokens, err := generationOptions(fields, family)
	if err != nil {
		return payload{}, err
	}
	if thinking := thinkingOptions(request.Effort, request.Model, family, maxTokens); thinking != nil {
		config["thinkingConfig"] = thinking
		// Claude 的回答必须在思考之外留出空间：未指定输出上限时按预算补足。
		if budget, ok := thinking["thinkingBudget"].(int); ok && family == familyClaude && config["maxOutputTokens"] == nil {
			config["maxOutputTokens"] = budget + 32000
		}
	}
	if len(config) > 0 {
		native["generationConfig"] = config
	}
	native["sessionId"] = sessionID(firstUser)
	envelope := object{
		"project": cred.Project, "model": request.Model, "userAgent": "antigravity",
		"requestType": "agent", "requestId": "agent-" + randomHex(16), "request": native,
	}
	body, err := json.Marshal(envelope)
	if err != nil {
		return payload{}, proxy.Failure("invalid_request", 400)
	}
	return payload{body: body, system: system}, nil
}
