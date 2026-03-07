package engine

import "strings"

// RenderPrompt 渲染提示词模板，将 {{变量名}} 替换为 variables 中的值
// 未匹配的变量保留原样
func RenderPrompt(template string, variables map[string]string) string {
	if len(variables) == 0 {
		return template
	}
	result := template
	for key, value := range variables {
		result = strings.ReplaceAll(result, "{{"+key+"}}", value)
	}
	return result
}
