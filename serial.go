package device

import (
	"errors"
	"time"

	_const "github.com/238Studio/child-nodes-assist/const"
	"github.com/238Studio/child-nodes-error-manager/errpack"
)

// ParseDataToSerialMessage 将纯数据转为数据
// 传入：*byte[]
// 传出：*SerialMessage
func ParseDataToSerialMessage(data *[]byte) *SerialMessage {
	return nil
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
		return errpack.NewError(errpack.CommonException, errpack.Device, errors.New("map key not exist"))
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
	// 分配数据缓存标号
	data_ := make([]byte, 0)
	data_ = append(data_, Uint32ToBytes(targetModuleID)...)
	data_ = append(data_, []byte(targetFunction)...)
	data_ = append(data_, *data...)
	// 加入发送序列
	id := app.sendBuffer.RegisterSendData(COM, channel, &data_)
	app.sendBuffer.ReadySend(COM, channel, id)
}

// 发送数据给下位机
// 传入：COM口，数据
// 传出：无
func (app *SerialApp) sendToDevice(COM string, data *[]byte) error {
	// 返回状态数据报
	device := app.serialDevicesByCOM[COM]
	// 向串口写入
	_, err := device.portIO.Write(*data)
	if err != nil {
		return errpack.NewError(errpack.CommonException, errpack.Device, errors.New("SendFailed"))
	}
	return err
}

// StartSendMessage 监听管道讯息 把准备发送的讯息发送到下位机
// 传入：moduleID
// 传出：无
func (app *SerialApp) StartSendMessage(moduleID uint32) {
	serialChannel := app.serialChannelByNodeModulesID[moduleID]
	go func() {
		for {
			select {
			case data := <-*serialChannel.SendDataChannel:
				// 如果出错 则录入错误数据库
				err := app.send(serialChannel, data.TargetModuleID, data.TargetFunction, &data.Data)
				if err != nil {
					// todo:err
				}
			case <-*serialChannel.stopSendDataChannel:
				break
			default:
				continue
			}
		}
	}()
}

// StopSendMessage 终止某个SerialChannel的发送
// 传入：moduleID
// 传出：无
func (app *SerialApp) StopSendMessage(moduleID uint32) {
	*app.serialChannelByNodeModulesID[moduleID].stopSendDataChannel <- struct{}{}
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
	listenBuffer := make([]byte, portLen)
	// 一个数据报在之前读取的数据的有效长度
	lastRead := 0
	// 之前读取的数据
	lastBuffer := make([]byte, portLen)
	// 每次读取都是把上次读取的长度和这次读取的加起来 直到达到portLen
	for {
		select {
		case <-app.revBuffer.revFuncStopChannels[COM]:
			err := app.serialDevicesByCOM[COM].portIO.Flush()
			if err != nil {
				return err
			}
			//todo:err
			break
		default:
			nowTime := time.Now().UnixMilli()
			// 清理超时revBuffer
			for bufferID, lastTime := range *app.revBuffer.revBufferHangingPeriod[COM] {
				if (nowTime - lastTime) > app.RevBufferWaitTimeOut {
					delete(*app.revBuffer.revBufferResidue[COM], bufferID)
					delete(*app.revBuffer.revBuffer[COM], bufferID)
					delete(*app.revBuffer.revBufferHangingPeriod[COM], bufferID)
				}
			}
			// 读取串口
			read, err := app.serialDevicesByCOM[COM].portIO.Read(listenBuffer)
			if read > 0 {
				lastRead = read
				if err != nil {
					return err
					//todo:处理错误
				}
				// 如果加上这次读取的还是不够一个数据报的长度 则继续读取
				if lastRead+read < int(portLen) {
					lastRead += read
					lastBuffer = append(lastBuffer[:lastRead], listenBuffer[:read]...)
					// 如果刚好是一个数据报
				} else if lastRead+read == int(_const.PortLen) {
					// 将该数据呈递给缓冲区
					dataBuffer := append(lastBuffer[:lastRead], listenBuffer[:read]...)
					lastRead = 0
					lastBuffer = make([]byte, _const.PortLen)
					data := InitRevDataBuffer(&dataBuffer)
					err := app.revBuffer.submitDataFrame(COM, data)
					if err != nil {
						return err
						//todo:err
					}
				} else {
					// 截断数据 然后提交给缓冲区
					dataBuffer := append(lastBuffer[:lastRead], listenBuffer[:(int)(portLen)-lastRead]...)
					lastBuffer = append(listenBuffer[((int)(portLen) - lastRead):read])
					lastRead = (int)(portLen) - lastRead
					data := InitRevDataBuffer(&dataBuffer)
					err := app.revBuffer.submitDataFrame(COM, data)
					if err != nil {
						return err
						//todo:err
					}
				}
			}
			continue
		}
	}
}

// StartAllSendChannels 开始所有发送线程
// 传入：无
// 传出：无
func (sendBuffer *SendBuffer) StartAllSendChannels() []error {
	var errs = make([]error, 0)
	for COM, _ := range sendBuffer.app.serialDevicesByCOM {
		err := sendBuffer.StartSendChannel(COM)
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

// StopAllSendChannels 取消所有发送线程
// 传入：无
// 传出：无
func (sendBuffer *SendBuffer) StopAllSendChannels() {
	for _, v := range sendBuffer.sendFuncStopChannels {
		*v <- struct{}{}
	}
}

// StartSendChannel 加入一个发送线程 通过COM 并发开始发送 每个发送线程都是发送该线程对应的COM的讯息
// 传入：COM
// 传出：无
func (sendBuffer *SendBuffer) StartSendChannel(COM string) error {
	_, ok := sendBuffer.readySendBuffer[COM]
	if !ok {
		return errpack.NewError(errpack.TrivialException, errpack.Device, errors.New("NoSuchCOM"))
	}
	stopChannel := make(chan struct{})
	*sendBuffer.sendFuncStopChannels[COM] = stopChannel
	go sendBuffer.sendFunc(stopChannel, COM)
	return nil
}

/*
 数据的格式是 数据报编号[32位] 数据报帧号[32位] 数据报总帧数[32位] 数据报实际长度[32位](也就是这个数据报内要截取多少 只包含有效数据的长度)  数据[] 补0 奇校验码[8位] 一帧总长度是固定的
*/
// 发送线程，这个线程会轮转式的，向下位机发送被注册的，需要发送的数据报
// 传入：COM
// 传出：无
func (sendBuffer *SendBuffer) sendFunc(stopChan chan struct{}, COM string) {
	for {
		select {
		case <-stopChan:
			break
		default:
			// 执行删除超时发送的数据报的任务
			nowTime := time.Now().UnixMilli()
			for bufferID, lastTime := range *sendBuffer.sendBufferWaitTime[COM] {
				if nowTime-lastTime > sendBuffer.app.SendBufferWaitTimeOut {
					delete(*sendBuffer.sendBuffer[COM], bufferID)
					delete(*sendBuffer.sendBufferWaitTime[COM], bufferID)
					delete(*sendBuffer.readySendBuffer[COM], bufferID)
				}
			}
			// 执行轮转发送数据片的任务 e
			for _, send := range *sendBuffer.readySendBuffer[COM] {
				// 发送数据帧
				err, frameID, frame := (*send).nextDataFrame()
				err = sendBuffer.app.sending(COM, send, frameID, frame)
				if err != nil {
					return

				}
				//todo:err
			}
			continue
		}
	}
}

// 发送消息数据帧

// 发送数据帧
// 传入：COM string, send *SendDataBuffer, frameID uint32, frame *[]byte
// 传出：error
func (app *SerialApp) sending(COM string, send *SendDataBuffer, frameID uint32, frame *[]byte) error {
	sendFrame := make([]byte, 0)
	// 将数据的各个段的内容加入
	// 加入实际数据长度
	sendFrame = append(Uint32ToBytes((uint32)(len(*frame))), *frame...)
	// 加入总帧数
	sendFrame = append(Uint32ToBytes((*send).frameNum), sendFrame...)
	// 加入帧ID
	sendFrame = append(Uint32ToBytes(frameID), sendFrame...)
	// 加入缓冲ID
	sendFrame = append(Uint32ToBytes((*send).bufferID), sendFrame...)
	// 补零
	zeros := make([]byte, int(portLen)-len(*frame)-1)
	sendFrame = append(sendFrame, zeros...)
	sendFrame = append(sendFrame, CalculateOddParity(&sendFrame))
	err := app.sendToDevice(COM, &sendFrame)
	return err
}

// StopSendChannel 取消一个COM的发送线程 通过COM
// 传入：无
// 传出：无
func (sendBuffer *SendBuffer) StopSendChannel(COM string) {
	*sendBuffer.sendFuncStopChannels[COM] <- struct{}{}
	delete(sendBuffer.sendFuncStopChannels, COM)
}
