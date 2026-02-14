package client

type ContentBlock map[string]any

func ContentBlockText(text string) ContentBlock {
	return ContentBlock{"type": "text", "text": text}
}

func ContentBlockJSON(value any) ContentBlock {
	return ContentBlock{"type": "json", "json": value}
}

func ContentBlockImage(base64Data, mimeType string) ContentBlock {
	return ContentBlock{
		"type":      "image",
		"data":      base64Data,
		"mime_type": mimeType,
	}
}

func ContentBlockCustom(kind string, fields map[string]any) ContentBlock {
	block := ContentBlock{"type": kind}
	for key, value := range fields {
		block[key] = value
	}
	return block
}
