package response

// Result 统一结果返回
type Result struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

func NewResult(code int, message string, data interface{}) *Result {
	return &Result{
		Code:    code,
		Message: message,
		Data:    data,
	}
}

// Success 返回带数据的成功结果
func Success(data interface{}) *Result {
	return NewResult(1, "success", data)
}

// SuccessWithoutData 返回不带数据的成功结果
func SuccessWithoutData() *Result {
	return NewResult(1, "success", nil)
}

// Error 返回带错误信息的失败结果
func Error(message string) *Result {
	return NewResult(0, message, nil)
}
