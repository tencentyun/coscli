package util

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/mozillazg/go-httpheader"
	cosgo "github.com/tencentyun/cos-go-sdk-v5"
)

/**
1. 首先源地址 /ofs
2. 目标地址 https://tina-coscli-test-1253960454.cos.ap-chengdu.myqcloud.com/x?rename
3. header头部
*/
type ObjectMoveOptions struct {
	*cosgo.BucketHeadOptions
	*cosgo.ACLHeaderOptions
	XCosRenameSource string `header:"x-cos-rename-source" url:"-" xml:"-"`
}
type jsonError struct {
	Code      int    `json:"code,omitempty"`
	Message   string `json:"message,omitempty"`
	RequestID string `json:"request_id,omitempty"`
}

func PutRename(ctx context.Context, config *Config, param *Param, c *cosgo.Client, name, dstURL string,
	closeBody bool) (resp *http.Response, err error) {
	var cancel context.CancelFunc
	if closeBody {
		ctx, cancel = context.WithCancel(ctx)
		defer cancel()
	}
	durl := strings.SplitN(dstURL, "/", 2)
	if len(durl) < 2 {
		return nil, errors.New(fmt.Sprintf("x-cos-rename-source format error: #{dstURL}"))
	}
	var u string
	u = fmt.Sprintf("%s/%s?rename", c.BaseURL.BucketURL, durl[1])

	req, err := http.NewRequest("PUT", u, nil)
	if err != nil {
		return
	}
	copyOpt := &ObjectMoveOptions{
		&cosgo.BucketHeadOptions{},
		&cosgo.ACLHeaderOptions{},
		"/" + name,
	}

	//header
	req.Header, err = addHeaderOptions(req.Header, copyOpt)
	if err != nil {
		return
	}
	if v := req.Header.Get("Content-Length"); req.ContentLength == 0 && v != "" && v != "0" {
		req.ContentLength, _ = strconv.ParseInt(v, 10, 64)
	}
	if c.Host != "" {
		req.Host = c.Host
	}
	if c.Conf.RequestBodyClose {
		req.Close = true
	}
	secretID := config.Base.SecretID
	secretKey := config.Base.SecretKey
	secretToken := config.Base.SessionToken
	if param.SecretID != "" {
		secretID = param.SecretID
	}
	if param.SecretKey != "" {
		secretKey = param.SecretKey
	}
	if param.SessionToken != "" {
		secretToken = param.SessionToken
	}
	client := &http.Client{
		Transport: &cosgo.AuthorizationTransport{
			SecretID:     secretID,
			SecretKey:    secretKey,
			SessionToken: secretToken,
		},
	}
	req = req.WithContext(ctx)
	resp, err = client.Do(req)
	if err != nil {
		// If we got an error, and the context has been canceled,
		// the context's error is probably more useful.
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		return nil, err
	}
	defer func() {
		if closeBody {
			// Close the body to let the Transport reuse the connection
			io.Copy(ioutil.Discard, resp.Body)
			resp.Body.Close()
		}
	}()

	err = checkResponse(resp)
	if err != nil {
		// StatusCode != 2xx when Get Object
		if !closeBody {
			resp.Body.Close()
		}
		// even though there was an error, we still return the response
		// in case the caller wants to inspect it further
		return resp, err
	}

	return resp, err
}

// addHeaderOptions adds the parameters in opt as Header fields to req
func addHeaderOptions(header http.Header, opt interface{}) (http.Header, error) {
	v := reflect.ValueOf(opt)
	if v.Kind() == reflect.Ptr && v.IsNil() {
		return header, nil
	}

	h, err := httpheader.Header(opt)
	if err != nil {
		return nil, err
	}

	for key, values := range h {
		for _, value := range values {
			header.Add(key, value)
		}
	}
	return header, nil
}

func checkResponse(r *http.Response) error {
	if c := r.StatusCode; 200 <= c && c <= 299 {
		return nil
	}
	errorResponse := &cosgo.ErrorResponse{Response: r}
	data, err := ioutil.ReadAll(r.Body)
	if err == nil && data != nil {
		xml.Unmarshal(data, errorResponse)
	}
	// 是否为 json 格式
	if errorResponse.Code == "" {
		ctype := strings.TrimLeft(r.Header.Get("Content-Type"), " ")
		if strings.HasPrefix(ctype, "application/json") {
			var jerror jsonError
			json.Unmarshal(data, &jerror)
			errorResponse.Code = strconv.Itoa(jerror.Code)
			errorResponse.Message = jerror.Message
			errorResponse.RequestID = jerror.RequestID
		}

	}
	return errorResponse
}
