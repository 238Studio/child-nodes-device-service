package device_test

import (
	"testing"
)

func TestName(t *testing.T) {
	bytes := []byte{1, 2, 3, 2, 34, 0}
	bytes = append(bytes, CalculateOddParity(&bytes))
	println(VerifyOddParity(&bytes))
}

// VerifyOddParity 验证奇校验数
// 传入：数据
// 传出：是否通过奇校验
func VerifyOddParity(data *[]byte) bool {
	parity := (*data)[len(*data)-1]
	// 计算数据中包含的 "1" 的数量
	countOnes := 0
	for _, b := range (*data)[:len(*data)-1] {
		// 使用位运算检查每个字节中包含的 "1" 的数量
		for i := 0; i < 8; i++ {
			countOnes += int((b >> uint(i)) & 1)
		}
	}

	// 判断奇偶性并验证奇校验位
	return (countOnes%2 == 1 && parity == 1) || (countOnes%2 == 0 && parity == 0)
}

// CalculateOddParity 获得奇校验数
// 传入：数据
// 传出：奇校验数
func CalculateOddParity(data *[]byte) byte {
	// 计算数据中包含的 "1" 的数量
	countOnes := 0
	for _, b := range *data {
		// 使用位运算检查每个字节中包含的 "1" 的数量
		for i := 0; i < 8; i++ {
			countOnes += int((b >> uint(i)) & 1)
		}
	}
	// 判断奇偶性并返回校验位
	if countOnes%2 == 1 {
		return 1
	}
	return 0
}
