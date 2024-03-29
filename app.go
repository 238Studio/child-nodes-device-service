package device

import (
	_const "github.com/238Studio/child-nodes-assist/const"
	"github.com/tarm/serial"
	"time"
)

// PutDeviceIntoSerialApp 将一个下位机注册到串口应用中 实现从COM口到串口设备的映射
// 传入：下位机
// 传出：无
func (app *SerialApp) PutDeviceIntoSerialApp(device *SerialDevice) {
	app.serialDevicesByCOM[device.COM] = device
	m := make(map[uint32]int64)
	app.revBuffer.revBufferHangingPeriod[device.COM] = &m
}

// RemoveDeviceFromSerialApp 将一个硬件从串口设备中移除
// 传入：硬件COM
// 传出：无
func (app *SerialApp) RemoveDeviceFromSerialApp(COM string) {
	delete(app.serialDevicesByCOM, COM)
	app.DeregisterSubModulesWithDevice(COM)
}

// OpenPort 打开某个硬件的端口
// 传入：该硬件的COM口
// 传出：无
func (app *SerialApp) OpenPort(COM string) error {
	portIO, err := serial.OpenPort(&app.serialDevicesByCOM[COM].serialConfig)
	if err != nil {
		return err
	}
	app.serialDevicesByCOM[COM].portIO = portIO
	app.serialDevicesByCOM[COM].isConnected = true

	return nil
}

// ClosePort 关闭某个硬件的端口
// 传入：该硬件COM口
// 传出：无
func (app *SerialApp) ClosePort(COM string) error {
	if app.serialDevicesByCOM[COM].isConnected {
		err := app.serialDevicesByCOM[COM].portIO.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

// RegisterSubModulesWithDevice 注册下位机关联模块 下位机功能模块->下位机集合 实现映射
// 传入：关联模块moduleID，下位机COM
// 传出：无
func (app *SerialApp) RegisterSubModulesWithDevice(moduleID []uint32, COM string) {
	for moduleID_ := range moduleID {
		_, ok := app.serialDevicesBySubModuleID[moduleID[moduleID_]]
		if !ok {
			k := make(map[string]*SerialDevice)
			app.serialDevicesBySubModuleID[moduleID[moduleID_]] = &k
		}
		(*app.serialDevicesBySubModuleID[moduleID[moduleID_]])[COM] = app.serialDevicesByCOM[COM]

	}
}

// DeregisterSubModulesWithDevice 取消注册下位机关联模块
// 传入：下位机COM
// 传出：无
func (app *SerialApp) DeregisterSubModulesWithDevice(COM string) {
	for device := range app.serialDevicesBySubModuleID {
		_, ok := (*app.serialDevicesBySubModuleID[device])[COM]
		if ok {
			delete(*app.serialDevicesBySubModuleID[device], COM)
		}
	}
}

// GetSerialMessageChannel 获取并注册子节点消息通道
// 传入：子节点模块ID
// 传出：串口消息通道
func (app *SerialApp) GetSerialMessageChannel(nodeModuleID uint32) *SerialChannel {
	channel := new(SerialChannel)
	c0 := make(chan *SerialMessage, 1)
	channel.ReceiveDataChannel = &c0
	c1 := make(chan *SerialMessage, 1)
	channel.SendDataChannel = &c1
	c2 := make(chan struct{})
	channel.stopSendDataChannel = &c2
	app.serialChannelByNodeModulesID[nodeModuleID] = channel
	return channel
}

// RemoveSerialChannel 取消注册一个消息通道
// 传入：子节点模块ID
// 传出：无
func (app *SerialApp) RemoveSerialChannel(nodeModuleID uint32) {
	delete(app.serialChannelByNodeModulesID, nodeModuleID)
}

// StartAutoResend 开启自动重传
// 传入：无
// 传出：无
func (app *SerialApp) StartAutoResend() {
	app.GetSerialMessageChannel(_const.FeedbackModule)
	go app.resend()
}

// StopAutoResend 关闭自动重传
// 传入：无
// 传出：无
func (app *SerialApp) StopAutoResend() {
	app.StopSendMessage(_const.FeedbackModule)

}

// 重发数据
// 传入：无
// 传出：无
func (app *SerialApp) resend() {
	for {
		select {
		case <-*app.frameFeedbackChannel.stopSendDataChannel:
			break
		case msg := <-*app.frameFeedbackChannel.ReceiveDataChannel:
			// 接收到下位机重发的数据
			COM_ := msg.Data[8]
			COM := "COM" + string(COM_)
			bufferID := BytesToUint32(msg.Data[:4])
			frameID := BytesToUint32(msg.Data[4:8])
			if msg.TargetFunction == _const.ReSendData {
				reData := msg.Data[9:]
				// 载入数据并更新销毁时间戳
				(*app.revBuffer.revBufferResidue[COM])[bufferID]--
				(*app.revBuffer.revBufferHangingPeriod[COM])[bufferID] = time.Now().UnixMilli()
				(*(*app.revBuffer.revBuffer[COM])[bufferID])[frameID] = &reData
				continue
			}
			//收到下位机的重发通知
			resendFrame := (*app.sendBuffer.readySendBuffer[COM])[bufferID]
			d := *resendFrame.getFrame(frameID)
			d = append(Uint32ToBytes(frameID), d...)
			d = append(Uint32ToBytes(bufferID), d...)
			*app.frameFeedbackChannel.SendDataChannel <- &SerialMessage{
				TargetModuleID: _const.FeedbackModule,
				TargetFunction: _const.ReSendData,
				Data:           d,
			}
		default:
			continue
		}
	}
}
