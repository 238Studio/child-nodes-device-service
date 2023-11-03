package deviceService

import "github.com/tarm/serial"

// PutDeviceIntoSerialApp 将一个下位机注册到串口应用中 实现从COM口到串口设备的映射
// 传入：下位机
// 传出：无
func (app *SerialApp) PutDeviceIntoSerialApp(device *SerialDevice) {
	app.serialDevicesByCOM[device.COM] = device
}

// RemoveDeviceFromSerialApp 将一个硬件从串口设备中移除
// 传入：硬件COM
// 传出：无
func (app *SerialApp) RemoveDeviceFromSerialApp(COM string) {
	delete(app.serialDevicesByCOM, COM)
}

// OpenPort 打开某个硬件的端口
// 传入：该硬件的COM口
// 传出：无
func (app *SerialApp) OpenPort(COM string) {
	portIO, err := serial.OpenPort(&app.serialDevicesByCOM[COM].serialConfig)
	if err != nil {
		//TODO:ERR
	}
	app.serialDevicesByCOM[COM].portIO = portIO
	app.serialDevicesByCOM[COM].isConnected = true
}

// ClosePort 关闭某个硬件的端口
// 传入：该硬件COM口
// 传出：无
func (app *SerialApp) ClosePort(COM string) {
	if app.serialDevicesByCOM[COM].isConnected {
		err := app.serialDevicesByCOM[COM].portIO.Close()
		if err != nil {
			//TODO:ERR
			return
		}
	}
}

// RegisterSubModulesWithDevice 注册下位机关联模块 下位机功能模块->下位机集合 实现映射
// 传入：关联模块moduleID，下位机COM
// 传出：无
func (app *SerialApp) RegisterSubModulesWithDevice(moduleID []byte, COM string) {
	for moduleID_ := range moduleID {
		_, ok := app.serialDevicesBySubModuleID[moduleID[moduleID_]]
		if !ok {
			app.serialDevicesBySubModuleID[moduleID[moduleID_]] = make(map[string]*SerialDevice)
		}
		app.serialDevicesBySubModuleID[moduleID[moduleID_]][COM] = app.serialDevicesByCOM[COM]
	}
}

// DeregisterSubModulesWithDevice 取消注册下位机关联模块
// 传入：下位机COM
// 传出：无
func (app *SerialApp) DeregisterSubModulesWithDevice(COM string) {
	for device := range app.serialDevicesBySubModuleID {
		_, ok := app.serialDevicesBySubModuleID[device][COM]
		if ok {
			delete(app.serialDevicesBySubModuleID[device], COM)
		}
	}
}

// GetSerialMessageChannel 获取并注册消息通道
// 传入：子节点模块ID
// 传出：串口消息通道
func (app *SerialApp) GetSerialMessageChannel(nodeModuleID byte) *SerialChannel {
	channel := new(SerialChannel)
	channel.app = app
	channel.receiveDataChannel = make(chan *SerialMessage, 1)
	channel.sendDataChannel = make(chan *SerialMessage, 1)
	channel.stopSendDataChannel = make(chan int, 1)
	app.serialChannelByNodeModulesID[nodeModuleID] = channel
	return channel
}

// RemoveSerialChannel 取消注册一个消息通道
// 传入：子节点模块ID
// 传出：无
func (app *SerialApp) RemoveSerialChannel(nodeModuleID byte) {
	delete(app.serialChannelByNodeModulesID, nodeModuleID)
}
