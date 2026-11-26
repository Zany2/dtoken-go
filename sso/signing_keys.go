package sso

import "net/url"

// validateVerificationKey prevents an extra key from creating an implicit signing identity. validateVerificationKey 避免额外验签密钥隐式建立签名身份。
func validateVerificationKey(secret, verification, field string) error {
	if verification != "" && secret == "" {
		return configError(field, "requires an active SecretKey in the same configuration", ErrSignSecretRequired)
	}
	return nil
}

// signingKeys selects one complete trust scope and never merges client and fallback keys. signingKeys 选择完整的信任范围，不合并客户端与默认密钥组。
func (h *HTTPServer) signingKeys(client *Client) (string, string, error) {
	return h.exchangeCore(nil).signingKeys(client)
}

// verifyProtocolSignature accepts only the selected active key or its optional verification companion. verifyProtocolSignature 仅接受已选择的签名密钥及其可选验签密钥。
func verifyProtocolSignature(values url.Values, names ParamNames, secret, verification string) bool {
	if secret == "" {
		return false
	}
	if NewSignerWithParams(secret, names).Verify(values) {
		return true
	}
	return verification != "" && verification != secret && NewSignerWithParams(verification, names).Verify(values)
}
