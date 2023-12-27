package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/valyala/fasthttp"
)

type Sniper struct {
	error_logs      LimitedQueue
	buy_logs        []string
	search_logs     LimitedQueue
	total_searchers int
	average_speed   LimitedQueue
	x_token         string
	user_id         json.Number
}

func (s *Sniper) Summary() {
	summary := fmt.Sprintf("Total Searches: %d\n\n\nSearch Logs:\n%s\n\nBuy Logs:\n%s\n\n\nTotal Items bought: %d\n\n\nError Logs:\n%s",
		s.total_searchers,
		strings.Join(s.search_logs.ToStringSlice(), "\n"),
		strings.Join(s.buy_logs, "\n"),
		len(s.buy_logs),
		strings.Join(s.error_logs.ToStringSlice(), "\n"),
	)
	clearConsole()
	fmt.Println(summary)
}

func clearConsole() {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "cls")
	case "linux", "darwin":
		cmd = exec.Command("clear")
	default:
		fmt.Println("Unsupported operating system")
		return
	}

	cmd.Stdout = os.Stdout
	cmd.Run()
}

func split_list(inputMap map[string]struct {
	MaxPrice int "json:\"max_price\""
}) [][]string {
	maxLen := 120
	keys := make([]string, 0, len(inputMap))
	for key := range inputMap {
		keys = append(keys, key)
	}

	if len(keys) <= maxLen {
		return [][]string{keys}
	}

	numChunks := -(-maxLen / len(keys))
	repeatedKeys := make([]string, 0, len(keys)*numChunks)
	for i := 0; i < numChunks; i++ {
		repeatedKeys = append(repeatedKeys, keys...)
	}

	result := make([][]string, 0, numChunks)
	for i := 0; i < len(repeatedKeys); i += maxLen {
		end := i + maxLen
		if end > len(repeatedKeys) {
			end = len(repeatedKeys)
		}
		result = append(result, repeatedKeys[i:end])
	}

	return result
}

func setup(config Config) {
	var wg sync.WaitGroup
	for id := range config.Items.List {
		_, err := strconv.Atoi(id)
		if err != nil {
			panic(fmt.Sprintln("Item id is not a number: ", id))
		}
	}
	sniper := Sniper{
		LimitedQueue{MaxLen: 5, Items: make([]interface{}, 0, 5)},
		[]string{},
		LimitedQueue{MaxLen: 5, Items: make([]interface{}, 0, 5)},
		0,
		LimitedQueue{MaxLen: 20, Items: make([]interface{}, 0, 20)},
		"",
		"",
	}
	client := &fasthttp.Client{}
	get_x_token(client, config.Cookie, &sniper)
	sniper.user_id = get_user_id(client, config.Cookie)
	wg.Add(1)
	go func() {
		main_v1(config, sniper)
	}()
	wg.Wait()
}
