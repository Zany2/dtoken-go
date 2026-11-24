package acceptance

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
)

// acceptanceLogoutComplete requires business completion and delivery to each expected client. acceptanceLogoutComplete 要求业务完成且每个预期客户端均已收到通知。
func acceptanceLogoutComplete(response *acceptanceResponse, expected []string) error {
	if response == nil || response.status != http.StatusOK {
		return errors.New("global logout did not return HTTP 200")
	}
	var result struct {
		Code *int `json:"code"`
		Data struct {
			Result        string `json:"result"`
			Stage         string `json:"stage"`
			Notifications struct {
				Complete bool `json:"complete"`
				Clients  []struct {
					ClientID string `json:"clientId"`
					Status   string `json:"status"`
				} `json:"clients"`
			} `json:"notifications"`
		} `json:"data"`
	}
	if json.Unmarshal(response.body, &result) != nil || result.Code == nil || *result.Code != 0 || result.Data.Result != "ok" || result.Data.Stage != "complete" || !result.Data.Notifications.Complete {
		return errors.New("global logout did not acknowledge business success")
	}
	delivered := make(map[string]bool)
	for _, client := range result.Data.Notifications.Clients {
		if client.ClientID == "" || delivered[client.ClientID] || client.Status != "delivered" {
			return errors.New("global logout callback report contains missing identities, duplicates or incomplete deliveries")
		}
		delivered[client.ClientID] = true
	}
	for _, id := range expected {
		if id == "" || !delivered[id] {
			return errors.New("global logout did not confirm every expected application")
		}
	}
	return nil
}

// cleanupLogout requires a fresh signed redirect and an explicit business acknowledgement. cleanupLogout 要求新签名跳转及明确的业务成功确认。
func (c *acceptanceClient) cleanupLogout(app, center string, expected []string) error {
	response, err := c.request(http.MethodPost, app+"/logout-all", url.Values{}, app, nil)
	if err != nil {
		return err
	}
	target, err := c.redirect(app+"/logout-all", response, center, "/sso/logout")
	if err != nil {
		return err
	}
	response, err = c.request(http.MethodGet, target, nil, "", nil)
	if err != nil {
		return err
	}
	return acceptanceLogoutComplete(response, expected)
}
