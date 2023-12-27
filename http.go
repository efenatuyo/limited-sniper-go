package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
)

func get_x_token(client *fasthttp.Client, cookie string, sniper *Sniper) {
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)
	req.SetRequestURI("https://accountsettings.roblox.com/v1/email")
	req.Header.SetCookie(".ROBLOSECURITY", cookie)
	req.Header.SetMethod("POST")
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)
	client.Do(req, resp)
	csrfToken := string(resp.Header.Peek("x-csrf-token"))
	if csrfToken == "" {
		panic("Failed to get the x-csrf-token please check your cookie")
	} else {
		sniper.x_token = csrfToken
	}
}

func get_user_id(client *fasthttp.Client, cookie string) json.Number {
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)
	req.SetRequestURI("https://users.roblox.com/v1/users/authenticated")
	req.Header.SetCookie(".ROBLOSECURITY", cookie)
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	if err := client.Do(req, resp); err != nil {
		panic(fmt.Sprintf("failed to make GET request: %v", err))
	}

	var data map[string]interface{}
	decoder := json.NewDecoder(strings.NewReader(string(resp.Body())))
	decoder.UseNumber()

	if err := decoder.Decode(&data); err != nil {
		panic(fmt.Sprintf("failed to unmarshal JSON: %v", err))
	}

	userID, ok := data["id"].(json.Number)
	if !ok {
		panic(fmt.Sprintf("couldn't scrape user ID. Response: %v", data))
	}
	return userID
}

func get_v1(items []string, sniper *Sniper, req *fasthttp.Request, cookie string) (interface{}, error) {
	req.SetRequestURI("https://catalog.roblox.com/v1/catalog/items/details")
	req.Header.SetContentType("application/json")
	req.Header.SetMethod("POST")

	requestBody, err := json.Marshal(map[string]interface{}{
		"items": func() []map[string]interface{} {
			var itemList []map[string]interface{}
			for _, item := range items {
				itemList = append(itemList, map[string]interface{}{
					"itemType": "Asset",
					"id":       item,
				})
			}
			return itemList
		}(),
	})
	if err != nil {
		return nil, fmt.Errorf("error encoding request body: %v", err)
	}
	req.Header.Set("x-csrf-token", sniper.x_token)
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	req.SetBody(requestBody)
	if err := fasthttp.Do(req, resp); err != nil {
		return nil, fmt.Errorf("error making HTTP request: %v", err)
	}

	switch resp.StatusCode() {
	case fasthttp.StatusForbidden:
		errResp := make(map[string]interface{})
		if err := json.Unmarshal(resp.Body(), &errResp); err != nil {
			return nil, fmt.Errorf("error parsing response body: %v", err)
		}
		if errMsg, ok := errResp["message"].(string); ok && errMsg == "Token Validation Failed" {
			get_x_token(&fasthttp.Client{}, cookie, sniper)
			return nil, fmt.Errorf("generated new x_token")
		}
	case fasthttp.StatusTooManyRequests:
		return nil, fmt.Errorf("rate limit exceeded")
	}

	decoder := json.NewDecoder(strings.NewReader(string(resp.Body())))
	decoder.UseNumber()

	data := make(map[string]interface{})
	if err := decoder.Decode(&data); err != nil {
		return nil, fmt.Errorf("error parsing response body: %v", err)
	}

	return data["data"], nil
}

func get_data_item(req *fasthttp.Request, collectibleItemID string, sniper *Sniper, cookie string) (map[string]interface{}, error) {
	url := fmt.Sprintf("https://apis.roblox.com/marketplace-sales/v1/item/%s/resellers?limit=1", collectibleItemID)

	req.SetRequestURI(url)
	req.Header.Set("x-csrf-token", sniper.x_token)
	req.Header.Set("Accept-Encoding", "gzip, deflate")
	req.Header.SetMethod("GET")

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	if err := fasthttp.Do(req, resp); err != nil {
		return nil, fmt.Errorf("error making HTTP request: %v", err)
	}

	switch resp.StatusCode() {
	case fasthttp.StatusForbidden:
		errResp := make(map[string]interface{})
		decoder := json.NewDecoder(strings.NewReader(string(resp.Body())))
		decoder.UseNumber()
		if err := decoder.Decode(&errResp); err != nil {
			return nil, fmt.Errorf("error parsing response body: %v", err)
		}
		if errMsg, ok := errResp["message"].(string); ok && errMsg == "Token Validation Failed" {
			get_x_token(&fasthttp.Client{}, cookie, sniper)
			return nil, fmt.Errorf("generated new x-csrf-token")
		}
	case fasthttp.StatusTooManyRequests:
		return nil, fmt.Errorf("rate limit exceeded")
	}

	if len(resp.Body()) == 0 {
		return nil, fmt.Errorf("empty JSON response")
	}

	var data map[string]interface{}
	decoder := json.NewDecoder(strings.NewReader(string(resp.Body())))
	decoder.UseNumber()
	if err := decoder.Decode(&data); err != nil {
		return nil, fmt.Errorf("error parsing response body: %v", err)
	}

	return data["data"].([]interface{})[0].(map[string]interface{}), nil
}

func purchase(item item_data, req *fasthttp.Request, sniper *Sniper) error {
	data := map[string]interface{}{
		"collectibleItemId":         item.collectible_item_id,
		"expectedCurrency":          1,
		"expectedPrice":             item.price,
		"expectedPurchaserId":       sniper.user_id,
		"expectedPurchaserType":     "User",
		"expectedSellerId":          item.creator,
		"expectedSellerType":        "User",
		"idempotencyKey":            uuid.New().String(),
		"collectibleProductId":      item.productid_data,
		"collectibleItemInstanceId": item.collectible_item_instance_id,
	}

	url := fmt.Sprintf("https://apis.roblox.com/marketplace-sales/v1/item/%s/purchase-resale", item.collectible_item_id)

	req.SetRequestURI(url)
	req.Header.Set("x-csrf-token", sniper.x_token)
	req.Header.SetContentType("application/json")
	req.Header.SetMethod("POST")

	body, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("error encoding request body: %v", err)
	}

	req.SetBody(body)

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	if err := fasthttp.Do(req, resp); err != nil {
		return fmt.Errorf("error making HTTP request: %v", err)
	}
	if resp.StatusCode() == fasthttp.StatusOK {
		var jsonResp map[string]interface{}
		if err := json.NewDecoder(strings.NewReader(string(resp.Body()))).Decode(&jsonResp); err != nil {
			return fmt.Errorf("error parsing response body: %v", err)
		}

		if !jsonResp["purchased"].(bool) {
			return fmt.Errorf("Failed to buy item %d, reason: %v", item.id, jsonResp["errorMessage"])
		}

		sniper.buy_logs = append(sniper.buy_logs, fmt.Sprintf("[%s] Bought item %d for a price of %d", time.Now().Format("15:04:05"), item.id, item.price))
	} else {
		return fmt.Errorf("Failed to buy item %d", item.id)
	}

	return nil
}
