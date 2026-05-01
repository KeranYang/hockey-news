package main

import (
	"encoding/json"
	"os"
)

const seenFile = "seen.json"
const pageHashFile = "page_hashes.json"

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

func loadPageHashes() map[string]string {
	data, err := os.ReadFile(pageHashFile)
	if err != nil {
		return make(map[string]string)
	}
	var hashes map[string]string
	if err := json.Unmarshal(data, &hashes); err != nil {
		return make(map[string]string)
	}
	return hashes
}

func savePageHashes(hashes map[string]string) {
	data, _ := json.Marshal(hashes)
	os.WriteFile(pageHashFile, data, 0644)
}
