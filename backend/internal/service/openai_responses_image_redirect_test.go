package service

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestBuildOpenAIImagesRequestFromResponses_Generation(t *testing.T) {
	body := []byte(`{
		"model":"gpt-5.4",
		"input":"draw a calm data center",
		"tools":[{"type":"image_generation","model":"gpt-image-2","size":"1024x1024","quality":"high","n":2}],
		"tool_choice":{"type":"image_generation"}
	}`)

	out, endpoint, err := BuildOpenAIImagesRequestFromResponses(body)
	require.NoError(t, err)
	require.Equal(t, "/v1/images/generations", endpoint)
	require.Equal(t, "draw a calm data center", gjson.GetBytes(out, "prompt").String())
	require.Equal(t, "gpt-image-2", gjson.GetBytes(out, "model").String())
	require.Equal(t, "1024x1024", gjson.GetBytes(out, "size").String())
	require.Equal(t, "high", gjson.GetBytes(out, "quality").String())
	require.Equal(t, int64(2), gjson.GetBytes(out, "n").Int())
	require.Equal(t, "b64_json", gjson.GetBytes(out, "response_format").String())
	require.False(t, gjson.GetBytes(out, "stream").Exists())
}

func TestBuildOpenAIImagesRequestFromResponses_EditWithInputImages(t *testing.T) {
	body := []byte(`{
		"model":"gpt-5.4",
		"input":[
			{"role":"system","content":"ignore"},
			{"role":"user","content":[
				{"type":"input_text","text":"turn this into line art"},
				{"type":"input_image","image_url":"data:image/png;base64,abc"},
				{"type":"input_image","image_url":"https://example.test/in.png"}
			]}
		],
		"tools":[{"type":"image_generation","model":"gpt-image-2","input_image_mask":{"image_url":"data:image/png;base64,mask"}}]
	}`)

	out, endpoint, err := BuildOpenAIImagesRequestFromResponses(body)
	require.NoError(t, err)
	require.Equal(t, "/v1/images/edits", endpoint)
	require.Equal(t, "turn this into line art", gjson.GetBytes(out, "prompt").String())
	require.Len(t, gjson.GetBytes(out, "images").Array(), 2)
	require.Equal(t, "data:image/png;base64,abc", gjson.GetBytes(out, "images.0.image_url").String())
	require.Equal(t, "https://example.test/in.png", gjson.GetBytes(out, "images.1.image_url").String())
	require.Equal(t, "data:image/png;base64,mask", gjson.GetBytes(out, "mask.image_url").String())
}

func TestResponsesImageGenerationRedirectConfig_PlatformMap(t *testing.T) {
	ch := &Channel{FeaturesConfig: map[string]any{
		"responses_image_generation_redirect": map[string]any{
			"openai": map[string]any{
				"enabled":  true,
				"group_id": json.Number("42"),
			},
		},
	}}

	cfg := ch.ResponsesImageGenerationRedirectOverride(PlatformOpenAI)
	require.NotNil(t, cfg)
	require.True(t, cfg.Enabled)
	require.EqualValues(t, 42, cfg.GroupID)
	require.Nil(t, ch.ResponsesImageGenerationRedirectOverride(PlatformAnthropic))
}

func TestBuildOpenAIResponsesImageRedirectSSE(t *testing.T) {
	body := buildOpenAIResponsesImageRedirectBody(
		"resp_test",
		"gpt-image-2",
		1710000000,
		[]openAIResponsesImageResult{{Result: "abc", Size: "1024x1024", Model: "gpt-image-2"}},
		openAIResponsesImageResult{Model: "gpt-image-2", Size: "1024x1024"},
		OpenAIUsage{},
	)

	sse := string(BuildOpenAIResponsesImageRedirectSSE(body))
	require.Contains(t, sse, "event: response.created")
	require.Contains(t, sse, `"type":"response.output_item.done"`)
	require.Contains(t, sse, "event: response.completed")
	require.Contains(t, sse, `"id":"resp_test"`)
	require.Contains(t, sse, `"type":"image_generation_call"`)
	require.NotContains(t, sse, "[DONE]")
}
