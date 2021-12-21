package main

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const dateFormat = "2006-01-02"
const dateTimeFormat = "2006-01-02 15:04:05"

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func fmin(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func fmax(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func intArrayToString(ints []int) string {
	strs := make([]string, len(ints))
	for i, value := range ints {
		strs[i] = strconv.Itoa(value)
	}

	return strings.Join(strs, ",")
}

func mustParseFloat(v string) float64 {
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		panic(err)
	}
	return f
}

func writeJsonFile(fileName string, data interface{}) error {
	if err := os.MkdirAll(filepath.Dir(fileName), 0755); err != nil {
		return err
	}
	f, err := os.OpenFile(fileName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}

	return json.NewEncoder(f).Encode(data)
}

func readJsonFile(fileName string, data interface{}) error {
	f, err := os.Open(fileName)
	if err != nil {
		return err
	}

	return json.NewDecoder(f).Decode(data)
}

func fround(f float64) float64 {
	return math.Floor(f + .5)
}

func froundn(f float64, places int) float64 {
	shift := math.Pow(10, float64(places))
	return fround(f*shift) / shift
}
