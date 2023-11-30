package device

import (
	"errors"
	_const "github.com/238Studio/child-nodes-assist/const"
	"github.com/238Studio/child-nodes-assist/util"

	"math"
)

// 获取下一个数据块以及其id，如果不存在下一个数据块则返回error
// 传入：无
// 传出：无
func (sendDataBuffer *SendDataBuffer) nextDataFrame() (err error, frameID uint32, dataFrame *[]byte) {
	// 如果到了最后一个数据帧 则返回错误
	if sendDataBuffer.frameID >= sendDataBuffer.frameNum {
		return errors.New(""), 0, nil
	}
	defer func() { sendDataBuffer.bufferID++ }()
	re := (*sendDataBuffer.data)[sendDataBuffer.frameID*(_const.PortLen-17) : (sendDataBuffer.bufferID+1)*(_const.PortLen-17)]
	return nil, sendDataBuffer.frameID, &re
}

// ReleaseSendData 释放发送缓存数据块，注意，需要先停止再释放
// 传入：COM号，该数据块的消息通道，数据块号
// 传出：无
func (sendBuffer *SendBuffer) ReleaseSendData(COM string, channel *SerialChannel, bufferID uint32) {
	delete(*(*sendBuffer.sendBuffer[COM])[channel], bufferID)
}

// ReadySend 开始发送指定缓存数据块的数据
// 传入：COM号，该数据块的消息通道，数据块号
// 传出：无
func (sendBuffer *SendBuffer) ReadySend(COM string, channel *SerialChannel, bufferID uint32) {
	_, ok := sendBuffer.readySendBuffer[COM]
	if !ok {
		m0 := make(map[*SerialChannel]*map[uint32]*SendDataBuffer)
		sendBuffer.readySendBuffer[COM] = &m0
	}
	_, ok_ := (*sendBuffer.readySendBuffer[COM])[channel]
	if !ok_ {
		m1 := make(map[uint32]*SendDataBuffer)
		(*sendBuffer.readySendBuffer[COM])[channel] = &m1
	}
	(*(*sendBuffer.readySendBuffer[COM])[channel])[bufferID] = (*(*sendBuffer.sendBuffer[COM])[channel])[bufferID]
}

// StopSend 停止发送指定数据块的数据
// 传入：COM号，该数据块的消息通道，数据块号
// 传出：无
func (sendBuffer *SendBuffer) StopSend(COM string, channel *SerialChannel, bufferID uint32) {
	delete(*(*sendBuffer.readySendBuffer[COM])[channel], bufferID)
}

// RegisterSendData 生成并注册缓冲数据块
// 传入：需要发送的数据
// 传出：数据块号
func (sendBuffer *SendBuffer) RegisterSendData(COM string, channel *SerialChannel, data *[]byte) uint32 {
	buffer := SendDataBuffer{
		data:     data,
		frameID:  0,
		bufferID: sendBuffer.i,
		frameNum: uint32(math.Ceil(float64(uint32(len(*data)) / _const.PortLen))),
	}
	(*(*sendBuffer.sendBuffer[COM])[channel])[sendBuffer.i] = &buffer
	if sendBuffer.i > 0xFFFFFF {
		sendBuffer.i = 0
	}
	sendBuffer.i++
	return buffer.bufferID
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
		v <- struct{}{}
	}
}

// StartSendChannel 加入一个发送线程 通过COM
// 传入：COM
// 传出：无
func (sendBuffer *SendBuffer) StartSendChannel(COM string) error {
	_, ok := sendBuffer.readySendBuffer[COM]
	if !ok {
		return util.NewError(_const.TrivialException, _const.Device, errors.New("NoSuchCOM"))
	}
	stopChannel := make(chan struct{})
	sendBuffer.sendFuncStopChannels[COM] = stopChannel
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
			// 执行轮转发送数据片的任务 e
			for channel, v := range *sendBuffer.readySendBuffer[COM] {
				for _, send := range *v {
					err, frameID, frame := (*send).nextDataFrame()
					//发送完毕后清理缓存
					if err != nil {
						delete(*v, frameID)
						delete(*(*sendBuffer.sendBuffer[COM])[channel], frameID)
					}
					// 加入
					*frame = append(Uint32ToBytes((uint32)(len(*frame))), *frame...)
					*frame = append(Uint32ToBytes((*send).frameNum), *frame...)
					*frame = append(Uint32ToBytes(frameID), *frame...)
					*frame = append(Uint32ToBytes((*send).bufferID), *frame...)
					zeros := make([]byte, int(_const.PortLen)-len(*frame)-1)
					*frame = append(*frame, zeros...)
					*frame = append(*frame, CalculateOddParity(frame))
					err_ := sendBuffer.app.sendToDevice(COM, frame, 0)
					if err != nil {
						//todo:err
						println(err_)
					}
				}
			}
		}
	}
}

// StopSendChannel 取消一个发送线程 通过COM
// 传入：无
// 传出：无
func (sendBuffer *SendBuffer) StopSendChannel(COM string) {
	sendBuffer.sendFuncStopChannels[COM] <- struct{}{}
	delete(sendBuffer.sendFuncStopChannels, COM)
}
