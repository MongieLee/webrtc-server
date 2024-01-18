package utils

import (
	"encoding/json"
	"fmt"
)

// mapToJson 将给定的 map[string]interface{} 转换为 JSON 格式的字符串。
// 如果转换成功，则返回 JSON 字符串；如果转换失败，则返回空字符串。
func MapToJson(m map[string]interface{}) string {
	if byt, err := json.Marshal(m); err != nil {
		_ = fmt.Errorf(err.Error())
		return ""
	} else {
		return string(byt)
	}
}

// json字符串转成对应的map结构体
func JsonStrToStruct(jsonData string) (result map[string]interface{}, err error) {
	err = json.Unmarshal([]byte(jsonData), &result)
	if err != nil {
		_ = fmt.Errorf(err.Error())
	}
	return
}
