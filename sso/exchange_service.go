package sso

import (
	"context"
	"errors"
	"net/url"
	"slices"
	"sync"
	"time"
)

// ExchangeEnricher adds trusted session metadata without changing identity or scopes. ExchangeEnricher 补充可信会话元数据，不改变身份或授权范围。
type ExchangeEnricher func(context.Context, string, *TicketExchangeResult) error

// ExchangeOptions configures the transport-independent exchange service; register clients separately. ExchangeOptions 配置独立于传输的兑换服务，客户端须另行注册。
type ExchangeOptions struct {
	// Params names protocol fields, including signed extension parameters. Params 指定协议字段名称，保留签名扩展参数。
	Params ParamNames
	// CheckSign requires protocol signatures. CheckSign 要求协议签名。
	CheckSign bool
	// SecretKey is the fallback signing key. SecretKey 是默认签名密钥。
	SecretKey string
	// VerificationKey accepts a previous fallback key during rotation. VerificationKey 在轮换时接受旧默认密钥。
	VerificationKey string
	// AllowAnonymousClient selects the registered anonymous client's signing key when the ID is absent. AllowAnonymousClient 在 ID 缺失时选择已注册匿名客户端的签名密钥。
	AllowAnonymousClient bool
	// EnableSLO permits logout callback registration. EnableSLO 允许登记退出回调。
	EnableSLO bool
	// MaxRegisteredClient limits applications per account; negative disables the limit. MaxRegisteredClient 限制每个账号登记的应用数，负数禁用限制。
	MaxRegisteredClient int
	// RequestLimits bounds local form input using the same body budget as HTTP. RequestLimits 使用与 HTTP 相同的请求体预算限制本地表单。
	RequestLimits RequestLimits
	// ReplayProtection validates and claims signed request nonces. ReplayProtection 校验并消费签名请求随机值。
	ReplayProtection *ReplayProtection
	// AuthorizationPolicy evaluates the validated grant before consumption. AuthorizationPolicy 在消费前评估已校验的授权。
	AuthorizationPolicy AuthorizationPolicyFunc
	// EnrichExchangeResult runs after consumption and before callback registration. EnrichExchangeResult 在消费后及回调登记前执行。
	EnrichExchangeResult ExchangeEnricher
}

// DefaultExchangeOptions returns the existing center's exchange defaults. DefaultExchangeOptions 返回现有中心的兑换默认配置。
func DefaultExchangeOptions() ExchangeOptions {
	return ExchangeOptions{Params: DefaultParamNames(), CheckSign: true, EnableSLO: true, MaxRegisteredClient: 32, RequestLimits: DefaultRequestLimits()}
}

// ExchangeService validates and consumes one-time credentials without HTTP or resource ownership. ExchangeService 无需 HTTP 即可校验并消费一次性凭证，不拥有外部资源。
type ExchangeService struct {
	server          *Server
	options         ExchangeOptions
	clientSessionMu *sync.Mutex
}

// NewExchangeService validates options without registering clients or closing borrowed dependencies. NewExchangeService 校验配置，不注册客户端或关闭借用依赖。
func NewExchangeService(server *Server, options ExchangeOptions) (*ExchangeService, error) {
	if server == nil || server.storage == nil || server.serializer == nil {
		return nil, ErrServerNotInitialized
	}
	options.Params = normalizeParamNames(options.Params)
	options.RequestLimits = normalizeRequestLimits(options.RequestLimits)
	if options.MaxRegisteredClient == 0 {
		options.MaxRegisteredClient = 32
	}
	if err := options.Params.Validate(); err != nil {
		return nil, err
	}
	if err := validateVerificationKey(options.SecretKey, options.VerificationKey, "Exchange.VerificationKey"); err != nil {
		return nil, err
	}
	if options.VerificationKey != "" && !options.CheckSign {
		return nil, configError("Exchange.VerificationKey", "requires CheckSign", nil)
	}
	if err := options.ReplayProtection.validateConfig(options.CheckSign, options.Params); err != nil {
		return nil, err
	}
	return &ExchangeService{server: server, options: options, clientSessionMu: &sync.Mutex{}}, nil
}

// initializationError rejects nil and zero-value exchange services. initializationError 拒绝 nil 和零值兑换服务。
func (h *ExchangeService) initializationError() error {
	if h == nil || h.server == nil || h.server.storage == nil || h.server.serializer == nil || h.clientSessionMu == nil {
		return ErrServerNotInitialized
	}
	return nil
}

// Exchange authenticates normalized protocol fields before consuming a Ticket or code; never pass identity assertions. Exchange 在消费 Ticket 或授权码前认证规范化协议字段，调用方不得传入身份断言。
func (h *ExchangeService) Exchange(ctx context.Context, values url.Values) (*TicketExchangeResult, error) {
	if err := h.initializationError(); err != nil {
		return nil, err
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := checkFormSize(values, h.options.RequestLimits.MaxBodyBytes); err != nil {
		return nil, err
	}

	// Bound the encoded form as HTTP does, before signature and nonce consumption. 与 HTTP 一样在签名及随机值消费前限制编码后表单大小。
	if int64(len(values.Encode())) > h.options.RequestLimits.MaxBodyBytes {
		return nil, ErrRequestTooLarge
	}
	values = cloneValues(values)
	if err := h.verifyRequest(ctx, values, ActionToken, true); err != nil {
		return nil, err
	}
	return h.exchangeCredential(ctx, values)
}

// checkClientSessionLimit checks capacity while the caller holds clientSessionMu. checkClientSessionLimit 在调用方持有 clientSessionMu 时检查容量。
func (h *ExchangeService) checkClientSessionLimit(ctx context.Context, loginID, clientID string) error {
	limit := h.options.MaxRegisteredClient
	if limit > 0 {
		sessions, err := h.server.GetClientSessions(ctx, loginID)
		if err != nil {
			return err
		}
		registered := false
		for i := range sessions {
			if sessions[i].ClientID == clientID {
				registered = true
				break
			}
		}
		if !registered && len(sessions) >= limit {
			return ErrClientSessionLimit
		}
	}
	return nil
}

// authenticateClient validates the registered client and its optional secret. authenticateClient 校验已注册客户端及其可选密钥。
func (h *ExchangeService) authenticateClient(ctx context.Context, values url.Values) error {
	params := h.options.Params
	clientID := values.Get(params.Client)
	if clientID == "" {
		return ErrClientOrClientIDEmpty
	}
	client, err := h.server.getClient(ctx, clientID)
	if err != nil {
		return err
	}
	if !clientSecretMatches(client.ClientSecret, values.Get(params.ClientSecret)) {
		return ErrInvalidClientCredentials
	}
	return nil
}

// verifyRequest authenticates before checking freshness or recording a nonce. verifyRequest 在检查时效或记录随机值前完成认证。
func (h *ExchangeService) verifyRequest(ctx context.Context, values url.Values, action string, consume bool) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := validatePKCEParameters(values, h.options.Params, action == ActionAuthorize); err != nil {
		return err
	}
	if err := validateScalarParameters(values, h.options.Params); err != nil {
		return err
	}
	if err := validateCredentialParameters(values, h.options.Params, action); err != nil {
		return err
	}
	if err := h.verifySign(ctx, values); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	protection := h.options.ReplayProtection
	if protection == nil {
		return nil
	}
	if err := protection.validateConfig(h.options.CheckSign, h.options.Params); err != nil {
		return err
	}
	if consume {
		return protection.claim(ctx, values, h.options.Params, action)
	}
	return protection.validate(values, h.options.Params, action, time.Now())
}

// verifySign validates request signatures when enabled. verifySign 在启用时校验请求签名。
func (h *ExchangeService) verifySign(ctx context.Context, values url.Values) error {
	if !h.options.CheckSign {
		return nil
	}
	params := h.options.Params
	if len(values[params.Client]) > 1 {
		return ErrInvalidRequest
	}
	clientID := values.Get(params.Client)
	if clientID == "" && h.options.AllowAnonymousClient {
		clientID = ClientAnonymous
	}

	// Resolve the current registration on each request; lookup failures must not fall back to another key. 每次请求读取当前注册配置；查询失败时不能降级为其他密钥。
	var client *Client
	if clientID != "" {
		var err error
		client, err = h.server.getClient(ctx, clientID)
		if err != nil {
			return err
		}
	}
	secret, verification, err := h.signingKeys(client)
	if err != nil {
		return err
	}
	if !verifyProtocolSignature(values, params, secret, verification) {
		return ErrInvalidSign
	}
	return nil
}

// exchangeCredential dispatches credential exchange by mode. exchangeCredential 按模式分发凭证交换。
func (h *ExchangeService) exchangeCredential(ctx context.Context, values url.Values) (*TicketExchangeResult, error) {
	params := h.options.Params
	mode := Mode(values.Get(params.Mode))
	codeValue := values.Get(params.Code)
	ticketValue := values.Get(params.Ticket)

	// Resolve a single mode before consuming either credential. 消费凭证前先确定唯一模式。
	if codeValue != "" && ticketValue != "" {
		return nil, ErrInvalidRequest
	}
	if mode == "" {
		mode = ModeTicket
		if codeValue != "" {
			mode = ModeOAuth2
		}
	}
	if mode != ModeTicket && mode != ModeOAuth2 {
		return nil, ErrModeUnsupported
	}
	if mode == ModeTicket && codeValue != "" || mode == ModeOAuth2 && ticketValue != "" {
		return nil, ErrInvalidRequest
	}

	clientID := values.Get(params.Client)
	callbackURL := values.Get(params.Callback)
	registerCallback := h.options.EnableSLO && callbackURL != ""
	var authorization AuthorizationRequest
	var grantedScopes []string
	if registerCallback || h.options.AuthorizationPolicy != nil {
		var err error
		authorization, err = h.exchangeAuthorization(ctx, values, mode)
		if err != nil {
			return nil, err
		}
	}
	if registerCallback {
		// Check callback ownership and capacity before consuming a one-time credential. 消费一次性凭证前检查回调归属和登记容量。
		h.clientSessionMu.Lock()
		defer h.clientSessionMu.Unlock()
		client, err := h.server.getClient(ctx, clientID)
		if err != nil {
			return nil, err
		}
		if !h.server.isValidLogoutCallbackURL(client, callbackURL) {
			return nil, ErrInvalidCallbackURL
		}
		if err := h.checkClientSessionLimit(ctx, authorization.LoginID, clientID); err != nil {
			return nil, err
		}
	}
	if h.options.AuthorizationPolicy != nil {
		var err error
		grantedScopes, err = h.evaluateAuthorization(ctx, authorization)
		if err != nil {
			return nil, err
		}
	}

	var result *TicketExchangeResult
	if mode == ModeOAuth2 {
		code, err := h.server.ConsumeOAuth2CodeWithVerifier(
			ctx,
			codeValue,
			values.Get(params.Client),
			values.Get(params.ClientSecret),
			values.Get(params.Redirect),
			values.Get(params.CodeVerifier),
		)
		if err != nil {
			return nil, err
		}
		result = OAuth2CodeResult(code)
	} else {
		ticket, err := h.server.ConsumeTicketWithVerifier(
			ctx,
			ticketValue,
			values.Get(params.Client),
			values.Get(params.ClientSecret),
			values.Get(params.Redirect),
			values.Get(params.CodeVerifier),
		)
		if err != nil {
			return nil, err
		}
		result = TicketResult(ticket)
	}

	// Recheck the consumed payload against the policy input before applying a reduced grant. 应用缩减授权前，核对实际消费载荷与策略输入一致。
	if h.options.AuthorizationPolicy != nil {
		if result.LoginID != authorization.LoginID || !slices.Equal(result.Scopes, authorization.Scopes) {
			return nil, ErrAuthorizationDecision
		}
		result.Scopes = grantedScopes
	}

	// Allow application-owned session metadata without changing the credential's authorization. 允许应用补充自己管理的会话信息，但不能改变凭证授权。
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if h.options.EnrichExchangeResult != nil {
		loginID, centerID := result.LoginID, result.CenterID
		scopes := append([]string(nil), result.Scopes...)
		if err := h.options.EnrichExchangeResult(ctx, clientID, result); err != nil {
			return nil, err
		}
		if result.LoginID != loginID || result.CenterID != centerID || !slices.Equal(result.Scopes, scopes) {
			return nil, errors.New("exchange result hook changed credential identity or scopes")
		}
		if result.RemainTokenTimeout < -1 || result.RemainSessionTimeout < -1 {
			return nil, errors.New("exchange result hook returned an invalid session timeout")
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Bind only the identity returned by successful consumption. 仅登记成功消费凭证后返回的身份。
	if registerCallback {
		if _, err := h.server.RegisterClientSession(ctx, result.LoginID, clientID, callbackURL); err != nil {
			return nil, err
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

// evaluateAuthorization copies policy inputs and accepts only a subset of the original scopes. evaluateAuthorization 复制策略输入且仅接受原始范围的子集。
func (h *ExchangeService) evaluateAuthorization(ctx context.Context, request AuthorizationRequest) ([]string, error) {
	if h.options.AuthorizationPolicy == nil {
		return slices.Clone(request.Scopes), nil
	}
	if err := ctx.Err(); err != nil {
		return nil, &authorizationPolicyError{ErrAuthorizationPolicyUnavailable, err}
	}
	original := slices.Clone(request.Scopes)
	request.Scopes = slices.Clone(original)
	decision, err := h.options.AuthorizationPolicy(ctx, request)
	if ctx.Err() != nil {
		return nil, &authorizationPolicyError{ErrAuthorizationPolicyUnavailable, ctx.Err()}
	}
	if err != nil {
		return nil, &authorizationPolicyError{ErrAuthorizationPolicyUnavailable, err}
	}
	if !decision.Allowed {
		return nil, ErrAuthorizationDenied
	}
	if decision.Scopes == nil {
		return original, nil
	}
	for _, scope := range decision.Scopes {
		if !slices.Contains(original, scope) {
			return nil, ErrAuthorizationDecision
		}
	}

	// Retain original order and avoid introducing duplicates through policy output. 保留原始顺序，避免通过策略输出引入重复范围。
	granted := make([]string, 0, len(original))
	for _, scope := range original {
		if slices.Contains(decision.Scopes, scope) && !slices.Contains(granted, scope) {
			granted = append(granted, scope)
		}
	}
	return granted, nil
}

// exchangeAuthorization validates authentication, mode, bindings, and proof before exposing identity to policy or callback registration. exchangeAuthorization 在向策略或回调登记提供身份前校验认证、模式、绑定和证明。
func (h *ExchangeService) exchangeAuthorization(ctx context.Context, values url.Values, mode Mode) (AuthorizationRequest, error) {
	params := h.options.Params
	request := AuthorizationRequest{Phase: AuthorizationExchange, ClientID: values.Get(params.Client), Mode: mode}
	if err := h.authenticateClient(ctx, values); err != nil {
		return request, err
	}
	client, err := h.server.getClient(ctx, request.ClientID)
	if err != nil {
		return request, err
	}
	if !h.server.isModeAllowed(client, mode) {
		return request, ErrModeUnsupported
	}
	if mode == ModeOAuth2 {
		code, err := h.server.getOAuth2Code(ctx, values.Get(params.Code))
		if err != nil {
			return request, err
		}
		if err := h.server.validateOAuth2CodeExchange(code, request.ClientID, values.Get(params.Redirect), values.Get(params.CodeVerifier), client.RequirePKCE); err != nil {
			return request, err
		}
		request.LoginID, request.Scopes = code.LoginID, slices.Clone(code.Scopes)
	} else {
		ticket, err := h.server.getTicket(ctx, values.Get(params.Ticket))
		if err != nil {
			return request, err
		}
		if err := h.server.validateTicketExchange(ticket, request.ClientID, values.Get(params.Redirect), values.Get(params.CodeVerifier), client.RequirePKCE); err != nil {
			return request, err
		}
		request.LoginID, request.Scopes = ticket.LoginID, slices.Clone(ticket.Scopes)
	}
	return request, nil
}

// signingKeys selects one complete trust scope and never merges client and fallback keys. signingKeys 选择完整的信任范围，不合并客户端与默认密钥组。
func (h *ExchangeService) signingKeys(client *Client) (string, string, error) {
	secret, verification := h.options.SecretKey, h.options.VerificationKey
	if client != nil && (client.SecretKey != "" || client.VerificationKey != "") {
		secret, verification = client.SecretKey, client.VerificationKey
	}
	if secret == "" {
		return "", "", ErrSignSecretRequired
	}
	return secret, verification, nil
}
