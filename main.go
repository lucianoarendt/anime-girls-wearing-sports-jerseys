package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
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

	// ==============================
	// Collect images (relative paths normalized)
	// ==============================
	err := filepath.WalkDir("images", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() {
			ext := strings.ToLower(filepath.Ext(path))
			if ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".webp" {
				cleanPath := filepath.ToSlash(filepath.Clean(path))
				imageSet[cleanPath] = true
			}
		}

		return nil
	})
	if err != nil {
		fmt.Println("Error walking images:", err)
		os.Exit(1)
	}

	// ==============================
	// Collect entries
	// ==============================
	err = filepath.WalkDir("entries", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if filepath.Ext(path) != ".json" {
			return nil
		}

		file, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		var entry Entry
		if err := json.Unmarshal(file, &entry); err != nil {
			return fmt.Errorf("invalid JSON in %s: %w", path, err)
		}

		// ==============================
		// Basic schema validation
		// ==============================
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

		// ==============================
		// Image validation (cross-platform safe)
		// ==============================
		imagePath := filepath.ToSlash(filepath.Clean(entry.Image))

		if strings.Contains(imagePath, "..") {
			return fmt.Errorf("image path cannot contain '..': %s", entry.ID)
		}

		if !strings.HasPrefix(imagePath, "images/") {
			return fmt.Errorf("image must be inside images/: %s", entry.ID)
		}

		if !imageSet[imagePath] {
			return fmt.Errorf(
				"image not found or path mismatch for entry %s: %s",
				entry.ID,
				entry.Image,
			)
		}

		usedImages[imagePath] = true
		entries = append(entries, entry)

		return nil
	})

	if err != nil {
		fmt.Println("Validation error:", err)
		os.Exit(1)
	}

	// ==============================
	// Check orphan images
	// ==============================
	for img := range imageSet {
		if !usedImages[img] {
			fmt.Println("Orphan image:", img)
			os.Exit(1)
		}
	}

	// ==============================
	// Deterministic ordering (important for CI stability)
	// ==============================
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].ID < entries[j].ID
	})

	// ==============================
	// Generate index
	// ==============================
	index := map[string]interface{}{
		"generatedAt": time.Now().UTC().Format(time.RFC3339),
		"total":       len(entries),
		"entries":     entries,
	}

	file, err := os.Create("index.json")
	if err != nil {
		fmt.Println("Error creating index file:", err)
		os.Exit(1)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(index); err != nil {
		fmt.Println("Error encoding index:", err)
		os.Exit(1)
	}

	fmt.Println("Validation passed. Index generated successfully.")
}
