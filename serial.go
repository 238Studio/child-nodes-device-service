package device

import (
	"errors"
	"time"

	_const "github.com/238Studio/child-nodes-assist/const"
	"github.com/238Studio/child-nodes-assist/util"
)

// ParseDataToSerialMessage 将纯数据转为数据
// 传入：*byte[]
// 传出：*SerialMessage
func ParseDataToSerialMessage(data *[]byte) *SerialMessage {
	//todo
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
	for device_ := range *devices {
		device := (*devices)[device_]
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
	id := app.sendBuffer.RegisterSendData(COM, channel, &data_)
	app.sendBuffer.ReadySend(COM, channel, id)
}

// 发送数据给下位机
// 传入：COM口，数据，尝试次数
// 传出：无

func (app *SerialApp) sendToDevice(COM string, data *[]byte, times int) error {
	var buffer0 []byte = make([]byte, 4)
	device := app.serialDevicesByCOM[COM]
	// 向串口写入
	_, err := device.portIO.Write(*data)
	if err != nil {
		return err
		//todo
	}
	// 等待串口返回确认数据报 超时则报错
	// 是否收到
	isRev := false
	isReading := true
	go func() {
		time.Sleep(app.ConfirmTimeout)
		if !isRev {
			err = util.NewError(_const.CommonException, _const.Device, errors.New("SerialTimeOut"))
			// 中断读取
			//todo : 是否是堵塞的
			isReading = false
		}
	}()
	for isReading {
		_, err_ := device.portIO.Read(buffer0)
		if err_ != nil {
			return err
			//todo
		}
	}
	isRev = true
	if BytesToUint32(buffer0) != _const.SuccessRev {
		if times > app.maxResendTimes {
			return util.NewError(_const.CommonException, _const.Device, errors.New("ResendFailedOverMaxTimes"))
		}
		err := app.sendToDevice(COM, data, times+1)
		if err != nil {
			return err
		}
	}
	return err
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
	app.revBuffer.revFuncStopChannels[COM] <- struct{}{}
}

// StopAllListenMessage 终止对所有下位机的传入数据的监听
// 传入：无
// 传出：无
func (app *SerialApp) StopAllListenMessage() {
	for COM, _ := range app.serialDevicesByCOM {
		app.StopListenMessage(COM)
	}
}

// StartAllListenMessage 监听下位机传入数据 把下位机内的数据传递到指定模块
// 传入：无
// 传出：无
func (app *SerialApp) StartAllListenMessage() *[]error {
	errs := make([]error, 0)
	for COM, _ := range app.serialDevicesByCOM {
		COM := COM
		go func() {
			err := app.ListenMessagePerDevice(COM, time.Now().UnixMilli())
			if err != nil {
				errs = append(errs, err)
			}
		}()
		//如果出错则返回给调用函数
	}
	return &errs
}

// ListenMessagePerDevice 监听单个下位机传入的原始讯息 并在分析后传递到指定模块
// 同时 其会每隔一段时间就清除过期的buffer 为了避免线程安全问题 这两个功能被放在一个线程以内
// 传入：下位机COM口，上一次清理buffer时间
// 传出：无
func (app *SerialApp) ListenMessagePerDevice(COM string, lastCleanBufferTime int64) error {
	// 从串口读取的缓存
	listenBuffer := make([]byte, _const.PortLen)
	// 一个数据报在之前读取的数据的有效长度
	lastRead := 0
	// 之前读取的数据
	lastBuffer := make([]byte, _const.PortLen)
	// 每次读取都是把上次读取的长度和这次读取的加起来 直到达到portLen
	for {
		select {
		case <-app.revBuffer.revFuncStopChannels[COM]:
			break
		default:
			// 清理超时buffer

			// 读取数据
			read, err := app.serialDevicesByCOM[COM].portIO.Read(listenBuffer)
			if read > 0 {
				lastRead = read
				if err != nil {
					return err
					//todo:处理错误
				}
				// 如果加上这次读取的还是不够一个数据报的长度 则继续读取
				if lastRead+read < int(_const.PortLen) {
					lastRead += read
					lastBuffer = append(lastBuffer[:lastRead], listenBuffer[:read]...)
					// 如果刚好是一个数据报
				} else if lastRead+read == int(_const.PortLen) {
					// 将该数据呈递给缓冲区
					dataBuffer := append(lastBuffer[:lastRead], listenBuffer[:read]...)
					lastRead = 0
					lastBuffer = make([]byte, _const.PortLen)
					data := InitRevDataBuffer(&dataBuffer)
					app.revBuffer.submitDataFrame(COM, data)
				} else {
					// 截断数据 然后提交给缓冲区
					dataBuffer := append(lastBuffer[:lastRead], listenBuffer[:(int)(_const.PortLen)-lastRead]...)
					lastBuffer = append(listenBuffer[((int)(_const.PortLen) - lastRead):read])
					lastRead = (int)(_const.PortLen) - lastRead
					data := InitRevDataBuffer(&dataBuffer)
					app.revBuffer.submitDataFrame(COM, data)
				}
			}
		}
	}
}
