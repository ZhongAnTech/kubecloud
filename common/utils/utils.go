package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/astaxie/beego"
	"k8s.io/apimachinery/pkg/util/uuid"
)

func ContainsString(slice []string, target string) bool {
	for _, item := range slice {
		if item == target {
			return true
		}
	}
	return false
}

func ContainsInt(slice []int, target int) bool {
	for _, item := range slice {
		if item == target {
			return true
		}
	}
	return false
}

func NowTime() time.Time {
	t, _ := time.Parse("2006-01-02 15:04:05", time.Now().Local().Format("2006-01-02 15:04:05"))
	return t
}

func StructToMap(v interface{}) (map[string]interface{}, error) {
	stream, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	err = json.Unmarshal(stream, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func MapToStruct(src interface{}, dest interface{}) error {
	stream, err := json.Marshal(src)
	if err != nil {
		return err
	}
	return json.Unmarshal(stream, dest)
}

func SimpleJsonMarshal(src interface{}, def string) string {
	if src == nil {
		return def
	}
	dest, err := json.Marshal(src)
	if err != nil {
		beego.Error("marshal ", src, "failed:", err)
		return def
	}
	return string(dest)
}

func SimpleJsonUnmarshal(src string, v interface{}) {
	if err := json.Unmarshal([]byte(src), v); err != nil {
		beego.Error("unmarshal ", src, "failed:", err)
	}
}

func GetLabelIntValue(label map[string]string, key string, def int) int {
	if val, ok := label[key]; ok {
		num, err := strconv.Atoi(val)
		if err == nil {
			return num
		}
		beego.Error(val, "strconv atoi failed:", err)
	}
	return def
}

func GetLabelStringValue(label map[string]string, key string, def string) string {
	if val, ok := label[key]; ok {
		return val
	}
	return def
}

func GetInt64Value(labels map[string]string, key string, def int64) int64 {
	if rawValue, ok := labels[key]; ok {
		value, err := strconv.ParseInt(rawValue, 10, 64)
		if err == nil {
			return value
		}
		beego.Error(fmt.Sprintf("Unable to parse %q: %q, falling back to %v. %v", key, rawValue, def, err))
	}
	return def
}

// ParseServiceLabel return service name and its namespace
func ParseServiceLabel(label string) (string, string) {
	rawname := strings.TrimSuffix(label, ".svc.cluster.local")
	items := strings.Split(rawname, ".")
	if len(items) > 1 {
		return strings.Join(items[0:len(items)-1], "."), items[len(items)-1]
	}
	return items[0], ""
}

// ObjectIsEqual test equality of two objects.
// Not working for 1. floating number,
// 2. objects with same keys but different in define order
func ObjectIsEqual(obj1, obj2 interface{}) bool {
	obj1Bytes, _ := json.Marshal(obj1)
	obj2Bytes, _ := json.Marshal(obj2)
	return bytes.Equal(obj1Bytes, obj2Bytes)
}

func GetRootPath(path string) string {
	if path == "" {
		return "/"
	}
	return path
}

func AddRootPath(path string) string {
	if !strings.HasPrefix(path, "/") {
		return "/" + path
	}
	return path
}

func PathsIsEqual(path1, path2 string) bool {
	return GetRootPath(path1) == GetRootPath(path2)
}

func GenerateStandardAppName(appname string) string {
	appname = strings.ToLower(appname)
	return strings.ReplaceAll(appname, "_", "-")
}

type PageInfo struct {
	TotalNum  int64 `json:"total_num"`
	PageIndex int   `json:"page_index"`
	PageSize  int   `json:"page_size"`
}

type QueryResult struct {
	Base PageInfo    `json:"base"`
	List interface{} `json:"list"`
}

type Pagination struct {
	Offset int
	Limit  int
}

var imageUrlPattern = regexp.MustCompile(`(?P<Host>[^/]+)/(?P<Name>[^/]+(?:/[^/]+)*)\:(?P<Tag>.*)`)

type ImageUrl struct {
	Host string `json:"host"`
	Name string `json:"name"`
	Tag  string `json:"tag"`
}

func InitQueryResult(List interface{}, query *FilterQuery) *QueryResult {
	res := &QueryResult{List: List}
	if query != nil {
		res.Base = PageInfo{
			PageIndex: query.PageIndex,
			PageSize:  query.PageSize,
		}
	}
	return res
}

// ParseImageUrl parse docker image URL
func ParseImageUrl(url string) *ImageUrl {
	matched := imageUrlPattern.FindStringSubmatch(url)
	if len(matched) != len(imageUrlPattern.SubexpNames()) {
		return nil
	}
	return &ImageUrl{
		Host: matched[1],
		Name: matched[2],
		Tag:  matched[3],
	}
}

// ToInterfaceSlice convert slice of any type to slice of interface
func ToInterfaceSlice(data interface{}) []interface{} {
	input := reflect.ValueOf(data)
	if input.Kind() != reflect.Slice {
		panic("ToInterfaceSlice given a non-slice type")
	}
	output := make([]interface{}, input.Len())
	for i := 0; i < input.Len(); i++ {
		output[i] = input.Index(i).Interface()
	}
	return output
}

// GoThrough call handler on each item of slice in concurrence
func GoThrough(slice []interface{}, handler func(interface{}) interface{}, limit int) interface{} {
	if limit < 1 {
		panic("GoThrough limit must be positive")
	}
	var err interface{}
	size := len(slice)
	for begin := 0; begin < size; begin = begin + limit {
		end := begin + limit
		if end > size {
			end = size
		}
		chunkSize := end - begin
		var wg sync.WaitGroup
		wg.Add(chunkSize)
		var mux sync.Mutex
		for idx := begin; idx < end; idx = idx + 1 {
			go func(data interface{}) {
				defer wg.Done()
				ret := handler(data)
				if ret != nil {
					mux.Lock()
					defer mux.Unlock()
					if err == nil {
						err = ret
					}
				}
			}(slice[idx])
		}
		wg.Wait()
		if err != nil {
			break
		}
	}
	return err
}

// WaitGroup ...
type WaitGroup struct {
	sync.WaitGroup
}

// Go ...
func (wg *WaitGroup) Go(fn func(...interface{}), args ...interface{}) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		fn(args...)
	}()
}

// DeepCopy deep copy from src object to dest object.
// NOTE: pointers of the src object should refer to different addresses
func DeepCopy(dest, src interface{}) error {
	bytes, err := json.Marshal(src)
	if err != nil {
		return err
	}
	return json.Unmarshal(bytes, dest)
}

func NewUUID() string {
	return string(uuid.NewUUID())
}

func ExecCommand(dir, name string, arg ...string) (string, error) {
	cmd := exec.Command(name, arg...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func GetIntFromLabel(labels map[string]string, key string, def int) int {
	if idstr, ok := labels[key]; ok {
		id, err := strconv.Atoi(idstr)
		if err != nil {
			beego.Warn(fmt.Sprintf("get int from labels for key %s failed, %v!", key, err))
			return def
		}
		return id
	}
	return def
}
