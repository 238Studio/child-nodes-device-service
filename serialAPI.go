package device

import (
	"errors"

	_const "github.com/238Studio/child-nodes-assist/const"
	"github.com/238Studio/child-nodes-assist/util"
)

// PortLen todo
const PortLen = 512

// 注意 是bytes

// Uint32ToBytes uint32->bytes
// 传入：uint32
// 传出：byte[4]
func Uint32ToBytes(num uint32) []byte {
	numB := make([]byte, 4)
	numB[3] = uint8(num)
	numB[2] = uint8(num >> 8)
	numB[1] = uint8(num >> 16)
	numB[0] = uint8(num >> 24)
	return numB
}

// BytesToUint32 bytes->uint32
// 传入：4位bytes
// 传出：uint32
func BytesToUint32(bytes []byte) uint32 {
	out := uint32(0)
	out |= uint32(bytes[0])
	out = out << 8
	out |= uint32(bytes[1])
	out = out << 8
	out |= uint32(bytes[2])
	out = out << 8
	out |= uint32(bytes[3])
	return out
}

// 通过串口发送数据给单个下位机 根据对应的模块功能
// 传入：下位机的模块ID
// 传出：无
func (app *SerialApp) send(channel *SerialChannel, targetModuleID uint32, targetFunction string, data *[]byte) error {
	app.mu.Lock()
	defer app.mu.Unlock()
	devices, ok := app.serialDevicesBySubModuleID[targetModuleID]
	if !ok {
		return util.NewError(_const.CommonException, _const.Device, errors.New("map key not exist"))
	}
	// 没有对应模块 则直接返回 且向上层抛出错误
	for device_ := range devices {
		device := devices[device_]
		app.readyToSendToDevice(channel, targetModuleID, targetFunction, device.COM, data)
	}

	return nil
}

// 预备发送数据到指定端口的下位机
// 传入：目标模块ID,目标功能，COM，数据
// 传出：无
func (app *SerialApp) readyToSendToDevice(channel *SerialChannel, targetModuleID uint32, targetFunction string, COM string, data *[]byte) {
	data_ := make([]byte, 0)
	data_ = append(data_, Uint32ToBytes(targetModuleID)...)
	data_ = append(data_, []byte(targetFunction)...)
	data_ = append(data_, *data...)
	app.sendBuffer.registerSendData(COM, channel, &data_)
}

// 发送数据给下位机
// 传入：COM口，数据
// 传出：无
func (app *SerialApp) sendToDevice(COM string, data *[]byte) {

}

// StartSendMessage 监听管道讯息 把准备发送的讯息发送到下位机
// 传入：无
// 传出：无
func (serialChannel *SerialChannel) StartSendMessage() {
	for {
		select {
		case data := <-serialChannel.sendDataChannel:
			// 如果出错 则录入错误数据库
			err := serialChannel.app.send(serialChannel, data.targetModuleID, data.targetFunction, &data.data)
			if err != nil {
				// todo:err
			}
		case <-serialChannel.stopSendDataChannel:
			break
		}
	}
}

// StopListenMessage 终止对单个下位机的传入数据的监听
// 传入：COM
// 传出：无
func (app *SerialApp) StopListenMessage(COM string) {
	app.stopListenSubMessageChannel[COM] <- struct{}{}
}

// StopAllListenMessage 终止对所有下位机的传入数据的监听
// 传入：无
// 传出：无
func (app *SerialApp) StopAllListenMessage() {
	for COM, _ := range app.serialDevicesByCOM {
		app.StopListenMessage(COM)
	}
}

// StartListenMessage 开始监听指定的下位机传入数据
// 传入：COM
// 传出：无
func (app *SerialApp) StartListenMessage(COM string) {
	go func() {
		err := app.ListenMessagePerDevice(COM)
		if err != nil {
			//TODO:err
		}
	}()
	//如果失败则向上抛出错误
}

// StartAllListenMessage 监听下位机传入数据 把下位机内的数据传递到指定模块
// 传入：无
// 传出：无
// TODO:有待讨论的
func (app *SerialApp) StartAllListenMessage() *[]error {
	errs := make([]error, 0)
	for COM, _ := range app.serialDevicesByCOM {
		COM := COM
		go func() {
			err := app.ListenMessagePerDevice(COM)
			if err != nil {
				errs = append(errs, err)
			}
		}()
		//如果出错则返回给调用函数
	}
	return &errs
}

// ListenMessagePerDevice 监听单个下位机传入的原始讯息 并在分析后传递到指定模块
// 传入：下位机COM口
// 传出：无
func (app *SerialApp) ListenMessagePerDevice(COM string) error {

}
