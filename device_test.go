package device_test

import (
	_const "github.com/238Studio/child-nodes-assist/const"
	device "github.com/238Studio/child-nodes-device-service"
	"testing"
	"time"
)

func TestName(t *testing.T) {
	// 测试硬件连接器
	println("硬件连接器")
	Baud := 9600
	serialApp := device.InitSerialApp(Baud, time.Second, 3, 1000, 1000)
	serialApp.AutoInitAllDevices()
	serialApp.StartAutoInit()
	serialApp.StartAllListenMessage()
	ch := make(chan struct{})
	virtualDevice := device.InitSerialApp(Baud, time.Second, 3, 1000, 1000)
	virtualDevice.StartAllListenMessage()
	initModule := virtualDevice.GetSerialMessageChannel(_const.InitModule)
	virtualDevice.StartSendMessage(_const.InitModule)
	var testData []byte
	testData=append([]byte{5},testData...)
	msg := device.SerialMessage{
		TargetModuleID: _const.InitModule,
		TargetFunction: "InitData",
		Data:           testData,
	}
	*initModule.SendDataChannel <- &msg
	<-ch
}
