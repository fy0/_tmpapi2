package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
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

func TestResolveResponsesImageRedirectTarget_FromSourceGroup(t *testing.T) {
	targetGroupID := int64(42)
	body := []byte(`{"model":"gpt-5.4","input":"draw","tools":[{"type":"image_generation"}]}`)
	repo := &responsesImageRedirectGroupRepo{
		groups: map[int64]*Group{
			targetGroupID: {
				ID:                   targetGroupID,
				Platform:             PlatformOpenAI,
				Status:               StatusActive,
				AllowImageGeneration: true,
				Hydrated:             true,
			},
		},
	}
	svc := &OpenAIGatewayService{groupRepo: repo}

	target, err := svc.ResolveResponsesImageRedirectTarget(context.Background(), &APIKey{
		Group: &Group{
			ID:                                      1,
			Platform:                                PlatformOpenAI,
			Status:                                  StatusActive,
			AllowImageGeneration:                    false,
			ResponsesImageGenerationRedirectGroupID: &targetGroupID,
			Hydrated:                                true,
		},
	}, body)

	require.NoError(t, err)
	require.NotNil(t, target)
	require.Equal(t, targetGroupID, target.Group.ID)
	require.Equal(t, targetGroupID, target.Config.GroupID)
	require.Equal(t, 1, repo.getByIDLiteCalls)
}

func TestResolveResponsesImageRedirectTarget_NoGroupConfig(t *testing.T) {
	svc := &OpenAIGatewayService{groupRepo: &responsesImageRedirectGroupRepo{}}
	body := []byte(`{"model":"gpt-5.4","input":"draw","tools":[{"type":"image_generation"}]}`)

	target, err := svc.ResolveResponsesImageRedirectTarget(context.Background(), &APIKey{
		Group: &Group{ID: 1, Platform: PlatformOpenAI, Status: StatusActive, Hydrated: true},
	}, body)

	require.NoError(t, err)
	require.Nil(t, target)
}

func TestResolveResponsesImageRedirectTarget_NoExplicitImageGenerationTool(t *testing.T) {
	targetGroupID := int64(42)
	svc := &OpenAIGatewayService{groupRepo: &responsesImageRedirectGroupRepo{}}
	body := []byte(`{"model":"gpt-5.4","input":"draw"}`)

	target, err := svc.ResolveResponsesImageRedirectTarget(context.Background(), &APIKey{
		Group: &Group{
			ID:                                      1,
			Platform:                                PlatformOpenAI,
			Status:                                  StatusActive,
			ResponsesImageGenerationRedirectGroupID: &targetGroupID,
			Hydrated:                                true,
		},
	}, body)

	require.NoError(t, err)
	require.Nil(t, target)
}

func TestResolveResponsesImageRedirectTarget_NonOpenAISourceGroupIgnored(t *testing.T) {
	targetGroupID := int64(42)
	repo := &responsesImageRedirectGroupRepo{}
	svc := &OpenAIGatewayService{groupRepo: repo}
	body := []byte(`{"model":"gpt-5.4","input":"draw","tools":[{"type":"image_generation"}]}`)

	target, err := svc.ResolveResponsesImageRedirectTarget(context.Background(), &APIKey{
		Group: &Group{
			ID:                                      1,
			Platform:                                PlatformAnthropic,
			Status:                                  StatusActive,
			ResponsesImageGenerationRedirectGroupID: &targetGroupID,
			Hydrated:                                true,
		},
	}, body)

	require.NoError(t, err)
	require.Nil(t, target)
	require.Equal(t, 0, repo.getByIDLiteCalls)
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

type responsesImageRedirectGroupRepo struct {
	groups           map[int64]*Group
	getByIDLiteCalls int
}

func (r *responsesImageRedirectGroupRepo) Create(context.Context, *Group) error { return nil }

func (r *responsesImageRedirectGroupRepo) GetByID(ctx context.Context, id int64) (*Group, error) {
	return r.GetByIDLite(ctx, id)
}

func (r *responsesImageRedirectGroupRepo) GetByIDLite(_ context.Context, id int64) (*Group, error) {
	r.getByIDLiteCalls++
	if r.groups != nil {
		if group, ok := r.groups[id]; ok {
			return group, nil
		}
	}
	return nil, ErrGroupNotFound
}

func (r *responsesImageRedirectGroupRepo) Update(context.Context, *Group) error { return nil }
func (r *responsesImageRedirectGroupRepo) Delete(context.Context, int64) error  { return nil }

func (r *responsesImageRedirectGroupRepo) DeleteCascade(context.Context, int64) ([]int64, error) {
	return nil, nil
}

func (r *responsesImageRedirectGroupRepo) List(context.Context, pagination.PaginationParams) ([]Group, *pagination.PaginationResult, error) {
	return nil, nil, nil
}

func (r *responsesImageRedirectGroupRepo) ListWithFilters(context.Context, pagination.PaginationParams, string, string, string, *bool) ([]Group, *pagination.PaginationResult, error) {
	return nil, nil, nil
}

func (r *responsesImageRedirectGroupRepo) ListActive(context.Context) ([]Group, error) {
	return nil, nil
}

func (r *responsesImageRedirectGroupRepo) ListActiveByPlatform(context.Context, string) ([]Group, error) {
	return nil, nil
}

func (r *responsesImageRedirectGroupRepo) ExistsByName(context.Context, string) (bool, error) {
	return false, nil
}

func (r *responsesImageRedirectGroupRepo) GetAccountCount(context.Context, int64) (int64, int64, error) {
	return 0, 0, nil
}

func (r *responsesImageRedirectGroupRepo) DeleteAccountGroupsByGroupID(context.Context, int64) (int64, error) {
	return 0, nil
}

func (r *responsesImageRedirectGroupRepo) GetAccountIDsByGroupIDs(context.Context, []int64) ([]int64, error) {
	return nil, nil
}

func (r *responsesImageRedirectGroupRepo) BindAccountsToGroup(context.Context, int64, []int64) error {
	return nil
}

func (r *responsesImageRedirectGroupRepo) UpdateSortOrders(context.Context, []GroupSortOrderUpdate) error {
	return nil
}
