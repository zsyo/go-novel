package util

import (
	"strings"

	"github.com/dop251/goja"
)

// JsEngine JavaScript引擎封装
type JsEngine struct {
	vm *goja.Runtime
}

// NewJsEngine 创建新的JavaScript引擎实例
func NewJsEngine() *JsEngine {
	vm := goja.New()

	// 注册常用的字符串方法
	vm.Set("replace", func(call goja.FunctionCall) goja.Value {
		str := call.Argument(0).String()
		from := call.Argument(1).String()
		to := call.Argument(2).String()
		result := strings.Replace(str, from, to, 1)
		return vm.ToValue(result)
	})

	vm.Set("replaceAll", func(call goja.FunctionCall) goja.Value {
		str := call.Argument(0).String()
		from := call.Argument(1).String()
		to := call.Argument(2).String()
		result := strings.ReplaceAll(str, from, to)
		return vm.ToValue(result)
	})

	// 注册更多字符串处理方法
	vm.Set("trim", func(call goja.FunctionCall) goja.Value {
		str := call.Argument(0).String()
		result := strings.TrimSpace(str)
		return vm.ToValue(result)
	})

	vm.Set("split", func(call goja.FunctionCall) goja.Value {
		str := call.Argument(0).String()
		separator := call.Argument(1).String()
		parts := strings.Split(str, separator)
		return vm.ToValue(parts)
	})

	return &JsEngine{vm: vm}
}

// Call 执行JavaScript代码处理输入
func (j *JsEngine) Call(jsCode string, input string) (string, error) {
	// 如果JavaScript代码为空，直接返回原始输入
	if jsCode == "" {
		return input, nil
	}

	// 构造函数代码
	funcCode := "function func(r) { " + jsCode + "; return r; }"

	// 执行函数定义
	_, err := j.vm.RunString(funcCode)
	if err != nil {
		return input, err
	}

	// 调用函数
	fn, ok := goja.AssertFunction(j.vm.Get("func"))
	if !ok {
		return input, nil // 如果不是函数，返回原始输入
	}

	// 执行函数并获取结果
	result, err := fn(goja.Undefined(), j.vm.ToValue(input))
	if err != nil {
		return input, err
	}

	return result.String(), nil
}

// GlobalJsEngine 全局JavaScript引擎实例
var GlobalJsEngine *JsEngine

func init() {
	GlobalJsEngine = NewJsEngine()
}

// CallJs 全局JavaScript调用函数
func CallJs(jsCode string, input string) (string, error) {
	return GlobalJsEngine.Call(jsCode, input)
}
