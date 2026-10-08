package application

import "encoding/json"

// Preserve the original OpenAI message, including tool calls, reasoning and
// multipart content. Content is a text projection used only for retrieval.
func (m *ChatMessage) UnmarshalJSON(data []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	*m = ChatMessage{fields: fields}
	if raw, ok := fields["role"]; ok {
		if err := json.Unmarshal(raw, &m.Role); err != nil {
			return err
		}
	}
	if raw, ok := fields["content"]; ok {
		if err := json.Unmarshal(raw, &m.Content); err != nil {
			var parts []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			}
			if err := json.Unmarshal(raw, &parts); err != nil {
				return err
			}
			for _, part := range parts {
				if part.Type == "text" {
					m.Content += part.Text + "\n"
				}
			}
		}
	}
	m.originalContent = m.Content
	return nil
}

func (m ChatMessage) MarshalJSON() ([]byte, error) {
	fields := make(map[string]json.RawMessage, len(m.fields)+2)
	for k, v := range m.fields {
		fields[k] = v
	}
	fields["role"], _ = json.Marshal(m.Role)
	if m.fields == nil || m.Content != m.originalContent {
		fields["content"], _ = json.Marshal(m.Content)
	}
	return json.Marshal(fields)
}
