package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"

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

func getAppDir() (string, error) {
	dir, err := os.UserConfigDir()

	if err != nil {
		return "", err
	}

	path := filepath.Join(dir, "wordef")

	err = os.MkdirAll(path, os.ModePerm)

	if err != nil {
		return "", err
	}

	return path, nil
}

func saveToAppDir(word string, rawJson []byte, appDir string) error {
	wordPath := path.Join(appDir, word + ".json")

	_, err := os.Stat(wordPath)

	if err == nil {
		return errors.New("Word already saved to file")
	}

	return os.WriteFile(wordPath, rawJson, os.ModePerm)
}

func searchWordLocal(word, appDir string) (parsed []WordInfo, rawJson []byte, err error) {
	wordPath := path.Join(appDir, word + ".json")

	_, err = os.Stat(wordPath)

	if err != nil {
		return nil, nil, err
	}

	rawJson, err = os.ReadFile(wordPath)

	if err != nil {
		return nil, nil, err
	}

	err = json.Unmarshal(rawJson, &parsed)

	if err != nil {
		return nil, nil, err
	}

	return parsed, rawJson, nil
}

func searchWord(word, appDir string) (parsed []WordInfo, err error) {

	local, rawJson, err := searchWordLocal(word, appDir)

	if err == nil {
		return local, nil
	}

	url := "https://api.dictionaryapi.dev/api/v2/entries/en/" + word

	fmt.Println(url)

	resp, err := http.Get(url)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	rawJson, err = io.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(rawJson, &parsed)

	if err != nil {
		return nil, err
	}

	saveToAppDir(word, rawJson, appDir)

	return parsed, nil
}

func renderDefinitionsTable(wordInfo WordInfo) {
	table := tablewriter.NewWriter(os.Stdout)

	table.SetHeader([]string { "POS", "Definition" })

	for _, v := range wordInfo.Meanings {
		pos := v.PartOfSpeech
		definition := v.Definitions[0].Definition

		table.Append([]string {pos, definition})
	}

	table.Render()
}

func main() {
	appDir, err := getAppDir()

	if err != nil {
		log.Fatalln(err)
	}

	args := os.Args

	if len(args) < 2 {
		log.Fatalln("Must pass word as argument")
	}

	word := args[1]

	var resp []WordInfo

	resp, err = searchWord(word, appDir)

	wordInfo := resp[0]

	if len(wordInfo.Meanings) == 0 {
		fmt.Println("No definitions found for word", word)
		return
	}

	renderDefinitionsTable(wordInfo)
}