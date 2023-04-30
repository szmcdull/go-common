package common

// NewJsonArray 生成用于序列化JSON数组的map
// NewJsonArray() 生成长度为0的数组
// NewJsonArray(100) 生成长度为100的数组
// NewJsonArray(0, 100) 生成长度为0，capacity为100的数组
func NewJsonArray(length ...int) []interface{} {
	switch len(length) {
	case 0:
		return make([]interface{}, 0)
	case 1:
		return make([]interface{}, length[0])
	default:
		return make([]interface{}, length[0], length[1])
	}
}

// NewJsonObject 生成用于序列化JSON的对象map
func NewJsonObject() map[string]interface{} {
	return make(map[string]interface{})
}
