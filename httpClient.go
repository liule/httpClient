package main

import (
	"bytes"
	"errors"
	"io/ioutil"
	"mime/multipart"
	"net"
	"net/http"
	"reflect"
	"sync"
	"time"
)

func init() {

}

// 暂时不支持cookie
type HttpRequest struct {
	Method       string
	Host         string
	Path         string
	ContentType  string
	Keeplive     bool
	Timeout      int
	Client       *http.Client
	Param        interface{}
	ReqBody      *bytes.Buffer
	RespCode     int
	RespBody     string
	IsMultipart  bool
	sync.RWMutex // TODO 此处是否需要加锁？
}

func (this *HttpRequest) SetMethod(method string) {
	this.Method = method
}

func (this *HttpRequest) SetMultipart() {
	this.IsMultipart = true
}

func (this *HttpRequest) SetHost(host string) {
	this.Host = host
}

// 设置keepalive，需要重新设置transport
func (this *HttpRequest) SetKeepAlive(timeout int) {
	if this.Keeplive {
		return
	}
	this.Keeplive = true
	this.Client.Transport = &http.Transport{
		Dial: func(netw, addr string) (net.Conn, error) {
			deadline := time.Now().Add(time.Duration(timeout) * time.Second)
			c, err := net.DialTimeout(netw, addr, time.Duration(timeout)*time.Second)
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

// 接收参数 string 或者 map[string]string
func (this *HttpRequest) SetParams(param interface{}) error {
	v := reflect.ValueOf(param)
	t := v.Type()
	switch t.Kind() {
	case reflect.String:
		this.Param = param
	case reflect.Map:
		this.Param = param
	default:
		return errors.New("param type error")
	}

	return nil
}

// 默认只能设置三个参数
func NewHttpRequest(host string, method string, timeout int) *HttpRequest {
	client := &http.Client{
		Timeout: time.Duration(timeout) * time.Second,
	}
	return &HttpRequest{
		Host:        host,
		Method:      method,
		Timeout:     timeout,
		Client:      client,
		Keeplive:    false,
		ContentType: "application/x-www-form-urlencoded",
	}
}

// 请求的响应码
func (this *HttpRequest) StatusCode() int {
	return this.RespCode
}

// 请求的响应结果
func (this *HttpRequest) ResponeBody() string {
	return this.RespBody
}

// Get请求
func (this *HttpRequest) Get() error {
	RealUrl := this.Host + this.Path
	if this.Param != nil {
		v := reflect.ValueOf(this.Param)
		t := v.Type()
		switch t.Kind() {
		case reflect.String:
			temp, _ := this.Param.(string)
			RealUrl = RealUrl + "?" + temp
		case reflect.Map:
			temp, _ := this.Param.(map[string]string)
			RealUrl = RealUrl + "?"
			for k, v := range temp {
				RealUrl = RealUrl + k + "=" + v + "&"
			}
		}
	}
	println("Get-url", RealUrl)
	return this.Exec("GET", RealUrl)
}

func (this *HttpRequest) Post() error {
	RealUrl := this.Host + this.Path
	if this.Param != nil {
		v := reflect.ValueOf(this.Param)
		t := v.Type()
		switch t.Kind() {
		case reflect.String:
			temp, _ := this.Param.(string)
			this.ReqBody = bytes.NewBufferString(temp)
			break
		case reflect.Map:
			param, _ := this.Param.(map[string]string)
			if this.IsMultipart {
				this.ReqBody = new(bytes.Buffer)
				w := multipart.NewWriter(this.ReqBody)
				for k, v := range param {
					w.WriteField(k, v)
				}
				w.Close()
				this.ContentType = w.FormDataContentType()
			} else {
				paramStr := ""
				for k, v := range param {
					paramStr = paramStr + k + "=" + v + "&"
				}
				this.ReqBody = bytes.NewBufferString(paramStr)
			}
			break
		}
	}

	return this.Exec("POST", RealUrl)
}

func (this *HttpRequest) Put() error {
	RealUrl := this.Host + this.Path
	if this.Param != nil {
		if temp, ok := this.Param.(string); ok {
			this.ReqBody = bytes.NewBufferString(temp)
		}
	}
	return this.Exec("PUT", RealUrl)
}

func (this *HttpRequest) Head() error {
	RealUrl := this.Host + this.Path
	if this.Param != nil {
		if temp, ok := this.Param.(string); ok {
			this.ReqBody = bytes.NewBufferString(temp)
		}
	}
	return this.Exec("HEAD", RealUrl)
}

func (this *HttpRequest) Delete() error {
	RealUrl := this.Host + this.Path
	if this.Param != nil {
		if temp, ok := this.Param.(string); ok {
			this.ReqBody = bytes.NewBufferString(temp)
		}
	}
	return this.Exec("DELETE", RealUrl)
}

func (this *HttpRequest) Exec(method string, RealUrl string) error {
	request, err := http.NewRequest(method, RealUrl, this.ReqBody)
	if err != nil {
		return err
	}
	if this.Keeplive {
		request.Header.Set("Connection", "Keep-Alive")
	}
	if this.IsMultipart {
		println("Content-Type", this.ContentType)
		request.Header.Set("Content-Type", this.ContentType)
	}
	response, err := this.Client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	this.RespCode = response.StatusCode
	contents, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}
	this.RespBody = string(contents)
	return nil
}

func main() {
	host := "http://10.108.87.71:7997"
	httpTest := NewHttpRequest(host, "POST", 20)
	postMap := make(map[string]string, 5)
	postMap["a1234567"] = "aaaaaaa"
	postMap["b1234567"] = "bbbbbbb"
	postMap["c1234567"] = "ccccccc"
	postMap["d1234567"] = "ddddddd"
	postMap["e1234567"] = "eeeeeee"
	httpTest.SetParams("postMap")
	// httpTest.SetMultipart()

	if err := httpTest.Post(); err != nil {
		panic(err.Error())
	}
	println("code:", httpTest.StatusCode())
	println("response:", httpTest.ResponeBody())
}
