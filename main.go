package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

type Config struct {
	Cookie string `json:"cookie"`
	Items  struct {
		GlobalMaxPrice int `json:"global_max_price"`
		List           map[string]struct {
			MaxPrice int `json:"max_price"`
		} `json:"list"`
	} `json:"items"`
}

func main() {
	file, err := os.Open("config.json")
	if err != nil {
		panic(fmt.Sprintln("Error opening config.json:", err))
	}

	fileData, err := io.ReadAll(file)
	if err != nil {
		panic(fmt.Sprintln("Error reading config.json:", err))
	}

	file.Close()
	var config Config
	err = json.Unmarshal(fileData, &config)
	if err != nil {
		panic(fmt.Sprintln("Error unmarshalling JSON:", err))
	}
	setup(config)
}
