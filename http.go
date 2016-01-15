package httpclient

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"io/ioutil"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"sync"
	"time"

	"go.intra.xiaojukeji.com/server/msgcenter/app/common/logger"
)

func init() {

}

// TODO 默认的content-type，是否可以不用设置？
var DefaultContentType = "application/x-www-form-urlencoded"

// 暂时不支持cookie
type HttpRequest struct {
	Method       string
	Url          string
	ContentType  string
	Keeplive     bool
	Timeout      int
	Client       *http.Client
	Param        interface{}
	HeadParam    interface{}
	ReqBody      *bytes.Buffer
	RespCode     int
	RespBody     string
	IsMultipart  bool
	IsHttps      bool
	sync.RWMutex // TODO 此处是否需要加锁？
}

// 设置请求方法，默认是GET
func (this *HttpRequest) SetMethod(method string) {
	this.Method = method
}

// 设置POST请求的内容请求方式
func (this *HttpRequest) SetMultipart() {
	this.IsMultipart = true
}

// 设置keepalive，需要重新设置transport
func (this *HttpRequest) SetKeepAlive(timeout int) {
	if this.Keeplive {
		return
	}
	this.Keeplive = true
	this.Client.Transport = &http.Transport{
		Dial: func(netw, addr string) (net.Conn, error) {
			deadline := time.Now().Add(time.Duration(timeout) * time.Millisecond)
			c, err := net.DialTimeout(netw, addr, time.Duration(timeout)*time.Millisecond)
			if err != nil {
				return nil, err
			}
			c.SetDeadline(deadline)
			return c, nil
		},
		MaxIdleConnsPerHost: 20,
		DisableKeepAlives:   false,
	}
}

// 默认只能设置三个参数
func NewHttpRequest(url string, method string, timeout int) *HttpRequest {
	client := &http.Client{
		Timeout: time.Duration(timeout) * time.Millisecond,
	}
	return &HttpRequest{
		Url:      url,
		Method:   method,
		Timeout:  timeout,
		Client:   client,
		Keeplive: false,
		ReqBody:  new(bytes.Buffer),
	}
}

//get参数放到请求的url中，post则直接post
func GetPost(logid string, url string, getParam map[string]interface{}, postParam map[string]interface{}, timeout int) (string, int, error) {
	h := NewHttpRequest(url, "POST", timeout)
	if getParam != nil {
		h.Url = h.Url + "?"
		for k, v := range getParam {
			vStr, convErr := convertToString(v)
			if convErr != nil {
				return "", 0, convErr
			}
			h.Url = h.Url + k + "=" + vStr + "&"
		}
	}

	paramStr := ""
	for k, v := range postParam {
		vStr, convErr := convertToString(v)
		if convErr != nil {
			return "", 0, convErr
		}
		paramStr = paramStr + k + "=" + vStr + "&"
	}
	h.ReqBody = bytes.NewBufferString(paramStr)

	if err := h.Exec(logid, "POST", h.Url); err != nil {
		return "", 0, err
	}
	return h.RespBody, h.RespCode, nil

}

//对外的Get请求，此处是否需要加上keepalive
func Get(logid string, url string, param map[string]interface{}, timeout int) (string, int, error) {
	h := NewHttpRequest(url, "GET", timeout)
	if param != nil {
		h.Url = h.Url + "?"
		for k, v := range param {
			vStr, convErr := convertToString(v)
			if convErr != nil {
				return "", 0, convErr
			}
			h.Url = h.Url + k + "=" + vStr + "&"
		}
	}
	if err := h.Exec(logid, "GET", h.Url); err != nil {
		return "", 0, err
	}
	return h.RespBody, h.RespCode, nil

}

// POST请求，支持application和multipart两种传输方式，默认是application
func PostNative(logid string, url string, param interface{}, headParam interface{}, timeout int, IsMultipart bool, IsHttps bool) (string, int, error) {
	h := NewHttpRequest(url, "POST", timeout)

	//设置请求的head
	h.HeadParam = headParam

	//是否是https请求
	h.IsHttps = IsHttps

	//设置Post参数
	if param != nil {
		v := reflect.ValueOf(param)
		t := v.Type()
		switch t.Kind() {
		case reflect.String:
			temp, _ := param.(string)
			h.ReqBody = bytes.NewBufferString(temp)
			break
		case reflect.Map:
			paramMap, _ := param.(map[string]interface{})
			if IsMultipart {
				h.ReqBody = new(bytes.Buffer)
				w := multipart.NewWriter(h.ReqBody)
				for k, v := range paramMap {
					vStr, convErr := convertToString(v)
					if convErr != nil {
						return "", 0, convErr
					}
					w.WriteField(k, vStr)
				}
				w.Close()
				h.IsMultipart = IsMultipart
				h.ContentType = w.FormDataContentType()
			} else {
				paramStr := ""
				for k, v := range paramMap {
					vStr, convErr := convertToString(v)
					if convErr != nil {
						return "", 0, convErr
					}
					paramStr = paramStr + k + "=" + vStr + "&"
				}
				h.ReqBody = bytes.NewBufferString(paramStr)
			}
			break
		}
	}

	if err := h.Exec(logid, "POST", h.Url); err != nil {
		return "", 0, err
	}
	return h.RespBody, h.RespCode, nil
}

func Post(logid string, url string, param interface{}, headParam interface{}, timeout int, IsMultipart bool) (string, int, error) {
	return PostNative(logid, url, param, headParam, timeout, IsMultipart, false)
}

func PostHttps(logid string, url string, param interface{}, headParam interface{}, timeout int, IsMultipart bool) (string, int, error) {
	return PostNative(logid, url, param, headParam, timeout, IsMultipart, true)
}

// PUT请求，数据格式string
func Put(logid string, url string, param string, timeout int) (string, int, error) {
	h := NewHttpRequest(url, "PUT", timeout)
	h.ReqBody = bytes.NewBufferString(param)
	if err := h.Exec(logid, "PUT", h.Url); err != nil {
		return "", 0, err
	}
	return h.RespBody, h.RespCode, nil
}

// HEAD请求，数据格式string
func Head(logid string, url string, param string, timeout int) (string, int, error) {
	h := NewHttpRequest(url, "HEAD", timeout)
	h.ReqBody = bytes.NewBufferString(param)
	if err := h.Exec(logid, "HEAD", h.Url); err != nil {
		return "", 0, err
	}
	return h.RespBody, h.RespCode, nil
}

// DELETE请求，数据格式string
func Delete(logid string, url string, param string, timeout int) (string, int, error) {
	h := NewHttpRequest(url, "DELETE", timeout)
	h.ReqBody = bytes.NewBufferString(param)
	if err := h.Exec(logid, "DELETE", h.Url); err != nil {
		return "", 0, err
	}
	return h.RespBody, h.RespCode, nil
}

// HTTP请求的具体执行方法
func (this *HttpRequest) Exec(logid string, method string, RealUrl string) error {
	request, err := http.NewRequest(method, RealUrl, this.ReqBody)
	if err != nil {
		return err
	}

	// set https
	if this.IsHttps {
		this.Client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}

	if this.Keeplive {
		request.Header.Set("Connection", "Keep-Alive")
	}
	if method == "POST" {
		if this.IsMultipart {
			request.Header.Set("Content-Type", this.ContentType)
		} else {
			request.Header.Set("Content-Type", DefaultContentType)
		}
	}

	// set header
	if this.HeadParam != nil {
		if paramMap, ok := this.HeadParam.(map[string]interface{}); ok {
			for k, v := range paramMap {
				value := fmt.Sprintf("%v", v)
				request.Header.Set(k, value)
			}
		}
	}

	startTime := time.Now()
	response, err := this.Client.Do(request)
	costTime := time.Now().Sub(startTime).Nanoseconds() / (1000 * 1000)
	if err != nil {
		logger.Errorf("[logid=%v][where=httpclient.Do][duration=%v(ms)][error=%v][url=%s]", logid, costTime, err.Error(), RealUrl)
		return err
	}
	defer response.Body.Close()
	this.RespCode = response.StatusCode
	contents, err := ioutil.ReadAll(response.Body)
	if err != nil {
		logger.Errorf("[logid=%v][where=httpclient.ReadAll][duration=%v(ms)][error=%v][url=%s][respCode=%d]", logid, costTime, err.Error(), RealUrl, this.RespCode)
		return err
	}
	this.RespBody = string(contents)
	logger.Infof("[logid=%v][where=httpclient][duration=%v(ms)][url=%s][respCode=%d][respBody=%s]", logid, costTime, RealUrl, this.RespCode, this.RespBody)
	return nil
}

func convertToString(d interface{}) (v string, err error) {
	f := reflect.ValueOf(d)
	switch f.Interface().(type) {
	case int, int8, int16, int32, int64:
		v = strconv.FormatInt(f.Int(), 10)
	case uint, uint8, uint16, uint32, uint64:
		v = strconv.FormatUint(f.Uint(), 10)
	case float32:
		v = fmt.Sprintf("%v", d)
	case float64:
		v = fmt.Sprintf("%v", d)
	case []byte:
		v = string(f.Bytes())
	case string:
		v = url.QueryEscape(f.String())
	default:
		err = errors.New("Unsupport data type in http. Only support primitives")
	}

	return
}
