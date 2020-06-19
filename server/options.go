package server

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"os"
)

const filePath string = "./config/"

func init() {
	err := os.Mkdir(filePath, 0644)
	if err != nil && !os.IsExist(err) {
		panic(err)
	}
}

func LoadOptions(v interface{}, name string) interface{} {
	path := filePath + name + ".json"
	options := loadFile(path, v)
	if options != nil {
		return options
	}
	writeFile(path, v)
	return v
}

func loadFile(path string, t interface{}) interface{} {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil
	}
	jsonErr := json.Unmarshal(data, t)
	if jsonErr != nil {
		panic(jsonErr)
	}
	return t
}

func writeFile(path string, t interface{}) {
	data, err := json.Marshal(t)
	if err != nil {
		panic(err)
	}
	var dist bytes.Buffer
	indentErr := json.Indent(&dist, data, "", "    ")
	if indentErr != nil {
		panic(indentErr)
	}
	fileErr := ioutil.WriteFile(path, dist.Bytes(), 0644)
	if fileErr != nil {
		panic(fileErr)
	}
}
