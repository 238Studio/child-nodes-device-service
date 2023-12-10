package device_test

import (
	device "github.com/238Studio/child-nodes-device-service"
	"testing"
	"time"
)

func TestName(t *testing.T) {
	Baud := 9600
	serialApp := device.InitSerialApp(Baud, time.Second, 3, 1000, 1000)
	serialApp.AutoInitAllDevices()
	serialApp.StartAutoInit()
	serialApp.StartAllListenMessage()
	println("结束")
	ch := make(chan struct{})
	<-ch
}
