package utils

import (
	"fmt"
	"sort"
	"strconv"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContainsString(t *testing.T) {
	assert.False(t, ContainsString(nil, ""))
	slice := []string{"1", "2", "3"}
	assert.False(t, ContainsString(slice, "0"))
	assert.True(t, ContainsString(slice, "1"))
	assert.True(t, ContainsString(slice, "2"))
	assert.True(t, ContainsString(slice, "3"))
}

func TestContainsInt(t *testing.T) {
	assert.False(t, ContainsInt(nil, 0))
	slice := []int{1, 2, 3}
	assert.False(t, ContainsInt(slice, 0))
	assert.True(t, ContainsInt(slice, 1))
	assert.True(t, ContainsInt(slice, 2))
	assert.True(t, ContainsInt(slice, 3))
}

func TestGetLabelIntValue(t *testing.T) {
	labels := map[string]string{
		"a": "1",
		"b": "2",
		"c": "3",
	}
	t.Run("ShouldFoundValue", func(t *testing.T) {
		for k, v := range labels {
			num, _ := strconv.Atoi(v)
			got := GetLabelIntValue(labels, k, 0)
			assert.Equal(t, num, got)
		}
	})
	t.Run("ShouldFoundDefault", func(t *testing.T) {
		def := 0
		got := GetLabelIntValue(labels, "d", def)
		assert.Equal(t, def, got)
	})
}

func TestGetInt64Value(t *testing.T) {
	labels := map[string]string{
		"a": "4294967296",
		"b": "4294967297",
		"c": "4294967298",
	}
	t.Run("ShouldFoundValue", func(t *testing.T) {
		for k, v := range labels {
			num, _ := strconv.ParseInt(v, 10, 64)
			got := GetInt64Value(labels, k, 0)
			assert.Equal(t, num, got)
		}
	})
	t.Run("ShouldFoundDefault", func(t *testing.T) {
		var def int64 = 9223372036854775807
		got := GetInt64Value(labels, "d", def)
		assert.Equal(t, def, got)
	})
}

func TestObjectIsEqual(t *testing.T) {
	obj1 := struct {
		Number   int
		String   string
		IntArray []int
		Data     struct {
			String string
		}
	}{
		Number:   10,
		String:   "hello",
		IntArray: []int{1, 2, 3},
		Data: struct {
			String string
		}{
			String: "world",
		},
	}

	obj2 := struct {
		Data struct {
			String string
		}
		IntArray []int
		String   string
		Number   int
	}{
		Data: struct {
			String string
		}{
			String: "world",
		},
		IntArray: []int{1, 2, 3},
		String:   "hello",
		Number:   10,
	}

	assert.True(t, ObjectIsEqual(obj1, obj1))
	//assert.True(t, ObjectIsEqual(obj1, obj2))
	obj2.Data.String = ""
	assert.False(t, ObjectIsEqual(obj1, obj2))
}

func TestGetLabelStringValue(t *testing.T) {
	labels := map[string]string{
		"a": "this is a",
		"b": "this is b",
		"c": "this is c",
	}
	t.Run("ShouldFoundValue", func(t *testing.T) {
		for k, v := range labels {
			got := GetLabelStringValue(labels, k, "")
			assert.Equal(t, v, got)
		}
	})
	t.Run("ShouldFoundDefault", func(t *testing.T) {
		def := "this is d"
		got := GetLabelStringValue(labels, "d", def)
		assert.Equal(t, def, got)
	})
}

func TestParseImageUrl(t *testing.T) {
	t.Parallel()
	t.Run("PositiveCases", func(t *testing.T) {
		cases := [][]string{
			[]string{"10.10.10.10:5050", "devops/nginx", "stable-alpine"},
			[]string{"a.b.com", "devops/nginx", "1.2.0"},
			[]string{"a.b.com", "oops-nginx", "1.2.0"},
			[]string{"a.b.com", "c/d/e/f", "1.2.0"},
		}
		for _, item := range cases {
			str := fmt.Sprintf(`%s/%s:%s`, item[0], item[1], item[2])
			url := ParseImageUrl(str)
			assert.NotNil(t, url)
			assert.Equal(t, item[0], url.Host)
			assert.Equal(t, item[1], url.Name)
			assert.Equal(t, item[2], url.Tag)
		}
	})
	t.Run("NegativeCases", func(t *testing.T) {
		cases := [][]string{
			[]string{"a.b.c.com"},
			[]string{"a.b.c.com/c/d"},
			[]string{"a.b.c.com:1.2"},
		}
		for _, item := range cases {
			str := item[0]
			url := ParseImageUrl(str)
			assert.Nil(t, url)
		}
	})
}

func TestToInterfaceSlice(t *testing.T) {
	t.Run("EmptyStringSlice", func(t *testing.T) {
		input := []string{}
		output := ToInterfaceSlice(input)
		assert.Equal(t, []interface{}{}, output)
	})
	t.Run("StringSlice", func(t *testing.T) {
		input := []string{"a", "b", "c"}
		output := ToInterfaceSlice(input)
		assert.Equal(t, []interface{}{"a", "b", "c"}, output)
	})
}

func TestGoThrough(t *testing.T) {
	expected := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	data := ToInterfaceSlice(expected)
	t.Run("Concurrency", func(t *testing.T) {
		t.Run("1", func(t *testing.T) {
			var mux sync.Mutex
			actual := []int{}
			ret := GoThrough(data, func(arg interface{}) interface{} {
				item, ok := arg.(int)
				assert.True(t, ok)
				mux.Lock()
				defer mux.Unlock()
				actual = append(actual, item)
				return nil
			}, 1)
			assert.Nil(t, ret)
			assert.Equal(t, expected, actual)
		})
		t.Run("n", func(t *testing.T) {
			var mux sync.Mutex
			actual := []int{}
			ret := GoThrough(data, func(arg interface{}) interface{} {
				item, ok := arg.(int)
				assert.True(t, ok)
				mux.Lock()
				defer mux.Unlock()
				actual = append(actual, item)
				return nil
			}, len(data))
			assert.Nil(t, ret)
			sort.Ints(actual)
			assert.Equal(t, expected, actual)
		})
	})

	t.Run("Return", func(t *testing.T) {
		t.Run("1", func(t *testing.T) {
			hit := 0
			expectedStatus := "failed"
			var mux sync.Mutex
			ret := GoThrough(data, func(arg interface{}) interface{} {
				mux.Lock()
				hit++
				mux.Unlock()
				return expectedStatus
			}, 1)
			actualStatus, ok := ret.(string)
			assert.True(t, ok)
			assert.Equal(t, expectedStatus, actualStatus)
			assert.Equal(t, 1, hit)
		})
		t.Run("n", func(t *testing.T) {
			hit := 0
			expectedStatus := "failed"
			var mux sync.Mutex
			ret := GoThrough(data, func(arg interface{}) interface{} {
				mux.Lock()
				hit++
				mux.Unlock()
				return expectedStatus
			}, len(data))
			actualStatus, ok := ret.(string)
			assert.True(t, ok)
			assert.Equal(t, expectedStatus, actualStatus)
			assert.Equal(t, len(data), hit)
		})
	})
}

func TestWaitGroup(t *testing.T) {
	var wg WaitGroup
	count := 100
	expected := make([]int, count)
	actual := make([]int, count)
	for i := 0; i < count; i++ {
		expected[i] = i
		wg.Go(func(args ...interface{}) {
			val, ok := args[0].(int)
			assert.True(t, ok)
			actual[val] = val
		}, i)
	}
	wg.Wait()
	assert.Equal(t, expected, actual)
}

func TestDeepCopy(t *testing.T) {
	t.Parallel()
	t.Run("Int", func(t *testing.T) {
		dest := 0
		src := 123
		err := DeepCopy(&dest, &src)
		assert.Nil(t, err)
		assert.Equal(t, src, dest)
	})
	t.Run("String", func(t *testing.T) {
		dest := "hello"
		src := "world"
		err := DeepCopy(&dest, &src)
		assert.Nil(t, err)
		assert.Equal(t, dest, src)
	})
	t.Run("Slice", func(t *testing.T) {
		var dest []int
		src := []int{1, 2, 3}
		err := DeepCopy(&dest, &src)
		assert.Nil(t, err)
		assert.Equal(t, src, dest)
	})
	t.Run("Map", func(t *testing.T) {
		var dest map[int]string
		src := map[int]string{
			1: "a",
			2: "b",
			3: "c",
		}
		err := DeepCopy(&dest, &src)
		assert.Nil(t, err)
		assert.Equal(t, src, dest)
	})
	t.Run("Struct", func(t *testing.T) {
		type Data struct {
			Num int
			Str string
		}
		var dest Data
		src := Data{
			Num: 123,
			Str: "123",
		}
		err := DeepCopy(&dest, &src)
		assert.Nil(t, err)
		assert.Equal(t, src, dest)
	})
}
