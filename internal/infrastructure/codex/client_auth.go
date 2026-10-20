package codex

import "encoding/json"

// UseClientAuth 仅在后端回调内提供可导入的 ChatGPT 授权，回调结束后清零副本。
func UseClientAuth(home, expectedID string, consume func([]byte) error) error {
	if expectedID == "" || consume == nil {
		return ErrProtocol
	}
	return useStoredAuth(home, func(data []byte) error {
		body, err := ClientAuthFile(data, expectedID)
		if err != nil {
			return err
		}
		defer clear(body)
		return consume(body)
	})
}

// ClientAuthFile 校验完整授权并移除混合 API 登录字段；返回值只供后端写入受控目标。
func ClientAuthFile(data []byte, expectedID string) ([]byte, error) {
	if len(data) == 0 || len(data) > 128<<10 || expectedID == "" || len(expectedID) > 256 {
		return nil, ErrProtocol
	}
	var auth struct {
		Mode   string `json:"auth_mode"`
		Tokens struct {
			ID      string `json:"id_token"`
			Access  string `json:"access_token"`
			Refresh string `json:"refresh_token"`
			Account string `json:"account_id"`
		} `json:"tokens"`
	}
	if json.Unmarshal(data, &auth) != nil || auth.Mode != "" && auth.Mode != "chatgpt" || auth.Tokens.Account != expectedID {
		return nil, ErrProtocol
	}
	for _, token := range []string{auth.Tokens.ID, auth.Tokens.Access, auth.Tokens.Refresh, auth.Tokens.Account} {
		if len(token) == 0 || len(token) > 32768 {
			return nil, ErrProtocol
		}
		for _, character := range token {
			if character < 33 || character > 126 {
				return nil, ErrProtocol
			}
		}
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(data, &fields) != nil {
		return nil, ErrProtocol
	}
	defer func() {
		for _, value := range fields {
			clear(value)
		}
	}()
	clear(fields["OPENAI_API_KEY"])
	delete(fields, "OPENAI_API_KEY")
	fields["auth_mode"] = json.RawMessage(`"chatgpt"`)
	body, err := json.MarshalIndent(fields, "", "  ")
	if err != nil {
		return nil, ErrProtocol
	}
	return append(body, '\n'), nil
}
