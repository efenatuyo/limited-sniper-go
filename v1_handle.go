package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/valyala/fasthttp"
)

type item_data struct {
	price                        int64
	creator                      int64
	id                           int64
	productid_data               string
	collectible_item_id          string
	collectible_item_instance_id string
}

func searcher_v1(items []string, sniper *Sniper, req *fasthttp.Request, config *Config, cookie string) {
	defer time.Sleep(1 * time.Second)
	defer sniper.Summary()

	item_s, err := get_v1(items, sniper, req, cookie)
	if err != nil {
		sniper.error_logs.Append(fmt.Sprintf("[%s] %v", time.Now().Format("15:04:05"), err))
		return
	}
	if itemList, ok := item_s.([]interface{}); ok {
		sniper.total_searchers += len(itemList)
		for _, item := range itemList {
			switch item := item.(type) {
			case map[string]interface{}:
				if status := item["priceStatus"]; status != "" {
					if status == "Off Sale" {
						if id, ok := item["id"].(json.Number); ok {
							delete(config.Items.List, id.String())
						}
						continue
					}
				} else {
					continue
				}
				price, _ := item["lowestResalePrice"].(json.Number).Int64()
				status := item["hasResellers"].(bool)
				if !status || price > int64(config.Items.GlobalMaxPrice) || price > int64(config.Items.List[item["id"].(json.Number).String()].MaxPrice) {
					continue
				}
				data_item := item_data{price: price, collectible_item_id: item["collectibleItemId"].(string)}
				data, err := get_data_item(req, data_item.collectible_item_id, sniper, cookie)
				if err == nil {
					data_item.collectible_item_instance_id = data["collectibleItemInstanceId"].(string)
					data_item.productid_data = data["collectibleProductId"].(string)
					seller_id, _ := data["seller"].(map[string]interface{})["sellerId"].(json.Number).Int64()
					data_item.creator = seller_id
					item_id, _ := item["id"].(json.Number).Int64()
					data_item.id = item_id
					if err := purchase(data_item, req, sniper); err != nil {
						sniper.error_logs.Append(fmt.Sprintf("[%s] %v", time.Now().Format("15:04:05"), err))
					}
				} else {
					sniper.error_logs.Append(fmt.Sprintf("[%s] %v", time.Now().Format("15:04:05"), err))
					continue
				}
			}
		}
		sniper.search_logs.Append(fmt.Sprintf("[%s] Searched %d", time.Now().Format("15:04:05"), len(itemList)))
	}
}

func main_v1(config Config, sniper Sniper) {
	req := fasthttp.AcquireRequest()
	req.SetRequestURI("https://catalog.roblox.com/v1/catalog/items/details")
	req.Header.SetCookie(".ROBLOSECURITY", config.Cookie)
	req.Header.Add("Accept-Encoding", "gzip, deflate")
	for {
		for _, items := range split_list(config.Items.List) {
			if len(items) > 0 {
				searcher_v1(items, &sniper, req, &config, config.Cookie)
			}
		}
	}

}
