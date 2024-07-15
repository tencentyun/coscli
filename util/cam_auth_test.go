package util

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"reflect"
	"testing"

	"bou.ke/monkey"
	"github.com/stretchr/testify/assert"
)

func TestCam_auth_true(t *testing.T) {
	roleName := "valid"
	var guard *monkey.PatchGuard
	var c *http.Client

	guard = monkey.PatchInstanceMethod(reflect.TypeOf(c), "Do", func(*http.Client, *http.Request) (*http.Response, error) {
		p := Data{
			TmpSecretId:  "ok",
			TmpSecretKey: "ok",
			ExpiredTime:  1,
			Expiration:   "ok",
			Token:        "ok",
			Code:         "Success",
		}
		body, _ := json.Marshal(p)
		res := &http.Response{
			Status:     "200 OK",
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       ioutil.NopCloser(bytes.NewReader(body)),
		}
		res.Header.Set("Content-Type", "application/json")
		return res, nil
	})
	defer guard.Unpatch()
	data := CamAuth(roleName)
	assert.Equal(t, "Success", data.Code, "they should be equal")
}
