package device

import (
	"errors"
	_const "github.com/238Studio/child-nodes-assist/const"
	"time"

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
	re := (*sendDataBuffer.data)[sendDataBuffer.frameID*(_const.PortLen-17) : (sendDataBuffer.frameID+1)*(_const.PortLen-17)]
	return nil, sendDataBuffer.frameID, &re
}

// 获取某个数据帧的数据
// 传入：数据帧号
// 传出：数据
func (sendDataBuffer *SendDataBuffer) getFrame(frameID uint32) *[]byte {
	re := (*sendDataBuffer.data)[frameID*(_const.PortLen-17) : (sendDataBuffer.frameID+1)*(_const.PortLen-17)]
	return &re
}

// ReadySend 开始发送指定缓存数据块的数据
// 传入：COM号，该数据块的消息通道，数据块号
// 传出：无
func (sendBuffer *SendBuffer) ReadySend(COM string, channel *SerialChannel, bufferID uint32) {
	_, ok := sendBuffer.readySendBuffer[COM]
	if !ok {
		m0 := make(map[uint32]*SendDataBuffer)
		sendBuffer.readySendBuffer[COM] = &m0
	}
	(*sendBuffer.readySendBuffer[COM])[bufferID] = (*sendBuffer.sendBuffer[COM])[bufferID]
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
	(*sendBuffer.sendBuffer[COM])[sendBuffer.i] = &buffer
	if sendBuffer.i > 0xFFE {
		sendBuffer.i = 0
	}
	sendBuffer.i++
	return buffer.bufferID
}

// 呈递数据片段 将刚刚接收到的数据片段呈递给缓冲区 缓冲区会放入数据片段并判断是否可以返回数据片段
// 传入：数据帧
// 传出：无
func (revBuffer *RevBuffer) submitDataFrame(COM string, buffer *RevDataBuffer) error {
	// 进行奇校验
	if !VerifyOddParity(buffer.data) {
		// 要求重发 bufferID frameID
		*revBuffer.app.frameFeedbackChannel.SendDataChannel <- &SerialMessage{
			TargetModuleID: _const.FeedbackModule,
			TargetFunction: _const.WrongOddVariation,
			Data:           append(Uint32ToBytes(buffer.bufferID), Uint32ToBytes(buffer.frameID)...),
		}
	}
	data, ok := (*(revBuffer.revBuffer[COM]))[buffer.bufferID]
	// 如果是新的buffer
	if !ok {
		d := make([]*[]byte, buffer.frameNum)
		// 分配空间
		(*(revBuffer.revBuffer[COM]))[buffer.bufferID] = &d
		// 打上时间戳
		(*(revBuffer.revBufferHangingPeriod[COM]))[buffer.bufferID] = time.Now().UnixMilli()
		// 记录剩余帧数量
		(*(revBuffer.revBufferResidue[COM]))[buffer.bufferID] = buffer.frameNum
	}
	// 放入纯数据
	pureDataLen := (int)(BytesToUint32((*buffer.data)[12:16]))
	pureData := (*buffer.data)[16 : 16+pureDataLen]
	(*data)[buffer.frameID] = &pureData
	// 剩余的--
	(*(revBuffer.revBufferResidue[COM]))[buffer.bufferID]--
	if (*(revBuffer.revBufferResidue[COM]))[buffer.bufferID] == 0 {
		//解析数据并发送到指定管道
		rev := (*(revBuffer.revBuffer[COM]))[buffer.bufferID]
		var revData []byte
		for i := range *rev {
			revData = append(revData, *(*rev)[i]...)
		}
		copy(revData, revData)
		// 将数据发送到指定通道
		message := ParseDataToSerialMessage(&revData)
		channel := revBuffer.app.serialChannelByNodeModulesID[message.TargetModuleID]
		// 开启数据缓冲删除倒计时
		(*revBuffer.revBufferHangingPeriod[COM])[buffer.bufferID] = time.Now().UnixMilli()
		// 将数据发送给需要的模块
		*channel.ReceiveDataChannel <- message
	}
	return nil
}
