package main

import (
	"encoding/json"
	"os"
)

const seenFile = "seen.json"

func loadSeen() map[string]bool {
	data, err := os.ReadFile(seenFile)
	if err != nil {
		return make(map[string]bool)
	}
	var seen map[string]bool
	if err := json.Unmarshal(data, &seen); err != nil {
		return make(map[string]bool)
	}
	return seen
}

func saveSeen(seen map[string]bool) {
	data, _ := json.Marshal(seen)
	os.WriteFile(seenFile, data, 0644)
}
