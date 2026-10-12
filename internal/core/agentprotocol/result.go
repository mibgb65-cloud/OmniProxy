package agentprotocol

import (
	"bufio"
	"encoding/json"
	"io"
	"strings"
)

// ReadChat 仅接收完成的 JSON 快照，转换失败不返回半个工具调用。
func ReadChat(reader io.Reader, kinds map[string]string) (Result, error) {
	body, err := io.ReadAll(io.LimitReader(reader, (8<<20)+1))
	defer clear(body)
	if err != nil || len(body) > 8<<20 {
		return Result{}, ErrUnsupported
	}
	var raw struct {
		ID, Model string
		Choices   []struct {
			Finish  string `json:"finish_reason"`
			Message struct {
				Content string
				Calls   []struct {
					ID       string
					Function struct{ Name, Arguments string }
				} `json:"tool_calls"`
			}
		}
		Usage struct {
			Input  int `json:"prompt_tokens"`
			Output int `json:"completion_tokens"`
		}
	}
	if json.Unmarshal(body, &raw) != nil || len(raw.Choices) != 1 {
		return Result{}, ErrUnsupported
	}
	choice := raw.Choices[0]
	r := Result{ID: raw.ID, Model: raw.Model, Text: choice.Message.Content, Finish: choice.Finish, InputTokens: raw.Usage.Input, OutputTokens: raw.Usage.Output}
	for _, c := range choice.Message.Calls {
		if c.ID == "" || c.Function.Name == "" || !json.Valid([]byte(c.Function.Arguments)) {
			return Result{}, ErrUnsupported
		}
		kind, name := kinds[c.Function.Name], c.Function.Name
		if parts := strings.Split(kind, "|"); len(parts) == 3 {
			kind = parts[0]
			name = parts[1] + "." + parts[2]
		}
		r.Calls = append(r.Calls, Call{c.ID, name, c.Function.Arguments, kind})
	}
	return r, nil
}

// ReadResponses 必须见到 completed；流被中断时不能把已有片段伪装成成功。
func ReadResponses(reader io.Reader) (Result, error) {
	scanner := bufio.NewScanner(io.LimitReader(reader, (32<<20)+1))
	scanner.Buffer(make([]byte, 4096), 2<<20)
	total := 0
	for scanner.Scan() {
		line := scanner.Text()
		total += len(line) + 1
		if total > 32<<20 {
			return Result{}, ErrUnsupported
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		var event Object
		if json.Unmarshal([]byte(strings.TrimSpace(strings.TrimPrefix(line, "data:"))), &event) != nil {
			return Result{}, ErrUnsupported
		}
		if text(event["type"]) == "response.failed" || text(event["type"]) == "error" {
			return Result{}, ErrUnsupported
		}
		if text(event["type"]) != "response.completed" {
			continue
		}
		return decodeResponses(object(event["response"]))
	}
	return Result{}, ErrUnsupported
}
func decodeResponses(raw Object) (Result, error) {
	if text(raw["status"]) != "completed" {
		return Result{}, ErrUnsupported
	}
	r := Result{ID: text(raw["id"]), Model: text(raw["model"]), Finish: "stop"}
	for _, v := range array(raw["output"]) {
		item := object(v)
		switch text(item["type"]) {
		case "message":
			s, e := content(item["content"])
			if e != nil {
				return r, e
			}
			r.Text += s
		case "function_call":
			args := text(item["arguments"])
			if !json.Valid([]byte(args)) {
				return r, ErrUnsupported
			}
			r.Calls = append(r.Calls, Call{text(item["call_id"]), text(item["name"]), args, "function"})
		case "reasoning":
		default:
			return r, ErrUnsupported
		}
	}
	if len(r.Calls) > 0 {
		r.Finish = "tool_calls"
	}
	usage := object(raw["usage"])
	r.InputTokens = intNumber(usage["input_tokens"])
	r.OutputTokens = intNumber(usage["output_tokens"])
	return r, nil
}
func intNumber(v any) int {
	n, _ := v.(float64)
	if n < 0 || n > 1e9 {
		return 0
	}
	return int(n)
}
