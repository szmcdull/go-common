package common

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/tidwall/gjson"
)

var (
	client = http.DefaultClient
)

func HttpSendRequest(req *http.Request) (statusCode int, statusText string, body string, err error) {
	resp, err := client.Do(req)
	if err != nil {
		return 0, ``, ``, NewError(`req.Do`, err)
	}
	defer resp.Body.Close()
	bts, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, resp.Status, ``, NewError(`ioutil.ReadAll`, err)
	}
	return resp.StatusCode, resp.Status, string(bts), nil
}

func HttpGet(url string) (statusCode int, statusText string, body string, err error) {
	req, err := http.NewRequest(`GET`, url, nil)
	if err != nil {
		return 0, ``, ``, NewError(`http.NewRequest`, err)
	}
	return HttpSendRequest(req)
}

func HttpPostRaw(url string, content string, headers map[string]string) (statusCode int, statusText string, body string, err error) {
	req, err := http.NewRequest(`POST`, url, strings.NewReader(content))
	if err != nil {
		return 0, ``, ``, NewError(`http.NewRequest`, err)
	}

	req.Header.Set(`Content-Length`, strconv.Itoa(len(content)))
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	return HttpSendRequest(req)
}

func HttpPost(uri string, content map[string]string, headers map[string]string) (statusCode int, statusText string, body string, err error) {
	values := url.Values{}
	for k, v := range content {
		values.Set(k, v)
	}

	contentStr := values.Encode()
	req, err := http.NewRequest(`POST`, uri, strings.NewReader(contentStr))
	if err != nil {
		return 0, ``, ``, NewError(`http.NewRequest`, err)
	}

	req.Header.Set(`Content-Type`, `application/x-www-form-urlencoded`)
	req.Header.Set(`Content-Length`, strconv.Itoa(len(contentStr)))
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	return HttpSendRequest(req)
}

func HttpPostJson(uri string, content interface{}, headers map[string]string) (statusCode int, statusText string, body string, err error) {
	contentStr, err := json.Marshal(content)
	if err != nil {
		return 0, ``, ``, NewError(`json.Marshal`, err)
	}

	req, err := http.NewRequest(`POST`, uri, bytes.NewBuffer(contentStr))
	if err != nil {
		return 0, ``, ``, NewError(`http.NewRequest`, err)
	}

	req.Header.Set(`Content-Type`, `application/json`)
	req.Header.Set(`Content-Length`, strconv.Itoa(len(contentStr)))
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	return HttpSendRequest(req)
}

func HttpResponse(statusCode int, statusText string, body string, err error) (error, string) {
	if err != nil {
		return err, ``
	}
	if statusCode != 200 {
		return NewError(fmt.Sprintf(`HTTP %d (%s): %s`, statusCode, statusText, body), nil), body
	}
	return nil, body
}

func JsonResponse(statusCode int, statusText string, body string, err error) (error, *gjson.Result) {
	var result gjson.Result
	if err != nil {
		return err, nil
	}
	result = gjson.Parse(body)
	if statusCode != 200 {
		return NewError(fmt.Sprintf(`HTTP %d (%s): %s`, statusCode, statusText, body), nil), &result
	}
	return nil, &result
}
