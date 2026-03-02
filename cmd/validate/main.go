package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Entry struct {
	ID        string   `json:"id"`
	Character string   `json:"character"`
	Anime     string   `json:"anime"`
	Team      string   `json:"team"`
	Sport     string   `json:"sport"`
	Year      int      `json:"year"`
	Image     string   `json:"image"`
	Tags      []string `json:"tags"`
}

func main() {
	imageSet := make(map[string]bool)
	usedImages := make(map[string]bool)
	ids := make(map[string]bool)
	var entries []Entry

	// Collect images
	filepath.WalkDir("images", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			ext := strings.ToLower(filepath.Ext(path))
			if ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".webp" {
				imageSet[filepath.ToSlash(path)] = true
			}
		}
		return nil
	})

	// Collect entries
	err := filepath.WalkDir("entries", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if filepath.Ext(path) == ".json" {
			file, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			var entry Entry
			if err := json.Unmarshal(file, &entry); err != nil {
				return fmt.Errorf("invalid JSON in %s: %w", path, err)
			}

			// Basic schema validation
			if entry.ID == "" ||
				entry.Character == "" ||
				entry.Anime == "" ||
				entry.Team == "" ||
				entry.Sport == "" ||
				entry.Year == 0 ||
				entry.Image == "" {
				return fmt.Errorf("missing required fields in %s", path)
			}

			if ids[entry.ID] {
				return fmt.Errorf("duplicate ID: %s", entry.ID)
			}
			ids[entry.ID] = true

			imagePath := filepath.ToSlash(entry.Image)

			if !strings.HasPrefix(imagePath, "images/") {
				return fmt.Errorf("image must be inside images/: %s", entry.ID)
			}

			if !imageSet[imagePath] {
				return fmt.Errorf("image not found for entry %s: %s", entry.ID, entry.Image)
			}

			usedImages[imagePath] = true
			entries = append(entries, entry)
		}
		return nil
	})

	if err != nil {
		fmt.Println("Validation error:", err)
		os.Exit(1)
	}

	// Check orphan images
	for img := range imageSet {
		if !usedImages[img] {
			fmt.Println("Orphan image:", img)
			os.Exit(1)
		}
	}

	// Generate index
	index := map[string]interface{}{
		"generatedAt": time.Now().UTC().Format(time.RFC3339),
		"total":       len(entries),
		"entries":     entries,
	}

	os.MkdirAll("generated", os.ModePerm)
	file, _ := os.Create("generated/index.json")
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	encoder.Encode(index)

	fmt.Println("Validation passed. Index generated.")
}
