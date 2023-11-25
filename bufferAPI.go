package device

import (
	"errors"
	"math"
)

/*
 数据的格式是 数据报编号[32位] 数据报帧号[32位] 数据报实际长度[32位] 奇校验码[8位] 数据[] 一帧总长度是固定的
*/
// 获取下一个数据块以及其id，如果不存在下一个数据块则返回error
// 传入：无
// 传出：无
func (sendDataBuffer *SendDataBuffer) nextDataFrame() (err error, frameID uint32, dataFrame *[]byte) {
	// 如果到了最后一个数据帧 则返回错误
	if sendDataBuffer.frameID >= sendDataBuffer.frameNum {
		return errors.New(""), 0, nil
	}
	defer func() { sendDataBuffer.bufferID++ }()
	re := (*sendDataBuffer.data)[sendDataBuffer.frameID*PortLen : (sendDataBuffer.bufferID+1)*PortLen]
	return nil, sendDataBuffer.frameID, &re
}

// 释放发送缓存数据块
// 传入：COM号，该数据块的消息通道，数据块号
// 传出：无
func (sendBuffer *SendBuffer) releaseSendData(COM string, channel *SerialChannel, bufferID uint32) {
	delete(sendBuffer.sendBuffer[COM][channel], bufferID)
}

// 生成并注册缓冲数据块
// 传入：需要发送的数据
// 传出：发送数据缓存块，数据块号
func (sendBuffer *SendBuffer) registerSendData(COM string, channel *SerialChannel, data *[]byte) {
	buffer := SendDataBuffer{
		data:     data,
		frameID:  0,
		bufferID: sendBuffer.i,
		frameNum: uint32(math.Ceil(float64(len(*data) / PortLen))),
	}
	sendBuffer.sendBuffer[COM][channel][sendBuffer.i] = &buffer
	if sendBuffer.i > 0xFFFFFF {
		sendBuffer.i = 0
	}
	sendBuffer.i++
}
