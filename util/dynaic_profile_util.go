package util

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"os"
	"slices"

	"github.com/aiwolfdial/aiwolf-nlp-server/model"
)

type ImageURL struct {
	URL string `json:"url"`
}

type ContentItem struct {
	Type     string   `json:"type"`
	Text     string   `json:"text,omitempty"`
	ImageURL ImageURL `json:"image_url,omitempty"`
}

type Message struct {
	Role    string        `json:"role"`
	Content []ContentItem `json:"content"`
}

type JSONSchema struct {
	Name   string          `json:"name"`
	Strict bool            `json:"strict"`
	Schema json.RawMessage `json:"schema"`
}

type ResponseFormat struct {
	Type       string     `json:"type"`
	JSONSchema JSONSchema `json:"json_schema"`
}

type ChatCompletionRequest struct {
	Model          string         `json:"model"`
	Messages       []Message      `json:"messages"`
	ResponseFormat ResponseFormat `json:"response_format"`
	MaxTokens      int            `json:"max_tokens"`
}

type ProfileData struct {
	Name        string `json:"name"`
	Age         int    `json:"age"`
	Gender      string `json:"gender"`
	Personality string `json:"personality"`
}

type Choice struct {
	Message struct {
		Content string `json:"content"`
	} `json:"message"`
}

type ChatCompletionResponse struct {
	Choices []Choice `json:"choices"`
}

func generateProfile(config model.DynamicProfileConfig, avatarURL string) (*model.Profile, error) {
	schemaJSON := `{
		"type": "object",
		"properties": {
			"name": {
				"type": "string"
			},
			"age": {
				"type": "number"
			},
			"gender": {
				"type": "string"
			},
			"personality": {
				"type": "string"
			}
		},
		"required": [
			"name",
			"age",
			"gender",
			"personality"
		],
		"additionalProperties": false
	}`

	request := ChatCompletionRequest{
		Model: config.Model,
		Messages: []Message{
			{
				Role: "user",
				Content: []ContentItem{
					{
						Type: "text",
						Text: config.Prompt,
					},
					{
						Type: "image_url",
						ImageURL: ImageURL{
							URL: avatarURL,
						},
					},
				},
			},
		},
		ResponseFormat: ResponseFormat{
			Type: "json_schema",
			JSONSchema: JSONSchema{
				Name:   "math_response",
				Strict: true,
				Schema: json.RawMessage(schemaJSON),
			},
		},
		MaxTokens: 300,
	}

	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(requestBody))
	slog.Info("ダイナミックプロフィールの生成をリクエストしました", "avatar", avatarURL)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+os.Getenv("OPENAI_API_KEY"))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var chatResponse ChatCompletionResponse
	if err := json.Unmarshal(bodyBytes, &chatResponse); err != nil {
		return nil, err
	}

	if len(chatResponse.Choices) == 0 {
		return nil, errors.New("no choices in response")
	}

	var profileData ProfileData
	if err := json.Unmarshal([]byte(chatResponse.Choices[0].Message.Content), &profileData); err != nil {
		return nil, err
	}

	profile := &model.Profile{
		Name:        profileData.Name,
		AvatarURL:   avatarURL,
		Age:         profileData.Age,
		Gender:      profileData.Gender,
		Personality: profileData.Personality,
	}
	return profile, nil
}

func generateProfileWithIgnoreNames(config model.DynamicProfileConfig, avatarURL string, ignoreNames []string) (*model.Profile, error) {
	for range config.Attempts {
		profile, err := generateProfile(config, avatarURL)
		if err != nil {
			return nil, err
		}
		if !slices.Contains(ignoreNames, profile.Name) {
			slog.Info("ダイナミックプロフィールを生成しました", "avatar", profile.AvatarURL, "name", profile.Name)
			return profile, nil
		}
	}
	return nil, errors.New("ユニークな名前を生成できませんでした")
}

func GenerateProfiles(config model.DynamicProfileConfig, size int) ([]model.Profile, error) {
	var profiles []model.Profile
	names := make([]string, 0, size)

	avatarURLs := make([]string, len(config.Avatars))
	copy(avatarURLs, config.Avatars)
	rand.Shuffle(len(avatarURLs), func(i, j int) {
		avatarURLs[i], avatarURLs[j] = avatarURLs[j], avatarURLs[i]
	})

	for i := range size {
		profile, err := generateProfileWithIgnoreNames(config, avatarURLs[i], names)
		if err != nil {
			return nil, err
		}
		profiles = append(profiles, *profile)
		names = append(names, profile.Name)
	}
	return profiles, nil
}
