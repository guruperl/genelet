package genelet

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"
)

var HTTPClient = &http.Client{Timeout: 15 * time.Second}

func Invoke0(any interface{}, name string, args ...interface{}) {
	_, _ = TryInvoke(any, name, args...)
}

func Invoke(any interface{}, name string, args ...interface{}) []reflect.Value {
	out, err := TryInvoke(any, name, args...)
	if err != nil {
		return []reflect.Value{reflect.ValueOf(err)}
	}
	return out
}

func InvokeVoid(any interface{}, name string, args ...interface{}) error {
	out, err := TryInvoke(any, name, args...)
	if err != nil {
		return err
	}
	if len(out) != 0 {
		return Err(1051, name+" returned values where none were expected")
	}
	return nil
}

func InvokeOptionalVoid(any interface{}, name string, args ...interface{}) error {
	if any == nil {
		return nil
	}
	if !reflect.ValueOf(any).MethodByName(name).IsValid() {
		return nil
	}
	return InvokeVoid(any, name, args...)
}

func InvokeError(any interface{}, name string, args ...interface{}) error {
	out, err := TryInvoke(any, name, args...)
	if err != nil {
		return err
	}
	if len(out) != 1 {
		return Err(1051, name+" must return one error value")
	}
	if out[0].Kind() == reflect.Interface && out[0].IsNil() {
		return nil
	}
	if out[0].Kind() == reflect.Ptr && out[0].IsNil() {
		return nil
	}
	if out[0].Interface() == nil {
		return nil
	}
	err, ok := out[0].Interface().(error)
	if !ok {
		return Err(1051, name+" did not return an error")
	}
	return err
}

func TryInvoke(any interface{}, name string, args ...interface{}) (out []reflect.Value, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = Err(1051, fmt.Sprintf("%s panicked during dispatch: %v", name, r))
			out = nil
		}
	}()
	if any == nil {
		return nil, Err(1051, name+" dispatch target is nil")
	}
	method := reflect.ValueOf(any).MethodByName(name)
	if !method.IsValid() {
		return nil, Err(1051, name+" method does not exist")
	}
	methodType := method.Type()
	fixedArgs := methodType.NumIn()
	if methodType.IsVariadic() {
		fixedArgs--
		if len(args) < fixedArgs {
			return nil, Err(1051, fmt.Sprintf("%s expects at least %d args, got %d", name, fixedArgs, len(args)))
		}
	} else if methodType.NumIn() != len(args) {
		return nil, Err(1051, fmt.Sprintf("%s expects %d args, got %d", name, methodType.NumIn(), len(args)))
	}
	inputs := make([]reflect.Value, len(args))
	for i := range args {
		want := methodType.In(i)
		if methodType.IsVariadic() && i >= fixedArgs {
			want = methodType.In(methodType.NumIn() - 1).Elem()
		}
		value, err := dispatchArgValue(reflect.ValueOf(args[i]), want)
		if err != nil {
			return nil, Err(1051, fmt.Sprintf("%s arg %d: %s", name, i, err.Error()))
		}
		inputs[i] = value
	}
	return method.Call(inputs), nil
}

func dispatchArgValue(value reflect.Value, want reflect.Type) (reflect.Value, error) {
	if !value.IsValid() {
		return reflect.Zero(want), nil
	}
	if value.Type().AssignableTo(want) {
		return value, nil
	}
	if value.Type().ConvertibleTo(want) {
		return value.Convert(want), nil
	}
	if nested, ok := embeddedAssignable(value, want); ok {
		return nested, nil
	}
	return reflect.Value{}, fmt.Errorf("%s is not assignable to %s", value.Type(), want)
}

func embeddedAssignable(value reflect.Value, want reflect.Type) (reflect.Value, bool) {
	if value.Kind() == reflect.Interface && !value.IsNil() {
		value = value.Elem()
	}
	if value.Kind() != reflect.Ptr || value.IsNil() || value.Elem().Kind() != reflect.Struct {
		return reflect.Value{}, false
	}
	value = value.Elem()
	valueType := value.Type()
	for i := 0; i < value.NumField(); i++ {
		fieldInfo := valueType.Field(i)
		if !fieldInfo.Anonymous {
			continue
		}
		field := value.Field(i)
		candidates := []reflect.Value{field}
		if field.CanAddr() {
			candidates = append(candidates, field.Addr())
		}
		for _, candidate := range candidates {
			if candidate.IsValid() && candidate.Type().AssignableTo(want) {
				return candidate, true
			}
			if nested, ok := embeddedAssignable(candidate, want); ok {
				return nested, true
			}
		}
	}
	return reflect.Value{}, false
}

func Interface2String(v interface{}) string {
	switch u := v.(type) {
	case nil:
		return ""
	case string:
		return u
	case []uint8:
		return string(u)
	case int, int32, int64, uint, uint32, uint64:
		return fmt.Sprintf("%d", u)
	case float32, float64:
		return fmt.Sprintf("%f", u)
	case bool:
		return strconv.FormatBool(u)
	default:
		if b, err := json.Marshal(u); err == nil {
			return string(b)
		}
		return fmt.Sprint(u)
	}
}

func SortMapMd5(secret, md5_name string, q url.Values) string {
	ks := []string{}
	vs := []string{}
	for k := range q {
		if k == md5_name {
			continue
		}
		ks = append(ks, k)
	}
	sort.Strings(ks)
	for _, k := range ks {
		vs = append(vs, q.Get(k))
	}
	return Digest(secret, strings.Join(vs, ""))
}

func Digest64(key string, message ...string) string {
	h := hmac.New(sha1.New, []byte(key))
	//h.Write([]byte(Joinstrings("", message ...)))
	h.Write([]byte(strings.Join(message, "")))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func Digest(key string, message ...string) string {
	str := Digest64(key, message...)
	str = strings.Replace(str, "+", "|", -1)
	str = strings.Replace(str, "/", "-", -1)
	str = strings.Replace(str, "=", "_", -1)
	return str
}

func Digesting(key string, message ...string) string {
	h := hmac.New(sha1.New, []byte(key))
	h.Write([]byte(strings.Join(message, "")))
	return url.QueryEscape(string(h.Sum(nil)))
}

func Unix_timestamp() int {
	return int(time.Now().Unix())
}

func Ip2int(ip string) uint32 {
	netip := net.ParseIP(ip)
	if netip == nil {
		return 0
	}
	to4 := netip.To4()
	if to4 == nil {
		if len(netip) < 16 {
			return 0
		}
		return binary.BigEndian.Uint32(netip[12:16])
	}
	return binary.BigEndian.Uint32(to4)
}

func Int2ip(ip uint32) string {
	netip := make(net.IP, 4)
	binary.BigEndian.PutUint32(netip, ip)
	return netip.String()
}

func Stripchars(chr, str string) string {
	return strings.Map(func(r rune) rune {
		if strings.IndexRune(chr, r) < 0 {
			return r
		}
		return -1
	}, str)
}

func Filtering(vs []string, f func(string) bool) []string {
	vsf := make([]string, 0)
	for _, v := range vs {
		if f(v) {
			vsf = append(vsf, v)
		}
	}
	return vsf
}

func Map(vs []string, f func(string) string) []string {
	vsm := make([]string, len(vs))
	for i, v := range vs {
		vsm[i] = f(v)
	}
	return vsm
}

func Index(vs []string, t string) int {
	for i, v := range vs {
		if v == t {
			return i
		}
	}
	return -1
}

func Grep(vs []string, t string) bool {
	return Index(vs, t) >= 0
}

/*
func Grep(v interface{}, in interface{}) (ok bool) {
	val := reflect.Indirect(reflect.ValueOf(in))
	switch val.Kind() {
	case reflect.Slice, reflect.Array:
		for i:=0; i < val.Len(); i++ {
			if ok = v == val.Index(i).Interface(); ok {
				return
			}
		}
	}
	return
}
*/

func Do(method string, url string, form url.Values, header map[string]string) ([]byte, error) {
	var req *http.Request
	var err error
	if form == nil {
		req, err = http.NewRequest(method, url, nil)
	} else if method == "GET" {
		req, err = http.NewRequest(method, url+"?"+form.Encode(), nil)
	} else {
		query := []byte(form.Encode())
		req, err = http.NewRequest(method, url, bytes.NewBuffer(query))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	if err != nil {
		return nil, err
	}
	if header != nil {
		for k, v := range header {
			req.Header.Set(k, v)
		}
	}

	res, err := HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}

	return process_(res)
}

func Get(url string, form url.Values) ([]byte, error) {
	target := url
	if form == nil {
		res, err := HTTPClient.Get(target)
		if err != nil {
			return nil, err
		}
		return process_(res)
	}

	res, err := HTTPClient.Get(target + "?" + form.Encode())
	if err != nil {
		return nil, err
	}
	return process_(res)
}

func Post(url string, form url.Values) ([]byte, error) {
	res, err := HTTPClient.PostForm(url, form)
	if err != nil {
		return nil, err
	}
	return process_(res)
}

func PostFile(url string, fn string, header map[string]string) ([]byte, error) {
	buf, err := os.Open(fn)
	if err != nil {
		return nil, err
	}
	defer buf.Close()

	req, err := http.NewRequest("POST", url, buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if header != nil {
		for k, v := range header {
			req.Header.Set(k, v)
		}
	}

	res, err := HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}

	return process_(res)
}

func process_(res *http.Response) ([]byte, error) {
	body, err := io.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return nil, err
	}

	return body, nil
}

func Do_hash(method string, url string, form url.Values, header map[string]string) (map[string]interface{}, error) {
	body, err := Do(method, url, form, header)
	if err != nil {
		return nil, err
	}

	return To_hash(body)
}

func Get_hash(url string, form url.Values) (map[string]interface{}, error) {
	body, err := Get(url, form)
	if err != nil {
		return nil, err
	}

	return To_hash(body)
}

func Post_hash(url string, form url.Values) (map[string]interface{}, error) {
	body, err := Post(url, form)
	if err != nil {
		return nil, err
	}

	return To_hash(body)
}

func PostFile_hash(url string, fn string, header map[string]string) (map[string]interface{}, error) {
	body, err := PostFile(url, fn, header)
	if err != nil {
		return nil, err
	}

	return To_hash(body)
}

func To_hash(body []byte) (map[string]interface{}, error) {
	tmp := make(map[string]interface{})
	err := json.Unmarshal(body, &tmp)
	if err != nil {
		return nil, err
	}
	return tmp, nil
}

func To_slice(body []byte) ([]map[string]interface{}, error) {
	tmp := make([]map[string]interface{}, 0)
	err := json.Unmarshal(body, &tmp)
	if err != nil {
		return nil, err
	}
	return tmp, nil
}

func MultipartRequest(url string, form url.Values, paramName, path string) (*http.Request, error) {

	b := new(bytes.Buffer)
	w := multipart.NewWriter(b)

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	fw, err := w.CreateFormFile(paramName, path)
	if err != nil {
		return nil, err
	}

	if _, err = io.Copy(fw, f); err != nil {
		return nil, err
	}

	for key, val := range form {
		_ = w.WriteField(key, val[0])
	}
	w.Close()

	request, err := http.NewRequest("POST", url, b)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", w.FormDataContentType())

	return request, nil
}

func MultipartUpload(url string, form url.Values, header map[string]string, paramName, path string) ([]byte, error) {
	req, err := MultipartRequest(url, form, paramName, path)
	if err != nil {
		return nil, err
	}

	if header != nil {
		for k, v := range header {
			req.Header.Set(k, v)
		}
	}

	res, err := HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}

	return process_(res)
}
