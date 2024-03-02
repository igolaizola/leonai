package leonardo

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/igolaizola/leonai/pkg/ratelimit"
	"github.com/igolaizola/leonai/pkg/session"
)

type Client struct {
	client          *http.Client
	debug           bool
	ratelimit       ratelimit.Lock
	token           string
	tokenExpiration time.Time
	cookieStore     CookieStore
	userID          string
}

type Config struct {
	Wait        time.Duration
	Debug       bool
	Client      *http.Client
	CookieStore CookieStore
}

type cookieStore struct {
	path string
}

func (c *cookieStore) GetCookie(ctx context.Context) (string, error) {
	b, err := os.ReadFile(c.path)
	if err != nil {
		return "", fmt.Errorf("leonardo: couldn't read cookie: %w", err)
	}
	return string(b), nil
}

func (c *cookieStore) SetCookie(ctx context.Context, cookie string) error {
	if err := os.WriteFile(c.path, []byte(cookie), 0644); err != nil {
		return fmt.Errorf("leonardo: couldn't write cookie: %w", err)
	}
	return nil
}

func NewCookieStore(path string) CookieStore {
	return &cookieStore{
		path: path,
	}
}

type CookieStore interface {
	GetCookie(context.Context) (string, error)
	SetCookie(context.Context, string) error
}

func New(cfg *Config) *Client {
	wait := cfg.Wait
	if wait == 0 {
		wait = 1 * time.Second
	}
	client := cfg.Client
	if client == nil {
		client = &http.Client{
			Timeout: 2 * time.Minute,
		}
	}
	return &Client{
		client:      client,
		ratelimit:   ratelimit.New(wait),
		debug:       cfg.Debug,
		cookieStore: cfg.CookieStore,
	}
}

func (c *Client) Start(ctx context.Context) error {
	// Get cookie
	cookie, err := c.cookieStore.GetCookie(ctx)
	if err != nil {
		return err
	}
	if cookie == "" {
		return fmt.Errorf("leonardo: cookie is empty")
	}
	if err := session.SetCookies(c.client, "https://app.leonardo.ai", cookie, nil); err != nil {
		return fmt.Errorf("leonardo: couldn't set cookie: %w", err)
	}

	// Authenticate
	if err := c.Auth(ctx); err != nil {
		return err
	}

	// Get user id
	cls, err := toClaims(c.token)
	if err != nil {
		return err
	}
	userID, err := c.user(ctx, cls.Sub)
	if err != nil {
		return err
	}
	if userID != cls.HasuraClaims.XHasuraUserID {
		return fmt.Errorf("leonardo: user id mismatch: %s != %s", userID, cls.HasuraClaims.XHasuraUserID)
	}
	c.userID = userID

	return nil
}

func (c *Client) Auth(ctx context.Context) error {
	if c.token != "" && time.Now().Before(c.tokenExpiration) {
		return nil
	}
	token, expiration, err := c.session(ctx)
	if err != nil {
		return err
	}
	c.token = token
	// Set token expiration to 90% of the actual expiration
	c.tokenExpiration = time.Now().Add(expiration.Sub(time.Now().UTC()) * 90 / 100).UTC()
	return nil
}

func (c *Client) Stop(ctx context.Context) error {
	cookie, err := session.GetCookies(c.client, "https://app.leonardo.ai")
	if err != nil {
		return fmt.Errorf("leonardo: couldn't get cookie: %w", err)
	}
	if err := c.cookieStore.SetCookie(ctx, cookie); err != nil {
		return err
	}
	return nil
}

type sessionResponse struct {
	User struct {
		Name  string `json:"name"`
		Email string `json:"email"`
		Sub   string `json:"sub"`
	} `json:"user"`
	Expires             string `json:"expires"`
	AccessToken         string `json:"accessToken"`
	AccessTokenIssuedAt int    `json:"accessTokenIssuedAt"`
	AccessTokenExpiry   int    `json:"accessTokenExpiry"`
	ServerTimestamp     int    `json:"serverTimestamp"`
}

type claims struct {
	HasuraClaimsRaw string `json:"https://hasura.io/jwt/claims"`
	HasuraClaims    hasuraClaims
	Sub             string `json:"sub"`
}

type hasuraClaims struct {
	XHasuraUserID string `json:"x-hasura-user-id"`
}

func toClaims(token string) (*claims, error) {
	// Split the JWT into its three parts
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, errors.New("leonardo: invalid access token")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("leonardo: couldn't decode access token: %w", err)
	}
	var c claims
	if err := json.Unmarshal(payload, &c); err != nil {
		return nil, fmt.Errorf("leonardo: couldn't unmarshal access token: %w", err)
	}
	var hs hasuraClaims
	if err := json.Unmarshal([]byte(c.HasuraClaimsRaw), &hs); err != nil {
		return nil, fmt.Errorf("leonardo: couldn't unmarshal hasura claims: %w", err)
	}
	c.HasuraClaims = hs
	if hs.XHasuraUserID == "" {
		return nil, errors.New("leonardo: empty hasura user id")
	}
	if c.Sub == "" {
		return nil, errors.New("leonardo: empty sub")
	}
	return &c, nil
}

func (c *Client) session(ctx context.Context) (string, time.Time, error) {
	var resp sessionResponse
	if _, err := c.do(ctx, "GET", "api/auth/session", nil, &resp); err != nil {
		return "", time.Time{}, fmt.Errorf("leonardo: couldn't get session: %w", err)
	}
	if resp.AccessToken == "" {
		return "", time.Time{}, errors.New("leonardo: empty access token")
	}
	expiration := time.Unix(int64(resp.AccessTokenExpiry), 0)
	return resp.AccessToken, expiration, nil
}

type graphqlRequest struct {
	OperationName string         `json:"operationName"`
	Variables     map[string]any `json:"variables"`
	Query         string         `json:"query"`
}

type userResponse struct {
	Data struct {
		Users []struct {
			ID          string `json:"id"`
			Username    string `json:"username"`
			Blocked     bool   `json:"blocked"`
			UserDetails []struct {
				Auth0Email                     string   `json:"auth0Email"`
				Plan                           string   `json:"plan"`
				PaidTokens                     int      `json:"paidTokens"`
				ApiCredit                      int      `json:"apiCredit"`
				SubscriptionTokens             int      `json:"subscriptionTokens"`
				SubscriptionModelTokens        int      `json:"subscriptionModelTokens"`
				SubscriptionGptTokens          int      `json:"subscriptionGptTokens"`
				SubscriptionSource             string   `json:"subscriptionSource"`
				Interests                      []string `json:"interests"`
				InterestsRoles                 string   `json:"interestsRoles"`
				InterestsRolesOther            string   `json:"interestsRolesOther"`
				ShowNsfw                       bool     `json:"showNsfw"`
				TokenRenewalDate               string   `json:"tokenRenewalDate"`
				PlanSubscribeFrequency         string   `json:"planSubscribeFrequency"`
				ApiSubscriptionTokens          any      `json:"apiSubscriptionTokens"`
				ApiPaidTokens                  any      `json:"apiPaidTokens"`
				ApiPlan                        any      `json:"apiPlan"`
				PaddleId                       any      `json:"paddleId"`
				ApiPlanAutoTopUpTriggerBalance any      `json:"apiPlanAutoTopUpTriggerBalance"`
				ApiPlanSubscribeFrequency      any      `json:"apiPlanSubscribeFrequency"`
				ApiPlanSubscribeDate           any      `json:"apiPlanSubscribeDate"`
				ApiPlanSubscriptionSource      any      `json:"apiPlanSubscriptionSource"`
				ApiPlanTokenRenewalDate        any      `json:"apiPlanTokenRenewalDate"`
				ApiPlanTopUpAmount             any      `json:"apiPlanTopUpAmount"`
				ApiConcurrencySlots            int      `json:"apiConcurrencySlots"`
				Typename                       string   `json:"__typename"`
			} `json:"user_details"`
			TeamMemberships []any  `json:"team_memberships"`
			Typename        string `json:"__typename"`
		} `json:"users"`
	} `json:"data"`
}

func (c *Client) user(ctx context.Context, sub string) (string, error) {
	req := &graphqlRequest{
		OperationName: "GetUserDetails",
		Variables: map[string]any{
			"userSub": sub,
		},
		Query: userQuery,
	}

	var resp userResponse
	if _, err := c.do(ctx, "POST", "graphql", req, &resp); err != nil {
		return "", err
	}
	if len(resp.Data.Users) == 0 {
		return "", errors.New("leonardo: no users found")
	}
	if resp.Data.Users[0].ID == "" {
		return "", errors.New("leonardo: empty user id")
	}
	return resp.Data.Users[0].ID, nil
}

type createUploadResponse struct {
	Data struct {
		UploadInitImage struct {
			ID     string `json:"id"`
			Fields string `json:"fields"`
			Key    string `json:"key"`
			URL    string `json:"url"`
		} `json:"uploadInitImage"`
	} `json:"data"`
}

type uploadFields struct {
	ContentType string `json:"Content-Type"`
	Bucket      string `json:"bucket"`
	Algorithm   string `json:"X-Amz-Algorithm"`
	Credential  string `json:"X-Amz-Credential"`
	Date        string `json:"X-Amz-Date"`
	Security    string `json:"X-Amz-Security-Token"`
	Key         string `json:"key"`
	Policy      string `json:"Policy"`
	Signature   string `json:"X-Amz-Signature"`
}

var webkitChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"

func webkitID(length int) string {
	b := make([]byte, length)
	_, _ = rand.Read(b) // generates len(b) random bytes
	for i := 0; i < length; i++ {
		b[i] = webkitChars[int(b[i])%len(webkitChars)]
	}
	return string(b)
}

func (c *Client) Upload(ctx context.Context, path string) (string, error) {
	// Authenticate if necessary
	if err := c.Auth(ctx); err != nil {
		return "", err
	}

	ext := filepath.Ext(path)
	if ext == "" {
		return "", fmt.Errorf("leonardo: couldn't get file extension")
	}
	ext = ext[1:]
	var fileType string
	switch ext {
	case "jpg", "jpeg":
		fileType = "image/jpeg"
	case "png":
		fileType = "image/png"
	default:
		return "", fmt.Errorf("leonardo: unsupported file extension: %s", ext)
	}

	// Check if file exists
	if _, err := os.Stat(path); err != nil {
		return "", fmt.Errorf("leonardo: couldn't stat file: %w", err)
	}

	req := &graphqlRequest{
		OperationName: "CreateUploadInitImage",
		Variables: map[string]any{
			"arg1": map[string]any{
				"fileType":  fileType,
				"extension": ext,
			},
		},
		Query: uploadQuery,
	}

	var resp createUploadResponse
	if _, err := c.do(ctx, "POST", "graphql", req, &resp); err != nil {
		return "", err
	}

	u := resp.Data.UploadInitImage.URL
	if u == "" {
		return "", fmt.Errorf("leonardo: couldn't get upload url")
	}

	var fields uploadFields
	if err := json.Unmarshal([]byte(resp.Data.UploadInitImage.Fields), &fields); err != nil {
		return "", fmt.Errorf("leonardo: couldn't unmarshal fields: %w", err)
	}
	if fields.Key == "" {
		return "", fmt.Errorf("leonardo: couldn't get key")
	}

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	if err := writer.SetBoundary(fmt.Sprintf("----WebKitFormBoundary%s", webkitID(16))); err != nil {
		return "", fmt.Errorf("leonardo: couldn't set boundary: %w", err)
	}

	// Add fields
	kvs := []struct {
		key   string
		value string
	}{
		{key: "Content-Type", value: fields.ContentType},
		{key: "bucket", value: fields.Bucket},
		{key: "X-Amz-Algorithm", value: fields.Algorithm},
		{key: "X-Amz-Credential", value: fields.Credential},
		{key: "X-Amz-Date", value: fields.Date},
		{key: "X-Amz-Security-Token", value: fields.Security},
		{key: "key", value: fields.Key},
		{key: "Policy", value: fields.Policy},
		{key: "X-Amz-Signature", value: fields.Signature},
	}
	for _, kv := range kvs {
		if err := writer.WriteField(kv.key, kv.value); err != nil {
			return "", fmt.Errorf("leonardo: couldn't write field %s: %w", kv.key, err)
		}
	}

	part, err := writer.CreateFormFile("file", filepath.Base(path))
	if err != nil {
		return "", fmt.Errorf("leonardo: couldn't create form file: %w", err)
	}

	// Open file
	reader, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("leonardo: couldn't open file: %w", err)
	}
	defer reader.Close()

	// Copy file to part
	if _, err := io.Copy(part, reader); err != nil {
		return "", fmt.Errorf("leonardo: couldn't copy file to part: %w", err)
	}

	// Close writer
	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("leonardo: couldn't close writer: %w", err)
	}

	// Upload file
	f := &form{
		writer: writer,
		data:   &buf,
	}
	if _, err := c.do(ctx, "POST", u, f, nil); err != nil {
		return "", err
	}
	return resp.Data.UploadInitImage.ID, nil
}

type createGenerationResponse struct {
	Data struct {
		MotionSVDGenerationJob struct {
			APICreditCost int    `json:"apiCreditCost"`
			GenerationID  string `json:"generationId"`
		} `json:"motionSvdGenerationJob"`
	} `json:"data"`
}

type feedResponse struct {
	Data struct {
		Generations []generation `json:"generations"`
	} `json:"data"`
}

type generation struct {
	Alchemy             any    `json:"alchemy"`
	ContrastRatio       any    `json:"contrastRatio"`
	HighResolution      any    `json:"highResolution"`
	GuidanceScale       any    `json:"guidanceScale"`
	InferenceSteps      any    `json:"inferenceSteps"`
	ModelId             any    `json:"modelId"`
	Scheduler           any    `json:"scheduler"`
	CoreModel           string `json:"coreModel"`
	SdVersion           any    `json:"sdVersion"`
	Prompt              string `json:"prompt"`
	NegativePrompt      any    `json:"negativePrompt"`
	ID                  string `json:"id"`
	Status              string `json:"status"`
	Quantity            int    `json:"quantity"`
	CreatedAt           string `json:"createdAt"`
	ImageHeight         int    `json:"imageHeight"`
	ImageWidth          int    `json:"imageWidth"`
	PresetStyle         any    `json:"presetStyle"`
	Public              bool   `json:"public"`
	Seed                int64  `json:"seed"`
	Tiling              any    `json:"tiling"`
	InitStrength        any    `json:"initStrength"`
	ImageToImage        bool   `json:"imageToImage"`
	HighContrast        bool   `json:"highContrast"`
	PromptMagic         bool   `json:"promptMagic"`
	PromptMagicVersion  any    `json:"promptMagicVersion"`
	PromptMagicStrength any    `json:"promptMagicStrength"`
	ImagePromptStrength any    `json:"imagePromptStrength"`
	ExpandedDomain      any    `json:"expandedDomain"`
	Motion              bool   `json:"motion"`
	PhotoReal           any    `json:"photoReal"`
	PhotoRealStrength   any    `json:"photoRealStrength"`
	Nsfw                bool   `json:"nsfw"`
	User                struct {
		Username string `json:"username"`
		ID       string `json:"id"`
		Typename string `json:"__typename"`
	} `json:"user"`
	CustomModel *string `json:"custom_model"`
	InitImage   struct {
		ID       string `json:"id"`
		URL      string `json:"url"`
		Typename string `json:"__typename"`
	} `json:"init_image"`
	GeneratedImages []struct {
		ID                              string        `json:"id"`
		URL                             string        `json:"url"`
		MotionGIFURL                    *string       `json:"motionGIFURL"`
		MotionMP4URL                    *string       `json:"motionMP4URL"`
		LikeCount                       int           `json:"likeCount"`
		Nsfw                            bool          `json:"nsfw"`
		GeneratedImageVariationGenerics []interface{} `json:"generated_image_variation_generics"`
		Typename                        string        `json:"__typename"`
	} `json:"generated_images"`
	GenerationElements    []interface{} `json:"generation_elements"`
	GenerationControlnets []interface{} `json:"generation_controlnets"`
	Typename              string        `json:"__typename"`
}

type statusResponse struct {
	Data struct {
		Generations []generationStatus `json:"generations"`
	} `json:"data"`
}

type generationStatus struct {
	ID       string `json:"id"`
	Status   string `json:"status"`
	Typename string `json:"__typename"`
}

func (c *Client) CreateMotion(ctx context.Context, id string, motionStrength int) (string, string, error) {
	// Authenticate if necessary
	if err := c.Auth(ctx); err != nil {
		return "", "", err
	}

	if motionStrength == 0 {
		motionStrength = 5
	}
	userID := c.userID
	if userID == "" {
		return "", "", errors.New("leonardo: empty user id")
	}

	createReq := &graphqlRequest{
		OperationName: "CreateMotionSvdGenerationJob",
		Variables: map[string]any{
			"arg1": map[string]any{
				"imageId":        id,
				"isPublic":       false,
				"isInitImage":    true,
				"isVariation":    false,
				"motionStrength": motionStrength,
			},
		},
		Query: createQuery,
	}

	var createResp createGenerationResponse
	if _, err := c.do(ctx, "POST", "graphql", createReq, &createResp); err != nil {
		return "", "", fmt.Errorf("leonardo: couldn't create motion: %w", err)
	}
	generationID := createResp.Data.MotionSVDGenerationJob.GenerationID
	if generationID == "" {
		return "", "", fmt.Errorf("leonardo: couldn't get generation id")
	}

	statusReq := &graphqlRequest{
		OperationName: "GetAIGenerationFeedStatuses",
		Variables: map[string]any{
			"where": map[string]any{
				"status": map[string]any{
					"_in": []string{"COMPLETE", "FAILED"},
				},
				"id": map[string]any{
					"_in": []string{generationID},
				},
			},
		},
		Query: statusQuery,
	}

	var last []byte
	for {
		select {
		case <-ctx.Done():
			log.Println("leonardo: context done, last response:", string(last))
			return "", "", ctx.Err()
		case <-time.After(5 * time.Second):
		}
		var statusResp statusResponse
		b, err := c.do(ctx, "POST", "graphql", statusReq, &statusResp)
		if err != nil {
			return "", "", fmt.Errorf("leonardo: couldn't get status: %w", err)
		}
		last = b
		if len(statusResp.Data.Generations) == 0 {
			continue
		}
		s := statusResp.Data.Generations[0]
		if s.Status != "COMPLETE" {
			return "", "", fmt.Errorf("leonardo: status generation %s", s.Status)
		}
		break
	}

	feedReq := &graphqlRequest{
		OperationName: "GetAIGenerationFeed",
		Variables: map[string]any{
			"where": map[string]any{
				"userId": map[string]any{
					"_eq": userID,
				},
				"teamId": map[string]any{
					"_is_null": true,
				},
				"canvasRequest": map[string]any{
					"_eq": false,
				},
				"universalUpscaler": map[string]any{
					"_is_null": true,
				},
				"isStoryboard": map[string]any{
					"_eq": false,
				},
			},
			"offset": 0,
			"limit":  10,
		},
		Query: feedQuery,
	}

	wait := 1 * time.Second
	var gen *generation
	for {
		select {
		case <-ctx.Done():
			return "", "", fmt.Errorf("leonardo: pending generation: %w", ctx.Err())
		case <-time.After(wait):
		}
		wait = 5 * time.Second
		var feedResp feedResponse
		if _, err := c.do(ctx, "POST", "graphql", feedReq, &feedResp); err != nil {
			return "", "", fmt.Errorf("leonardo: couldn't get feed: %w", err)
		}
		if len(feedResp.Data.Generations) == 0 {
			return "", "", errors.New("leonardo: no generations found")
		}
		var candidate *generation
		for _, g := range feedResp.Data.Generations {
			if g.ID != generationID {
				continue
			}
			candidate = &g
			break
		}
		if candidate == nil {
			return "", "", fmt.Errorf("leonardo: couldn't find generation %s", generationID)
		}
		switch candidate.Status {
		case "PENDING":
			continue
		case "COMPLETE":
		default:
			return "", "", fmt.Errorf("leonardo: feed generation %s", candidate.Status)
		}
		gen = candidate
		break
	}
	if len(gen.GeneratedImages) == 0 {
		return "", "", fmt.Errorf("leonardo: couldn't get generated images")
	}
	u := gen.GeneratedImages[0].MotionMP4URL
	if u == nil || *u == "" {
		return "", "", fmt.Errorf("leonardo: empty motion mp4 url")
	}
	id = gen.GeneratedImages[0].ID
	if id == "" {
		return "", "", fmt.Errorf("leonardo: empty generated image id")
	}
	return id, *u, nil
}

func (c *Client) log(format string, args ...interface{}) {
	if c.debug {
		format += "\n"
		log.Printf(format, args...)
	}
}

var backoff = []time.Duration{
	30 * time.Second,
	1 * time.Minute,
	2 * time.Minute,
}

func (c *Client) do(ctx context.Context, method, path string, in, out any) ([]byte, error) {
	maxAttempts := 3
	attempts := 0
	var err error
	for {
		if err != nil {
			log.Println("retrying...", err)
		}
		var b []byte
		b, err = c.doAttempt(ctx, method, path, in, out)
		if err == nil {
			return b, nil
		}
		// Increase attempts and check if we should stop
		attempts++
		if attempts >= maxAttempts {
			return nil, err
		}
		// If the error is temporary retry
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			continue
		}

		// Check if we should retry after waiting
		var retry bool

		// Check status code
		var errStatus errStatusCode
		if errors.As(err, &errStatus) {
			switch int(errStatus) {
			case http.StatusBadGateway, http.StatusGatewayTimeout, http.StatusTooManyRequests:
				// Retry on these status codes
				retry = true
			default:
				return nil, err
			}
		}

		// Check API error
		var errAPI errAPI
		if errors.As(err, &errAPI) {
			if errAPI.code == invalidJWTCode {
				// If the JWT is invalid we should re-authenticate
				if err := c.Auth(ctx); err != nil {
					return nil, err
				}
			}
			// Retry on any API error
			retry = true
		}

		if !retry {
			return nil, err
		}

		// Wait before retrying
		idx := attempts - 1
		if idx >= len(backoff) {
			idx = len(backoff) - 1
		}
		wait := backoff[idx]
		c.log("server seems to be down, waiting %s before retrying\n", wait)
		t := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-t.C:
		}
	}
}

type errorResponse struct {
	Errors []struct {
		Message    string `json:"message"`
		Extensions struct {
			Code string `json:"code"`
		} `json:"extensions"`
	} `json:"errors"`
}

type form struct {
	writer *multipart.Writer
	data   *bytes.Buffer
}

type errStatusCode int

func (e errStatusCode) Error() string {
	return fmt.Sprintf("%d", e)
}

// Known error codes
const (
	invalidJWTCode = "invalid-jwt"
)

type errAPI struct {
	code string
}

func (e errAPI) Error() string {
	return e.code
}

func (c *Client) doAttempt(ctx context.Context, method, path string, in, out any) ([]byte, error) {
	var body []byte
	var reqBody io.Reader
	contentType := "application/json"
	if f, ok := in.(*form); ok {
		reqBody = f.data
		contentType = f.writer.FormDataContentType()
	} else if in != nil {
		var err error
		body, err = json.Marshal(in)
		if err != nil {
			return nil, fmt.Errorf("leonardo: couldn't marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(body)
	}
	logBody := string(body)
	if len(logBody) > 100 {
		logBody = logBody[:100] + "..."
	}
	c.log("leonardo: do %s %s %s", method, path, logBody)

	// Check if path is absolute
	u := fmt.Sprintf("https://api.leonardo.ai/v1/%s", path)
	if strings.HasPrefix(path, "api") {
		u = fmt.Sprintf("https://app.leonardo.ai/%s", path)
	}
	if strings.HasPrefix(path, "http") {
		u = path
	}
	req, err := http.NewRequestWithContext(ctx, method, u, reqBody)
	if err != nil {
		return nil, fmt.Errorf("leonardo: couldn't create request: %w", err)
	}
	c.addHeaders(req, path, contentType)

	unlock := c.ratelimit.Lock(ctx)
	defer unlock()

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("leonardo: couldn't %s %s: %w", method, u, err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("leonardo: couldn't read response body: %w", err)
	}
	c.log("leonardo: response %s %s %d %s", method, path, resp.StatusCode, string(respBody))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		errMessage := string(respBody)
		if len(errMessage) > 100 {
			errMessage = errMessage[:100] + "..."
		}
		_ = os.WriteFile(fmt.Sprintf("logs/debug_%s.json", time.Now().Format("20060102_150405")), respBody, 0644)
		return nil, fmt.Errorf("leonardo: %s %s returned (%s): %w", method, u, errMessage, errStatusCode(resp.StatusCode))
	}
	if out != nil {
		var errResp errorResponse
		if err := json.Unmarshal(respBody, &errResp); err == nil && len(errResp.Errors) > 0 {
			var msgs []string
			for _, e := range errResp.Errors {
				msgs = append(msgs, fmt.Sprintf("%s (%s)", e.Message, e.Extensions.Code))
			}
			_ = os.WriteFile(fmt.Sprintf("logs/debug_%s.json", time.Now().Format("20060102_150405")), respBody, 0644)
			return nil, fmt.Errorf("leonardo: %s: %w", strings.Join(msgs, ", "), errAPI{code: errResp.Errors[0].Extensions.Code})
		}
		if err := json.Unmarshal(respBody, out); err != nil {
			// Write response body to file for debugging.
			_ = os.WriteFile(fmt.Sprintf("logs/debug_%s.json", time.Now().Format("20060102_150405")), respBody, 0644)
			return nil, fmt.Errorf("leonardo: couldn't unmarshal response body (%T): %w", out, err)
		}
	}
	return respBody, nil
}

func (c *Client) addHeaders(req *http.Request, path, contentType string) {
	switch {
	case strings.HasPrefix(contentType, "multipart/form-data"):
		req.Header.Set("Accept", "*")
		req.Header.Set("Accept-Language", "en-US,en;q=0.9")
		req.Header.Set("Connection", "keep-alive")
		req.Header.Set("Content-Type", contentType)
		req.Header.Set("Origin", "https://app.leonardo.ai")
		req.Header.Set("Referer", "https://app.leonardo.ai/")
		req.Header.Set("Sec-Fetch-Dest", "empty")
		req.Header.Set("Sec-Fetch-Mode", "cors")
		req.Header.Set("User-Agent", `Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36`)
		req.Header.Set("sec-ch-ua", `"Not A(Brand";v="99", "Google Chrome";v="121", "Chromium";v="121"`)
		req.Header.Set("sec-ch-ua-mobile", "?0")
		req.Header.Set("sec-ch-ua-platform", `"Windows"`)
	case strings.HasPrefix(path, "api"):
		req.Header.Set("Authority", "app.leonardo.ai")
		req.Header.Set("Accept", "*/*")
		req.Header.Set("Accept-Language", "en-US,en;q=0.9")
		// TODO: Check if this is necessary
		// req.Header.Set("Baggage", "sentry-environment=production,sentry-release=,sentry-public_key=,sentry-trace_id=")
		req.Header.Set("Content-Type", contentType)
		req.Header.Set("Origin", "https://app.leonardo.ai")
		req.Header.Set("Referer", "https://app.leonardo.ai/")
		req.Header.Set("sec-ch-ua", `"Not A(Brand";v="99", "Google Chrome";v="121", "Chromium";v="121"`)
		req.Header.Set("sec-ch-ua-mobile", "?0")
		req.Header.Set("sec-ch-ua-platform", `"Windows"`)
		req.Header.Set("sec-fetch-dest", "empty")
		req.Header.Set("sec-fetch-mode", "cors")
		req.Header.Set("sec-fetch-site", "same-origin")
		// TODO: Check if this is necessary
		// req.Header.Set("sentry-trace", "")
		req.Header.Set("user-agent", `Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36`)
	default:
		req.Header.Set("authority", "api.leonardo.ai")
		req.Header.Set("accept", "*/*")
		req.Header.Set("accept-language", "en-US,en;q=0.9")
		req.Header.Set("authorization", fmt.Sprintf("Bearer %s", c.token))
		req.Header.Set("content-yype", contentType)
		req.Header.Set("origin", "https://app.leonardo.ai")
		req.Header.Set("Referer", "https://app.leonardo.ai/")
		req.Header.Set("sec-fetch-dest", "empty")
		req.Header.Set("sec-fetch-mode", "cors")
		req.Header.Set("sec-fetch-site", "same-site")
		req.Header.Set("user-agent", `Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36`)
		req.Header.Set("sec-ch-ua", `"Not A(Brand";v="99", "Google Chrome";v="121", "Chromium";v="121"`)
		req.Header.Set("sec-ch-ua-mobile", "?0")
		req.Header.Set("sec-ch-ua-platform", `"Windows"`)
	}
}
