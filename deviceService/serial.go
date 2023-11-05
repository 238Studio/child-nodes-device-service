package deviceService

import (
	_const "github.com/UniversalRobotDriveTeam/child-nodes-assist/const"
	"github.com/UniversalRobotDriveTeam/child-nodes-assist/util"
)

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
func (app *SerialApp) send(targetModuleID byte, targetFunction string, data *[]byte) error {
	app.mu.Lock()
	defer app.mu.Unlock()
	devices, ok := app.serialDevicesBySubModuleID[targetModuleID]
	if !ok {
		return util.NewError(_const.TrivialException, _const.DeviceNoTargetModuleError, nil)
	}
	// 没有对应模块 则直接返回 且向上层抛出错误
	for device_ := range devices {
		device := devices[device_]
		err := app.sendToDevice(targetModuleID, targetFunction, device.COM, data)
		if err != nil {
			return util.NewError(_const.CommonException, _const.FailedToSendToDeviceError, err)
		}
	}
	return nil
}

/*
 数据的格式是 帧头  报总长度（不包含帧头帧尾） 目标模块编号 目标功能长度 目标功能 数据长度 数据 数据报ID 奇校验位 帧尾
*/
// 发送数据到指定端口的下位机
// 传入：COM，数据
// 传出：无
func (app *SerialApp) sendToDevice(targetModuleID byte, targetFunction string, COM string, data *[]byte) error {
	// 根据COM口获取对应的串口对象
	device := app.serialDevicesByCOM[COM]
	if !device.isConnected {
		return util.NewError(_const.TrivialException, _const.PortNotConnectedError, nil)
		//如果没连上 则返回连接错误
	}
	// 刷新串口 保证之前的数据都发出去了
	err := device.portIO.Flush()
	if err != nil {
		return err
		// IO错误扔回去 让上层重试 如果确实失败则放弃传输 并重新初始化
	}
	// 奇校验码
	verify := byte(0)
	// 准备发送的数据
	out := make([]byte, 0)
	// 模块功能名
	function := []byte(targetFunction)
	// 模块功能名长度
	functionLen := byte(len(function))
	// 按照顺序合并发送数据
	out = append(out, targetModuleID, functionLen)
	out = append(out, function...)
	out = append(out, Uint32ToBytes(uint32(len(*data)))...)
	out = append(out, *data...)
	out = append(Uint32ToBytes(uint32(len(out)+4)), out...)
	out = append([]byte{_const.PortHead}, out...)
	out = append(out, _const.PortEnd)
	// 奇校验码
	x := 0
	for i := range out {
		x += int(out[i])
	}
	if x%2 == 0 {
		verify = 1
	}
	out = append(out, verify)
	// 发送数据
	_, err__ := device.portIO.Write(out)
	if err__ != nil {
		return err
	}
	// IO错误扔回去
	return nil
}

// StartSendMessage 监听管道讯息 把准备发送的讯息发送到下位机
// 传入：无
// 传出：无
func (serialChannel *SerialChannel) StartSendMessage() {
	for {
		select {
		case data := <-serialChannel.sendDataChannel:
			// 如果出错 则录入错误数据库
			err := serialChannel.app.send(data.targetModuleID, data.targetFunction, &data.data)
			if err != nil {
				// todo
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
	app.stopListenSubMessageChannel[COM] <- 0
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

		}
	}()
	//如果失败则向上抛出错误
}

// StartAllListenMessage 监听下位机传入数据 把下位机内的数据传递到指定模块
// 传入：无
// 传出：无
func (app *SerialApp) StartAllListenMessage() *[]error {
	errors := make([]error, 0)
	for COM, _ := range app.serialDevicesByCOM {
		COM := COM
		go func() {
			err := app.ListenMessagePerDevice(COM)
			if err != nil {
				errors = append(errors, err)
			}
		}()
		//如果出错则返回给调用函数
	}
	return &errors
}

// ListenMessagePerDevice 监听单个下位机传入的原始讯息 并在分析后传递到指定模块
// 传入：下位机COM口
// 传出：无
func (app *SerialApp) ListenMessagePerDevice(COM string) error {
	// 上一次缓冲区剩下的部分
	lastBuffer := make([]byte, 0)
	// 缓存区
	buffer := make([]byte, _const.PortMaxLen)
	// 数据包
	data := make([]byte, 0)
	// 此时接收的是否是某个数据包最开始的一系列数据
	isHead := true
	// 当前数据包的大小
	dataLen := uint32(0)
	for {
		select {
		case <-app.stopListenSubMessageChannel[COM]:
			break
		default:
			num, err := app.serialDevicesByCOM[COM].portIO.Read(buffer)
			if err != nil {
				// 如果读取错误 刷新串口内容 将缓存清空
				err := app.serialDevicesByCOM[COM].portIO.Flush()
				if err != nil {
					//todo
					app.StopListenMessage(COM)
					// 如果刷新串口失败则打印错误并关闭消息监听
				}
				// 向下位机发送重发的要求 并直返回 等待重发的消息
				d := make([]byte, 0)
				err1 := app.sendToDevice(_const.SerialVerify, _const.FailedToRev, COM, &d)
				if err1 != nil {
					//todo
					app.StopListenMessage(COM)
				}
				continue
			}
			// 这个错误会打断数据流的读取 如果发生了这个读取错误 则向下位机要求重发 且抛弃缓存进行重发
			// 如果接收到的数据长度大于零 则拼接数据包 当数据包达到指定长度的时候 完成拼接 并进行数据分析 传递到对应的管道
			if num > 0 {
				data = append(data, buffer[:num]...)
			}
			if isHead {
				// 如果头错误 则要求重发 并直接返回 等待重发的消息
				dataLen = BytesToUint32(buffer[1:5])
				isHead = false
				data = append(lastBuffer, data...)
			}
			// 如果达到数据长度则传出报告
			if uint32(len(data)) > (dataLen + 2) {
				// 重新回到报头
				isHead = true
				// 报尾
				end := data[dataLen+1]
				// 数据
				data_ := data[1:dataLen]
				lastBuffer = data[dataLen+2:]
				if end != _const.PortEnd {
					// 通知下位机重发
					d := make([]byte, 0)
					err1 := app.sendToDevice(_const.SerialVerify, _const.FailedToRev, COM, &d)
					if err1 != nil {
						//todo
						app.StopListenMessage(COM)
					}
					continue
				}
				// 进行奇校验 如果奇校验没有通过 则要求重发
				x := 0
				for _, d := range data {
					x += int(d)
				}
				if x%2 == 0 {
					// 向下位机发送重发的要求 并直返回 等待重发的消息
					d := make([]byte, 0)
					err1 := app.sendToDevice(_const.SerialVerify, _const.FailedToRev, COM, &d)
					if err1 != nil {
						//todo
						app.StopListenMessage(COM)
					}
					continue
				}
				message := new(SerialMessage)
				message.targetModuleID = data_[4]
				functionLen := data_[5]
				// 如果是SerialVerify 则通知重发
				if data_[4] == _const.SerialVerify {
					id := int(data_[6])
					serialMessage := app.dataCache[COM][id]
					err := app.send(serialMessage.targetModuleID, serialMessage.targetFunction, &serialMessage.data)
					if err != nil {
						return err
					}
					//todo
				}
				message.targetFunction = string(data_[6 : 6+functionLen])
				dataOut := data_[6+functionLen:]
				message.data = dataOut
				(app.serialChannelByNodeModulesID[message.targetModuleID]).receiveDataChannel <- message
			}
		}
	}
	//todo:err nil
}

// 在发送数据缓存中加入数据 这个数组会抛弃最后一个数据 并返回一个目前发送数据的ID
// 传入：数据指针
// 传出：无
func (app *SerialApp) putDataToCache(COM string, message *SerialMessage) int {
	defer func() {
		if app.dataCacheIDNow == (len(app.dataCache) - 1) {
			app.dataCacheIDNow = 0
		} else {
			app.dataCacheIDNow++
		}
	}()
	app.dataCache[COM][app.dataCacheIDNow] = message
	return app.dataCacheIDNow
}

// 根据ID提取一个发送数据缓存中的数据
// 传入：数据ID
// 传出：数据指针
func (app *SerialApp) getDataFromCache(COM string, messageID int) *SerialMessage {
	return app.dataCache[COM][messageID]
}
