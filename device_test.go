package device_test

import (
	"fmt"
	"testing"
)

func TestName(t *testing.T) {
	bytes := []byte{1, 2, 3, 4}
	bytes1 := make([]byte, 0)
	bytes1 = append(bytes1, bytes...)
	copy(bytes, bytes)
	bytes1[1] = 43
	fmt.Printf("%+v", bytes)
}
