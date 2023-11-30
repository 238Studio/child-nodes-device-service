package device

import (
	"sync"
	"time"

	"github.com/tarm/serial"
)

// 子节点模块

// SerialDevice 对应了单个下位机的串口收发类
type SerialDevice struct {
	// 该串口配置
	serialConfig serial.Config
	// 串口号
	COM string
	// 串口通讯
	portIO *serial.Port
	// 该串口对应的下位机功能模块（注意 不是实际模块 而是注册的功能模块） moduleID
	SubModuleID []uint32
	// 是否处于连接状态
	isConnected bool
}

// SerialApp 容纳串口操作和信息的应用
type SerialApp struct {
	// 波特率
	Baud int
	// 确认回报等待时间
	ConfirmTimeout time.Duration
	// 串口消息等待时间
	ReadTimeout time.Duration
	// 互斥锁
	mu *sync.Mutex
	// 从下位机的模块对应了若干个下位机的串口收发模块 NodeModuleID->SerialAppPerDevice
	serialDevicesBySubModuleID map[uint32]map[string]*SerialDevice
	// COM->SerialAppPerDevice
	serialDevicesByCOM map[string]*SerialDevice
	// 从子节点功能模块对应到SerialChannel
	serialChannelByNodeModulesID map[uint32]*SerialChannel
	// 中止接收下位机传来信息的通道 COM->channel
	stopListenSubMessageChannel map[string]chan struct{}
	// 是否运行
	isAlive bool
	// 发送缓存
	sendBuffer *SendBuffer
	// 最大发送尝试次数
	maxResendTimes int
}

// InitSerialDataProcessor 初始化模块的数据转换器
type InitSerialDataProcessor struct {
	app          *SerialApp
	rightDevices []string
}

// InitSerialModuleApp 初始化模块
type InitSerialModuleApp struct {
	// 数据处理器
	dataProcessor *InitSerialDataProcessor
	// 消息通道
	channel *SerialChannel
	// 从下位机的功能模块对应了若干个下位机的串口收发模块 NodeModuleID->SerialAppPerDevice
	serialDevicesBySubModuleID map[byte]map[string]*SerialDevice
}

// SerialMessage 串口讯息
type SerialMessage struct {
	// 目标模块ID
	/* 目标模块 指的是功能模块 如果是上位机发送给下位机 那么指向的就是下位机的指定功能模块 反之就是上位机的指定功能模块
	下位机的功能模块 比较多样且可以自定义 例如底盘驱动 云台 或者发射器等 而上位机的功能模块则只有少数几个
	包括了 传感器反馈，错误通道，数据报告，初始化4个 也就是分别是传感器反馈的数据 底层出现软件错误汇报的错误 以及报告底层软件状态的数据报告
	*/
	targetModuleID uint32
	// 目标模块功能
	targetFunction string
	// 数据
	data []byte
}

// SerialChannel 和串口进行交互的对象
type SerialChannel struct {
	// 串口应用
	app *SerialApp
	// 模块从下位机收到数据的通道
	receiveDataChannel chan *SerialMessage
	// 模块发送讯息到下位机
	sendDataChannel chan *SerialMessage
	// 中止发送数据通道
	stopSendDataChannel chan struct{}
}

// SendDataBuffer 发送数据缓存区，其中是将被发送的数据
type SendDataBuffer struct {
	// 数据
	data *[]byte
	// 预备被发送的数据帧编号
	frameID uint32
	// 该数据报编号
	bufferID uint32
	// 总数据帧量
	frameNum uint32
}

// SendBuffer 发送缓冲器
type SendBuffer struct {
	// 发送总缓冲区，每个COM口一个发送缓存区,这里存储了要通过这个COM口发送的数据。COM->SerialChannel->bufferID->*DataBuffer
	sendBuffer map[string]*map[*SerialChannel]*map[uint32]*SendDataBuffer
	// 预备发送缓冲区，这里的是正在轮换发送的数据
	readySendBuffer map[string]*map[*SerialChannel]*map[uint32]*SendDataBuffer
	// 发送数据报计数器 用于唯一的标记每个数据报
	i uint32
	// 发送线程的停止管道 COM->chan
	sendFuncStopChannels map[string]chan struct{}
	// App
	app *SerialApp
}

// RevDataBuffer 接收数据缓存区，其中是将被接收的数据
type RevDataBuffer struct {
	// 数据
	data *[]byte
	// 预备被接收的数据帧编号
	frameID uint32
	// 该数据报编号
	bufferID uint32
	// 总数据帧量
	frameNum uint32
	// 创建时间
	startTimeMicro int64
}

// RevBuffer 发送缓冲器
type RevBuffer struct {
	// 接收总缓冲区，每个COM口一个接收缓存区,这里存储了要通过这个COM口接收的数据。COM->SerialChannel->bufferID->*DataBuffer
	revBuffer map[string]*map[*SerialChannel]*map[uint32]*RevDataBuffer
	// 接收线程的停止管道 COM->chan
	revFuncStopChannels map[string]chan struct{}
	// App
	app *SerialApp
}
