package utils

import (
	"context"
	"encoding/json"
	"net/http"
)

const (
	translateEndpoint = "/translate"
)

// TranslaterServerResponse is response of the translation server
type TranslaterServerResponse struct {
	TranslatedText string `json:"translatedText"`
}

// Translate translates the message from english to german
func Translate(ctx context.Context, translateServerEnabled bool, translateServerURL, targetLanguage, message string) (interface{}, error) {

	var result TranslaterServerResponse

	if !translateServerEnabled {
		result.TranslatedText = message
		return result, nil
	}

	payload := map[string]interface{}{
		"q":      message,
		"source": "en",
		"target": targetLanguage,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	err = PostJSONResource(ctx, &http.Client{}, translateServerURL+translateEndpoint, nil, "", []byte(jsonPayload), &result)
	if err != nil {
		return "", err
	}
	return result, nil
}
