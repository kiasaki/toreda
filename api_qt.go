package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"sync"
)

const (
	filePathQTCredentials = "data/qt_credentials.json"
)

type APIRequest struct {
	Method string
	Path   string
	Data   map[string]interface{}
}

type QTApi struct {
	Credentials *QTApiCredentials
	m           sync.Mutex
}

type QTApiCredentials struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ApiServer    string `json:"api_server"`
}

func NewQTApi() *QTApi {
	qt := QTApi{}
	c, err := loadQTApiCreds()
	if err != nil {
		panic(err)
	}
	qt.Credentials = c
	return &qt
}

func loadQTApiCreds() (*QTApiCredentials, error) {
	f, err := os.Open(filePathQTCredentials)
	if err != nil {
		return nil, err
	}

	c := QTApiCredentials{}
	err = json.NewDecoder(f).Decode(&c)
	if err != nil {
		return nil, err
	}

	return &c, nil
}

func saveQTApiCreds(c *QTApiCredentials) error {
	f, err := os.OpenFile(filePathQTCredentials, os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}

	return json.NewEncoder(f).Encode(c)
}

func (api *QTApi) RefreshCredentials() error {
	resp, err := http.PostForm("https://login.questrade.com/oauth2/token", url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {api.Credentials.RefreshToken},
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		resDump, err := httputil.DumpResponse(resp, true)
		if err != nil {
			return err
		}
		return errors.New(fmt.Sprintf(
			"Got non 200 status code when refreshing token: %d \n\n%s\n",
			resp.StatusCode, resDump,
		))
	}

	err = json.NewDecoder(resp.Body).Decode(&api.Credentials)
	if err != nil {
		return err
	}

	return saveQTApiCreds(api.Credentials)
}

func (api *QTApi) Request(r APIRequest, responseData interface{}) error {
	api.m.Lock()
	defer api.m.Unlock()

	var bodyReader io.Reader
	if len(r.Data) > 0 {
		body, err := json.Marshal(r.Data)
		if err != nil {
			return err
		}
		fmt.Println(string(body))
		bodyReader = bytes.NewReader(body)
	}

	if len(r.Method) == 0 {
		r.Method = http.MethodGet
	}

	if len(r.Path) == 0 {
		return errors.New("API Request path is required")
	}

	req, err := http.NewRequest(r.Method, api.Credentials.ApiServer+r.Path, bodyReader)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+api.Credentials.AccessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode == http.StatusUnauthorized {
		// Try refreshing OAuth token
		log.Println("Trying to refresh QT OAuth credentials")
		err := api.RefreshCredentials()
		if err != nil {
			log.Println("Failed to refresh QT OAuth credentials")
			return err
		}
		log.Println("Refreshed QT OAuth credentials")

		// Retry
		req.Header.Set("Authorization", "Bearer "+api.Credentials.AccessToken)
		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		body, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
	}

	if resp.StatusCode != http.StatusOK {
		return errors.New(fmt.Sprintf(
			"Got non 200 status code from Questrade API: %d\n\n%s\n\n%s",
			resp.StatusCode, string(body),
		))
	}

	// Debug
	fmt.Println(r.Path, string(body))
	return json.NewDecoder(bytes.NewReader(body)).Decode(responseData)
}
