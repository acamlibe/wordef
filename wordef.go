package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/olekukonko/tablewriter"
)

type WordInfo struct {
	Word      string `json:"word"`
	Phonetic  string `json:"phonetic"`
	Phonetics []struct {
		Text  string `json:"text"`
		Audio string `json:"audio,omitempty"`
	} `json:"phonetics"`
	Origin   string `json:"origin"`
	Meanings []struct {
		PartOfSpeech string `json:"partOfSpeech"`
		Definitions  []struct {
			Definition string `json:"definition"`
			Example    string `json:"example"`
			Synonyms   []any  `json:"synonyms"`
			Antonyms   []any  `json:"antonyms"`
		} `json:"definitions"`
	} `json:"meanings"`
}

func getCacheDir() (string, error) {
	dir, err := os.UserConfigDir()

	if err != nil {
		return "", fmt.Errorf("Failed to get user config directory: %w", err)
	}

	path := filepath.Join(dir, "wordef")

	err = os.MkdirAll(path, os.ModePerm)

	if err != nil {
		return "", fmt.Errorf("Failed to create app directory: %w", err)
	}

	return path, nil
}

func saveToCache(word string, rawJson []byte, cacheDir string) error {
	wordPath := path.Join(cacheDir, word+".json")

	_, err := os.Stat(wordPath)

	if err == nil {
		return errors.New("Word already saved to file")
	}

	err = os.WriteFile(wordPath, rawJson, os.ModePerm)

	if err != nil {
		return fmt.Errorf("Failed to write cache file to app directory: %w", err)
	}

	return nil
}

func fetchFromCache(word, cacheDir string) (rawJson []byte, err error) {
	wordPath := path.Join(cacheDir, word+".json")

	_, err = os.Stat(wordPath)

	if err != nil {
		return nil, fmt.Errorf("Word not found in cache: %w", err)
	}

	rawJson, err = os.ReadFile(wordPath)

	if err != nil {
		return nil, fmt.Errorf("failed to read cache file: %w", err)
	}

	return rawJson, nil
}

func fetchFromApi(word string) (rawJson []byte, err error) {
	url := "https://api.dictionaryapi.dev/api/v2/entries/en/" + word

	resp, err := http.Get(url)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	rawJson, err = io.ReadAll(resp.Body)

	if err != nil {
		return nil, fmt.Errorf("Failed to read response body: %w", err)
	}

	return rawJson, nil
}

func searchWord(word, cacheDir string) (parsed []WordInfo, err error) {

	rawJson, err := fetchFromCache(word, cacheDir)

	if err != nil {
		rawJson, err = fetchFromApi(word)
	}

	err = json.Unmarshal(rawJson, &parsed)

	if err != nil {
		return nil, err
	}

	saveToCache(word, rawJson, cacheDir)

	return parsed, nil
}

func getCachedWords(cacheDir string) (words []string, err error) {
	err = filepath.WalkDir(cacheDir, func(s string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		filePath := d.Name()

		if filepath.Ext(filePath) == ".json" {
			fileName := filepath.Base(filePath)
			fileNameNoExt := strings.Replace(fileName, ".json", "", 1)

			words = append(words, fileNameNoExt)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("Failed to get cached words from cache directory: %w", err)
	}

	return words, nil
}

func capitalizeString(s string) string {
	if len(s) == 0 {
		return ""
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

func renderDefinitionsTable(table *tablewriter.Table, wordInfo WordInfo) {
	table.SetHeader([]string{"POS", "Definition"})

	for _, v := range wordInfo.Meanings {
		pos := v.PartOfSpeech
		definition := v.Definitions[0].Definition

		table.Append([]string{pos, definition})
	}

	table.Render()
}

func renderCachedWordsTable(table *tablewriter.Table, cachedWords []string) {
	table.SetHeader([]string{"Saved Words"})

	for _, v := range cachedWords {
		table.Append([]string{v})
	}

	table.Render()
}

func handleSearchCommand(table *tablewriter.Table, word string, cacheDir string) error {
	var resp []WordInfo

	resp, err := searchWord(word, cacheDir)

	if err != nil {
		return fmt.Errorf("Failed to search for word %s: %w", word, err)
	}

	wordInfo := resp[0]

	if len(wordInfo.Meanings) == 0 {
		return fmt.Errorf("Failed to search for word %s: %w", word, err)
	}

	fmt.Println("Word:", wordInfo.Word)
	fmt.Println("Phonetic Spelling:", wordInfo.Phonetic)
	fmt.Println()

	renderDefinitionsTable(table, wordInfo)

	return nil
}

func handleWelcomeCommand(table *tablewriter.Table, cacheDir string) error {
	fmt.Println("wordef is used to lookup the phonetic spelling and the different definitions of a word, depending on the part-of-speech (noun, verb, adjective).")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("\twordef - shows this welcome message and shows a list of words searched and saved locally")
	fmt.Println("\twordef {word} - displays a word's phonetic spelling and definitions. Searches either through a local cache or through an API")
	fmt.Println()
	fmt.Println("Cache Directory:", cacheDir)

	cachedWords, err := getCachedWords(cacheDir)

	if err != nil {
		return fmt.Errorf("Failed to get list of cached words")
	}

	renderCachedWordsTable(table, cachedWords)

	return nil
}

func main() {
	cacheDir, err := getCacheDir()

	if err != nil {
		log.Fatalln(err)
	}

	args := os.Args

	table := tablewriter.NewWriter(os.Stdout)

	if len(args) == 2 {
		word := capitalizeString(args[1])
		handleSearchCommand(table, word, cacheDir)
	} else {
		handleWelcomeCommand(table, cacheDir)
	}
}
