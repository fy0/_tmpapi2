package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

type ResponsesImageGenerationRedirectConfig struct {
	Enabled bool
	GroupID int64
}

type ResponsesImageRedirectTarget struct {
	Config ResponsesImageGenerationRedirectConfig
	Group  *Group
}

type OpenAIResponsesImageRedirectResult struct {
	Result          *OpenAIForwardResult
	ResponseBody    []byte
	ResponseID      string
	CreatedAt       int64
	ImageResults    []openAIResponsesImageResult
	FirstMeta       openAIResponsesImageResult
	ImageRequest    *OpenAIImagesRequest
	UpstreamAccount *Account
	UpstreamGroupID int64
}

func (s *OpenAIGatewayService) ResolveResponsesImageRedirectTarget(ctx context.Context, apiKey *APIKey, body []byte) (*ResponsesImageRedirectTarget, error) {
	if !openAIResponsesRequestHasExplicitImageGenerationTool(body) {
		return nil, nil
	}
	sourceGroup := apiKeyGroup(apiKey)
	if sourceGroup == nil || sourceGroup.ResponsesImageGenerationRedirectGroupID == nil {
		return nil, nil
	}
	if sourceGroup.Platform != PlatformOpenAI {
		return nil, nil
	}
	targetGroupID := *sourceGroup.ResponsesImageGenerationRedirectGroupID
	if targetGroupID <= 0 {
		return nil, nil
	}
	return s.resolveResponsesImageRedirectTargetGroup(ctx, ResponsesImageGenerationRedirectConfig{
		Enabled: true,
		GroupID: targetGroupID,
	})
}

func (s *OpenAIGatewayService) resolveResponsesImageRedirectTargetGroup(ctx context.Context, cfg ResponsesImageGenerationRedirectConfig) (*ResponsesImageRedirectTarget, error) {
	if !cfg.Enabled || cfg.GroupID <= 0 {
		return nil, nil
	}
	if s == nil || s.groupRepo == nil {
		return nil, fmt.Errorf("responses image redirect group repository is not configured")
	}
	group, err := s.groupRepo.GetByIDLite(ctx, cfg.GroupID)
	if err != nil {
		return nil, fmt.Errorf("resolve responses image redirect group: %w", err)
	}
	if group == nil || group.ID <= 0 {
		return nil, fmt.Errorf("responses image redirect group not found")
	}
	if !group.IsActive() {
		return nil, fmt.Errorf("responses image redirect group is not active")
	}
	if group.Platform != PlatformOpenAI {
		return nil, fmt.Errorf("responses image redirect group must be an OpenAI group")
	}
	if !GroupAllowsImageGeneration(group) {
		return nil, fmt.Errorf("responses image redirect group does not allow image generation")
	}
	return &ResponsesImageRedirectTarget{Config: cfg, Group: group}, nil
}

func openAIResponsesRequestHasExplicitImageGenerationTool(body []byte) bool {
	if len(body) == 0 || !gjson.ValidBytes(body) {
		return false
	}
	return openAIJSONToolsContainImageGeneration(gjson.GetBytes(body, "tools")) ||
		openAIJSONToolChoiceSelectsImageGeneration(gjson.GetBytes(body, "tool_choice"))
}

func BuildOpenAIImagesRequestFromResponses(body []byte) ([]byte, string, error) {
	if len(body) == 0 || !gjson.ValidBytes(body) {
		return nil, "", fmt.Errorf("failed to parse request body")
	}
	tool, ok := firstOpenAIResponsesImageGenerationTool(body)
	if !ok {
		return nil, "", fmt.Errorf("image_generation tool is required")
	}
	prompt, images := extractOpenAIResponsesImagePromptAndImages(body)
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return nil, "", fmt.Errorf("image prompt is required")
	}
	endpoint := openAIImagesGenerationsEndpoint
	if len(images) > 0 {
		endpoint = openAIImagesEditsEndpoint
	}
	out := []byte(`{"prompt":"","response_format":"b64_json"}`)
	out, _ = sjson.SetBytes(out, "prompt", prompt)
	model := strings.TrimSpace(tool.Get("model").String())
	if model == "" {
		model = "gpt-image-2"
	}
	out, _ = sjson.SetBytes(out, "model", model)
	copyStringField := func(dstPath, srcPath string) {
		if value := strings.TrimSpace(tool.Get(srcPath).String()); value != "" {
			out, _ = sjson.SetBytes(out, dstPath, value)
		}
	}
	for _, field := range []string{"size", "quality", "background", "output_format", "moderation", "style"} {
		copyStringField(field, field)
	}
	if n := tool.Get("n"); n.Exists() && n.Type == gjson.Number {
		out, _ = sjson.SetBytes(out, "n", n.Int())
	}
	if outputCompression := tool.Get("output_compression"); outputCompression.Exists() && outputCompression.Type == gjson.Number {
		out, _ = sjson.SetBytes(out, "output_compression", outputCompression.Int())
	}
	if partialImages := tool.Get("partial_images"); partialImages.Exists() && partialImages.Type == gjson.Number {
		out, _ = sjson.SetBytes(out, "partial_images", partialImages.Int())
	}
	if maskURL := strings.TrimSpace(tool.Get("input_image_mask.image_url").String()); maskURL != "" {
		out, _ = sjson.SetBytes(out, "mask.image_url", maskURL)
	}
	if len(images) > 0 {
		out, _ = sjson.SetRawBytes(out, "images", []byte(`[]`))
		for _, imageURL := range images {
			item := []byte(`{"image_url":""}`)
			item, _ = sjson.SetBytes(item, "image_url", imageURL)
			out, _ = sjson.SetRawBytes(out, "images.-1", item)
		}
	}
	return out, endpoint, nil
}

func firstOpenAIResponsesImageGenerationTool(body []byte) (gjson.Result, bool) {
	tools := gjson.GetBytes(body, "tools")
	if !tools.IsArray() {
		return gjson.Result{}, false
	}
	var found gjson.Result
	ok := false
	tools.ForEach(func(_, item gjson.Result) bool {
		if openAIJSONString(item.Get("type")) == "image_generation" {
			found = item
			ok = true
			return false
		}
		return true
	})
	return found, ok
}

func extractOpenAIResponsesImagePromptAndImages(body []byte) (string, []string) {
	input := gjson.GetBytes(body, "input")
	if input.Type == gjson.String {
		return input.String(), nil
	}
	if !input.IsArray() {
		return "", nil
	}
	var prompt string
	var images []string
	input.ForEach(func(_, item gjson.Result) bool {
		role := strings.TrimSpace(item.Get("role").String())
		if role != "user" {
			return true
		}
		text, itemImages := extractOpenAIResponsesContentTextAndImages(item.Get("content"))
		if strings.TrimSpace(text) != "" {
			prompt = text
			images = itemImages
		}
		return true
	})
	return prompt, images
}

func extractOpenAIResponsesContentTextAndImages(content gjson.Result) (string, []string) {
	if content.Type == gjson.String {
		return content.String(), nil
	}
	if !content.IsArray() {
		return "", nil
	}
	var parts []string
	var images []string
	content.ForEach(func(_, part gjson.Result) bool {
		switch strings.TrimSpace(part.Get("type").String()) {
		case "input_text", "text":
			if text := strings.TrimSpace(part.Get("text").String()); text != "" {
				parts = append(parts, text)
			}
		case "input_image":
			if imageURL := strings.TrimSpace(part.Get("image_url").String()); imageURL != "" {
				images = append(images, imageURL)
			}
		}
		return true
	})
	return strings.Join(parts, "\n"), images
}

func (s *OpenAIGatewayService) ParseResponsesImageRedirectRequest(c *gin.Context, body []byte) ([]byte, *OpenAIImagesRequest, error) {
	imagesBody, endpoint, err := BuildOpenAIImagesRequestFromResponses(body)
	if err != nil {
		return nil, nil, err
	}
	if c == nil || c.Request == nil || c.Request.URL == nil {
		return nil, nil, fmt.Errorf("missing request context")
	}
	originalPath := c.Request.URL.Path
	originalHeader := c.Request.Header.Get("Content-Type")
	c.Request.URL.Path = endpoint
	c.Request.Header.Set("Content-Type", "application/json")
	parsed, parseErr := s.ParseOpenAIImagesRequest(c, imagesBody)
	c.Request.URL.Path = originalPath
	if originalHeader != "" {
		c.Request.Header.Set("Content-Type", originalHeader)
	} else {
		c.Request.Header.Del("Content-Type")
	}
	if parseErr != nil {
		return nil, nil, parseErr
	}
	return imagesBody, parsed, nil
}

func (s *OpenAIGatewayService) ForwardResponsesImageRedirect(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	body []byte,
	parsed *OpenAIImagesRequest,
	channelMappedModel string,
) (*OpenAIResponsesImageRedirectResult, error) {
	if parsed == nil {
		return nil, fmt.Errorf("parsed images request is required")
	}
	if account == nil {
		return nil, fmt.Errorf("account is required")
	}
	switch account.Type {
	case AccountTypeAPIKey:
		return s.forwardResponsesImageRedirectAPIKey(ctx, c, account, body, parsed, channelMappedModel)
	case AccountTypeOAuth:
		return s.forwardResponsesImageRedirectOAuth(ctx, c, account, parsed, channelMappedModel)
	default:
		return nil, fmt.Errorf("unsupported account type: %s", account.Type)
	}
}

func (s *OpenAIGatewayService) forwardResponsesImageRedirectAPIKey(ctx context.Context, c *gin.Context, account *Account, body []byte, parsed *OpenAIImagesRequest, channelMappedModel string) (*OpenAIResponsesImageRedirectResult, error) {
	startTime := time.Now()
	requestModel := strings.TrimSpace(parsed.Model)
	if mapped := strings.TrimSpace(channelMappedModel); mapped != "" {
		requestModel = mapped
	}
	if err := validateOpenAIImagesModel(requestModel); err != nil {
		return nil, err
	}
	upstreamModel := account.GetMappedModel(requestModel)
	if err := validateOpenAIImagesModel(upstreamModel); err != nil {
		return nil, err
	}
	forwardBody, forwardContentType, err := rewriteOpenAIImagesModel(body, parsed.ContentType, upstreamModel)
	if err != nil {
		return nil, err
	}
	upstreamCtx, releaseUpstreamCtx := detachStreamUpstreamContext(ctx, false)
	defer releaseUpstreamCtx()
	token, _, err := s.GetAccessToken(upstreamCtx, account)
	if err != nil {
		return nil, err
	}
	upstreamReq, err := s.buildOpenAIImagesRequest(upstreamCtx, c, account, forwardBody, forwardContentType, token, parsed.Endpoint)
	if err != nil {
		return nil, err
	}
	resp, err := s.doOpenAIImagesUpstream(upstreamReq, c, account)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 400 {
		respBody := s.readUpstreamErrorBody(resp)
		_ = resp.Body.Close()
		resp.Body = io.NopCloser(bytes.NewReader(respBody))
		upstreamMsg := sanitizeUpstreamErrorMessage(strings.TrimSpace(extractUpstreamErrorMessage(respBody)))
		if s.shouldFailoverOpenAIUpstreamResponse(resp.StatusCode, upstreamMsg, respBody) {
			s.handleFailoverSideEffects(upstreamCtx, resp, account, respBody, upstreamModel)
			return nil, &UpstreamFailoverError{StatusCode: resp.StatusCode, ResponseBody: respBody, RetryableOnSameAccount: account.IsPoolMode() && account.IsPoolModeRetryableStatus(resp.StatusCode)}
		}
		upErr := openAIImagesUpstreamErrorFromHTTP(resp.StatusCode, resp.Header, respBody)
		setOpsUpstreamError(c, upErr.clientStatusCode(), upErr.clientMessage(), "")
		return nil, upErr
	}
	bodyBytes, err := ReadUpstreamResponseBody(resp.Body, s.cfg, c, openAITooLargeError)
	if err != nil {
		return nil, err
	}
	usage, _ := extractOpenAIUsageFromJSONBytes(bodyBytes)
	results, createdAt, firstMeta := collectImagesAPIResults(bodyBytes, requestModel)
	if len(results) == 0 {
		return nil, fmt.Errorf("upstream did not return image output")
	}
	imageOutputSizes := openAIResponsesImageResultSizes(results)
	result := &OpenAIForwardResult{
		RequestID:        resp.Header.Get("x-request-id"),
		Usage:            usage,
		Model:            requestModel,
		UpstreamModel:    upstreamModel,
		Stream:           false,
		ResponseHeaders:  resp.Header.Clone(),
		Duration:         time.Since(startTime),
		ImageCount:       len(results),
		ImageSize:        parsed.SizeTier,
		ImageInputSize:   parsed.Size,
		ImageOutputSizes: imageOutputSizes,
	}
	responseBody := buildOpenAIResponsesImageRedirectBody("", requestModel, createdAt, results, firstMeta, usage)
	return &OpenAIResponsesImageRedirectResult{Result: result, ResponseBody: responseBody, ResponseID: extractOpenAIResponseIDFromJSONBytes(responseBody), CreatedAt: createdAt, ImageResults: results, FirstMeta: firstMeta, ImageRequest: parsed, UpstreamAccount: account}, nil
}

func (s *OpenAIGatewayService) doOpenAIImagesUpstream(req *http.Request, c *gin.Context, account *Account) (*http.Response, error) {
	proxyURL := ""
	if account != nil && account.ProxyID != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	upstreamStart := time.Now()
	resp, err := s.httpUpstream.Do(req, proxyURL, account.ID, account.Concurrency)
	SetOpsLatencyMs(c, OpsUpstreamLatencyMsKey, time.Since(upstreamStart).Milliseconds())
	if err != nil {
		safeErr := sanitizeUpstreamErrorMessage(err.Error())
		setOpsUpstreamError(c, 0, safeErr, "")
		return nil, fmt.Errorf("upstream request failed: %s", safeErr)
	}
	return resp, nil
}

func (s *OpenAIGatewayService) forwardResponsesImageRedirectOAuth(ctx context.Context, c *gin.Context, account *Account, parsed *OpenAIImagesRequest, channelMappedModel string) (*OpenAIResponsesImageRedirectResult, error) {
	startTime := time.Now()
	requestModel := strings.TrimSpace(parsed.Model)
	if mapped := strings.TrimSpace(channelMappedModel); mapped != "" {
		requestModel = mapped
	}
	if requestModel == "" {
		requestModel = "gpt-image-2"
	}
	if err := validateOpenAIImagesModel(requestModel); err != nil {
		return nil, err
	}
	upstreamCtx, releaseUpstreamCtx := detachUpstreamContext(ctx)
	defer releaseUpstreamCtx()
	token, _, err := s.GetAccessToken(upstreamCtx, account)
	if err != nil {
		return nil, err
	}
	responsesBody, err := buildOpenAIImagesResponsesRequest(parsed, requestModel)
	if err != nil {
		return nil, err
	}
	upstreamReq, err := s.buildUpstreamRequest(upstreamCtx, c, account, responsesBody, token, true, parsed.StickySessionSeed(), false)
	if err != nil {
		return nil, err
	}
	upstreamReq.Header.Set("Content-Type", "application/json")
	upstreamReq.Header.Set("Accept", "text/event-stream")
	resp, err := s.doOpenAIImagesUpstream(upstreamReq, c, account)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 400 {
		respBody := s.readUpstreamErrorBody(resp)
		_ = resp.Body.Close()
		resp.Body = io.NopCloser(bytes.NewReader(respBody))
		upstreamMsg := sanitizeUpstreamErrorMessage(strings.TrimSpace(extractUpstreamErrorMessage(respBody)))
		if s.shouldFailoverOpenAIUpstreamResponse(resp.StatusCode, upstreamMsg, respBody) {
			s.handleFailoverSideEffects(upstreamCtx, resp, account, respBody, requestModel)
			return nil, &UpstreamFailoverError{StatusCode: resp.StatusCode, ResponseBody: respBody, RetryableOnSameAccount: account.IsPoolMode() && account.IsPoolModeRetryableStatus(resp.StatusCode)}
		}
		upErr := openAIImagesUpstreamErrorFromHTTP(resp.StatusCode, resp.Header, respBody)
		setOpsUpstreamError(c, upErr.clientStatusCode(), upErr.clientMessage(), "")
		return nil, upErr
	}
	bodyBytes, err := ReadUpstreamResponseBody(resp.Body, s.cfg, c, openAITooLargeError)
	if err != nil {
		return nil, err
	}
	var usage OpenAIUsage
	forEachOpenAISSEDataPayload(string(bodyBytes), func(data []byte) {
		s.parseSSEUsageBytes(data, &usage)
	})
	results, createdAt, _, firstMeta, _, err := collectOpenAIImagesFromResponsesBody(bodyBytes)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		if upstreamErr := extractOpenAIImagesUpstreamError(bodyBytes); upstreamErr != nil {
			setOpsUpstreamError(c, upstreamErr.clientStatusCode(), upstreamErr.clientMessage(), "")
			return nil, upstreamErr
		}
		setOpsUpstreamError(c, http.StatusBadGateway, "upstream did not return image output", summarizeOpenAIImagesNoOutputBody(bodyBytes))
		return nil, &UpstreamFailoverError{StatusCode: http.StatusBadGateway, ResponseBody: bodyBytes, RetryableOnSameAccount: true}
	}
	if strings.TrimSpace(firstMeta.Model) == "" {
		firstMeta.Model = strings.TrimSpace(requestModel)
	}
	result := &OpenAIForwardResult{
		RequestID:        resp.Header.Get("x-request-id"),
		Usage:            usage,
		Model:            requestModel,
		UpstreamModel:    requestModel,
		Stream:           false,
		ResponseHeaders:  resp.Header.Clone(),
		Duration:         time.Since(startTime),
		ImageCount:       len(results),
		ImageSize:        parsed.SizeTier,
		ImageInputSize:   parsed.Size,
		ImageOutputSizes: openAIResponsesImageResultSizes(results),
	}
	responseBody := buildOpenAIResponsesImageRedirectBody("", requestModel, createdAt, results, firstMeta, usage)
	return &OpenAIResponsesImageRedirectResult{Result: result, ResponseBody: responseBody, ResponseID: extractOpenAIResponseIDFromJSONBytes(responseBody), CreatedAt: createdAt, ImageResults: results, FirstMeta: firstMeta, ImageRequest: parsed, UpstreamAccount: account}, nil
}

func collectImagesAPIResults(body []byte, fallbackModel string) ([]openAIResponsesImageResult, int64, openAIResponsesImageResult) {
	createdAt := gjson.GetBytes(body, "created").Int()
	if createdAt <= 0 {
		createdAt = time.Now().Unix()
	}
	var results []openAIResponsesImageResult
	for _, item := range gjson.GetBytes(body, "data").Array() {
		result := strings.TrimSpace(item.Get("b64_json").String())
		if result == "" {
			if url := strings.TrimSpace(item.Get("url").String()); strings.HasPrefix(url, "data:") {
				if idx := strings.Index(url, ";base64,"); idx >= 0 {
					result = url[idx+len(";base64,"):]
				}
			}
		}
		if result == "" {
			continue
		}
		results = append(results, openAIResponsesImageResult{
			Result:        result,
			RevisedPrompt: strings.TrimSpace(item.Get("revised_prompt").String()),
			OutputFormat:  strings.TrimSpace(firstNonEmptyString(item.Get("output_format").String(), gjson.GetBytes(body, "output_format").String())),
			Size:          strings.TrimSpace(firstNonEmptyString(item.Get("size").String(), gjson.GetBytes(body, "size").String())),
			Background:    strings.TrimSpace(firstNonEmptyString(item.Get("background").String(), gjson.GetBytes(body, "background").String())),
			Quality:       strings.TrimSpace(firstNonEmptyString(item.Get("quality").String(), gjson.GetBytes(body, "quality").String())),
			Model:         strings.TrimSpace(firstNonEmptyString(item.Get("model").String(), gjson.GetBytes(body, "model").String(), fallbackModel)),
		})
	}
	firstMeta := openAIResponsesImageResult{Model: strings.TrimSpace(fallbackModel)}
	if len(results) > 0 {
		firstMeta = results[0]
	}
	return results, createdAt, firstMeta
}

func buildOpenAIResponsesImageRedirectBody(responseID, model string, createdAt int64, results []openAIResponsesImageResult, firstMeta openAIResponsesImageResult, usage OpenAIUsage) []byte {
	if responseID == "" {
		responseID = "resp_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	}
	if createdAt <= 0 {
		createdAt = time.Now().Unix()
	}
	body := []byte(`{"id":"","object":"response","created_at":0,"status":"completed","model":"","output":[]}`)
	body, _ = sjson.SetBytes(body, "id", responseID)
	body, _ = sjson.SetBytes(body, "created_at", createdAt)
	body, _ = sjson.SetBytes(body, "model", strings.TrimSpace(model))
	for idx, img := range results {
		item := []byte(`{"id":"","type":"image_generation_call","status":"completed","result":""}`)
		item, _ = sjson.SetBytes(item, "id", fmt.Sprintf("ig_%d", idx+1))
		item, _ = sjson.SetBytes(item, "result", img.Result)
		if img.RevisedPrompt != "" {
			item, _ = sjson.SetBytes(item, "revised_prompt", img.RevisedPrompt)
		}
		if img.OutputFormat != "" {
			item, _ = sjson.SetBytes(item, "output_format", img.OutputFormat)
		}
		if img.Size != "" {
			item, _ = sjson.SetBytes(item, "size", img.Size)
		}
		if img.Background != "" {
			item, _ = sjson.SetBytes(item, "background", img.Background)
		}
		if img.Quality != "" {
			item, _ = sjson.SetBytes(item, "quality", img.Quality)
		}
		body, _ = sjson.SetRawBytes(body, "output.-1", item)
	}
	if firstMeta.OutputFormat != "" || firstMeta.Size != "" || firstMeta.Background != "" || firstMeta.Quality != "" || firstMeta.Model != "" {
		tool := []byte(`{"type":"image_generation"}`)
		if firstMeta.Model != "" {
			tool, _ = sjson.SetBytes(tool, "model", firstMeta.Model)
		}
		if firstMeta.OutputFormat != "" {
			tool, _ = sjson.SetBytes(tool, "output_format", firstMeta.OutputFormat)
		}
		if firstMeta.Size != "" {
			tool, _ = sjson.SetBytes(tool, "size", firstMeta.Size)
		}
		if firstMeta.Background != "" {
			tool, _ = sjson.SetBytes(tool, "background", firstMeta.Background)
		}
		if firstMeta.Quality != "" {
			tool, _ = sjson.SetBytes(tool, "quality", firstMeta.Quality)
		}
		body, _ = sjson.SetRawBytes(body, "tools", []byte(`[]`))
		body, _ = sjson.SetRawBytes(body, "tools.-1", tool)
	}
	if usage.InputTokens > 0 || usage.OutputTokens > 0 || usage.ImageOutputTokens > 0 || usage.CacheReadInputTokens > 0 {
		body, _ = sjson.SetBytes(body, "usage.input_tokens", usage.InputTokens)
		body, _ = sjson.SetBytes(body, "usage.output_tokens", usage.OutputTokens)
		body, _ = sjson.SetBytes(body, "usage.total_tokens", usage.InputTokens+usage.OutputTokens)
		if usage.CacheReadInputTokens > 0 {
			body, _ = sjson.SetBytes(body, "usage.input_tokens_details.cached_tokens", usage.CacheReadInputTokens)
		}
		if usage.ImageOutputTokens > 0 {
			body, _ = sjson.SetBytes(body, "usage.output_tokens_details.image_tokens", usage.ImageOutputTokens)
		}
	}
	return body
}

func BuildOpenAIResponsesImageRedirectSSE(body []byte) []byte {
	responseID := strings.TrimSpace(gjson.GetBytes(body, "id").String())
	created := gjson.GetBytes(body, "created_at").Int()
	model := strings.TrimSpace(gjson.GetBytes(body, "model").String())
	createdEvent := []byte(`{"type":"response.created","response":{"id":"","object":"response","created_at":0,"status":"in_progress","model":"","output":[]}}`)
	createdEvent, _ = sjson.SetBytes(createdEvent, "response.id", responseID)
	createdEvent, _ = sjson.SetBytes(createdEvent, "response.created_at", created)
	createdEvent, _ = sjson.SetBytes(createdEvent, "response.model", model)
	var out bytes.Buffer
	fmt.Fprintf(&out, "event: response.created\ndata: %s\n\n", createdEvent)
	output := gjson.GetBytes(body, "output")
	if output.IsArray() {
		for idx, item := range output.Array() {
			ev := []byte(`{"type":"response.output_item.done","output_index":0,"item":{}}`)
			ev, _ = sjson.SetBytes(ev, "output_index", idx)
			ev, _ = sjson.SetRawBytes(ev, "item", []byte(item.Raw))
			fmt.Fprintf(&out, "event: response.output_item.done\ndata: %s\n\n", ev)
		}
	}
	completed := []byte(`{"type":"response.completed","response":{}}`)
	completed, _ = sjson.SetRawBytes(completed, "response", body)
	fmt.Fprintf(&out, "event: response.completed\ndata: %s\n\n", completed)
	return out.Bytes()
}
