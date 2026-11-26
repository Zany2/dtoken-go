package sso

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"net/url"
	"time"
)

var (
	// ErrPKCERequired indicates that the client requires a proof-bound one-time credential. ErrPKCERequired 表示客户端要求绑定兑换证明的一次性凭证。
	ErrPKCERequired = errors.New("PKCE is required")
	// ErrInvalidPKCE indicates malformed challenge parameters or an invalid verifier. ErrInvalidPKCE 表示挑战参数格式错误或兑换证明无效。
	ErrInvalidPKCE = errors.New("invalid PKCE proof")
)

// PKCEMethodS256 is the only supported challenge method. PKCEMethodS256 是唯一支持的挑战算法。
const PKCEMethodS256 = "S256"

// OneTimeCredentialOptions controls ticket/code lifetime and optional PKCE binding. OneTimeCredentialOptions 控制 Ticket/授权码有效期及可选 PKCE 绑定。
type OneTimeCredentialOptions struct {
	// Timeout uses the server default when nonpositive. Timeout 非正值时使用服务端默认有效期。
	Timeout time.Duration
	// CodeChallenge is the base64url SHA-256 digest of the verifier. CodeChallenge 是 verifier 的 SHA-256 摘要，以无填充 base64url 编码。
	CodeChallenge string
	// CodeChallengeMethod must be S256 when CodeChallenge is provided. CodeChallengeMethod 在提供 CodeChallenge 时必须为 S256。
	CodeChallengeMethod string
}

// NewPKCEVerifier generates a 256-bit random verifier for one login flow. NewPKCEVerifier 为单次登录流程生成 256 位随机兑换证明。
func NewPKCEVerifier() string {
	var random [32]byte
	_, _ = rand.Read(random[:])
	return base64.RawURLEncoding.EncodeToString(random[:])
}

// PKCEChallenge derives an S256 challenge from a 43–128 character RFC 7636 verifier. PKCEChallenge 从符合 RFC 7636 的 43–128 字符兑换证明计算 S256 挑战。
func PKCEChallenge(verifier string) (string, error) {
	if !validPKCEVerifier(verifier) {
		return "", ErrInvalidPKCE
	}
	digest := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(digest[:]), nil
}

// validPKCEVerifier permits only the RFC 7636 unreserved ASCII alphabet. validPKCEVerifier 仅允许 RFC 7636 规定的 ASCII 非保留字符。
func validPKCEVerifier(verifier string) bool {
	if len(verifier) < 43 || len(verifier) > 128 {
		return false
	}
	for _, ch := range verifier {
		if !((ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '-' || ch == '.' || ch == '_' || ch == '~') {
			return false
		}
	}
	return true
}

// validatePKCEChallenge rejects partial, downgraded, or noncanonical challenges. validatePKCEChallenge 拒绝不完整、降级或非规范编码的挑战。
func validatePKCEChallenge(challenge, method string, required bool) error {
	if challenge == "" && method == "" {
		if required {
			return ErrPKCERequired
		}
		return nil
	}
	if method != PKCEMethodS256 || len(challenge) != 43 {
		return ErrInvalidPKCE
	}
	decoded, err := base64.RawURLEncoding.Strict().DecodeString(challenge)
	if err != nil || len(decoded) != sha256.Size {
		return ErrInvalidPKCE
	}
	return nil
}

// verifyPKCE checks proof before consuming credentials and rejects unbound verifier submissions. verifyPKCE 在消费凭证前检查证明，并拒绝对未绑定凭证提交 verifier。
func verifyPKCE(challenge, method, verifier string, required bool) error {
	if err := validatePKCEChallenge(challenge, method, required); err != nil {
		return err
	}
	if challenge == "" {
		if verifier != "" {
			return ErrInvalidPKCE
		}
		return nil
	}
	actual, err := PKCEChallenge(verifier)
	if err != nil || subtle.ConstantTimeCompare([]byte(actual), []byte(challenge)) != 1 {
		return ErrInvalidPKCE
	}
	return nil
}

// validatePKCEParameters rejects duplicates and proof parameters on the wrong protocol leg. validatePKCEParameters 拒绝重复参数及出现在错误协议阶段的证明参数。
func validatePKCEParameters(values url.Values, params ParamNames, authorize bool) error {
	for _, key := range []string{params.CodeChallenge, params.CodeChallengeMethod, params.CodeVerifier} {
		if items, exists := values[key]; exists && (len(items) != 1 || items[0] == "") {
			return ErrInvalidPKCE
		}
	}
	if authorize {
		if _, exists := values[params.CodeVerifier]; exists {
			return ErrInvalidPKCE
		}
	} else if len(values[params.CodeChallenge]) != 0 || len(values[params.CodeChallengeMethod]) != 0 {
		return ErrInvalidPKCE
	}
	return nil
}
