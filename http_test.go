package httpclient

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

var timeout = 5

func readAllString(r io.Reader) (string, error) {
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		return "", err
	}
	return buf.String(), nil
}

var copyHandlerFunc = http.HandlerFunc(
	func(w http.ResponseWriter, r *http.Request) {
		result := ""
		defer r.Body.Close()
		switch r.Method {
		case "GET":
			result = r.RequestURI
		case "POST":
			r.ParseMultipartForm(32 << 20)
			if r.MultipartForm != nil {
				values := r.MultipartForm.Value["test"]
				if len(values) > 0 {
					fmt.Fprintf(w, values[0])
					return
				}
			}
			result = r.PostFormValue("test")
		case "PUT":
			temp, _ := ioutil.ReadAll(r.Body)
			result = string(temp)
		case "DELETE":
			temp, _ := ioutil.ReadAll(r.Body)
			result = string(temp)
		default:
		}
		fmt.Fprintf(w, result)
		// io.Copy(w, r.Body)
	},
)

func TestGet(t *testing.T) {
	ts := httptest.NewServer(copyHandlerFunc)
	defer ts.Close()

	body, code, err := Get(ts.URL+"/TestGet", nil, timeout)
	if err != nil {
		t.Fatal(err)
	}

	if code != 200 || body != "/TestGet" {
		t.Fatal("get response error")
	}
}

func TestPost(t *testing.T) {
	ts := httptest.NewServer(copyHandlerFunc)
	defer ts.Close()
	param := make(map[string]interface{})
	param["test"] = "TestPost"
	body, code, err := Post(ts.URL, param, nil, timeout, false)
	if err != nil {
		t.Fatal(err)
	}

	if code != 200 || body != "TestPost" {
		t.Fatal("post response error")
	}

	param["test"] = "TestMultipart"
	body, code, err = Post(ts.URL, param, nil, timeout, true)
	if err != nil {
		t.Fatal(err)
	}
	if code != 200 || body != "TestMultipart" {
		t.Fatal("multipart response error")
	}
}

func TestGetPost(t *testing.T) {
	ts := httptest.NewServer(copyHandlerFunc)
	defer ts.Close()
	param := make(map[string]interface{})
	param["test"] = "TestPost"

	body, code, err := GetPost(ts.URL, param, param, timeout)
	if err != nil {
		t.Fatal(err)
	}

	if code != 200 || body != "TestPost" {
		t.Fatal("post response error")
	}

	param["test"] = "TestMultipart"
	body, code, err = Post(ts.URL, param, nil, timeout, true)
	if err != nil {
		t.Fatal(err)
	}
	if code != 200 || body != "TestMultipart" {
		t.Fatal("multipart response error")
	}
}

func TestPut(t *testing.T) {
	ts := httptest.NewServer(copyHandlerFunc)
	defer ts.Close()

	body, code, err := Put(ts.URL, "TestPut", timeout)
	if err != nil {
		t.Fatal(err)
	}

	if code != 200 || body != "TestPut" {
		t.Fatal("put response error")
	}
}

func TestDelete(t *testing.T) {
	ts := httptest.NewServer(copyHandlerFunc)
	defer ts.Close()

	body, code, err := Delete(ts.URL, "TestDelete", timeout)
	if err != nil {
		t.Fatal(err)
	}

	if code != 200 || body != "TestDelete" {
		t.Fatal("delete response error")
	}
}

func TestTimeout(t *testing.T) {
	ts := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(time.Duration(timeout) * time.Millisecond)
			// w.WriteHeader(500)
			fmt.Fprintf(w, "success")
		}),
	)
	defer ts.Close()

	//超时时间返回
	body, code, err := Get(ts.URL+"/TestGet", nil, (timeout + 10))
	if err != nil {
		t.Fatal(err)
	}

	if code != 200 || body != "success" {
		t.Fatal("get response error")
	}

	body, code, err = Get(ts.URL+"/TestGet", nil, (timeout - 1))
	if err == nil {
		t.Fatal(err)
	}

}
