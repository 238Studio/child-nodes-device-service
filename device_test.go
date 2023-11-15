package device_test

import (
	"testing"
	"time"

	device "github.com/238Studio/child-nodes-device-service"
)

func TestName(t *testing.T) {
	app := device.InitSerialApp(9600, time.Millisecond, 16)
	println(app)

}
