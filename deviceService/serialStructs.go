package deviceService

import (
	"github.com/UniversalRobotDriveTeam/child-nodes-assist/util"
	"github.com/tarm/serial"
	"sync"
	"time"
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
	SubModuleID []byte
	// 是否处于连接状态
	isConnected bool
}

// SerialApp 容纳串口操作和信息的应用
type SerialApp struct {
	// 波特率
	Baud int
	// 串口消息等待时间
	ReadTimeout time.Duration
	// 互斥锁
	mu *sync.Mutex
	// 从下位机的模块对应了若干个下位机的串口收发模块 NodeModuleID->SerialAppPerDevice
	serialDevicesBySubModuleID map[byte]map[string]*SerialDevice
	// 下位机模块功能映射
	serialDevicesFunctionByModuleID map[byte][]string
	// COM->SerialAppPerDevice
	serialDevicesByCOM map[string]*SerialDevice
	// 从子节点功能模块对应到moduleID->SerialChannel
	serialChannelByNodeModulesID map[byte]*SerialChannel
	// 中止接收下位机传来信息的通道 COM->channel
	stopListenSubMessageChannel map[string]chan int
	// 最大数据缓存长度
	maxDataCache int
	// 数据缓存 重发用 COM->data指针
	dataCache map[string][]*SerialMessage
	// 当前缓存指向的数据
	dataCacheIDNow int
	// 是否运行
	isAlive bool
}

// SerialDataProcessor 将原始的串口二进制数据转换成需要的对象
type SerialDataProcessor interface {
	// ProcessReadData 从串口里读取
	ProcessReadData(data []byte)
	// ProcessSendData 把数据转为byte[]
	ProcessSendData(data interface{}) []byte
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
	targetModuleID byte
	// 目标模块功能
	targetFunction string
	// 数据
	data []byte
}

// SerialChannel 和串口进行交互的对象
type SerialChannel struct {
	// 串口应用
	app                 *SerialApp
	receiveErrorChannel chan *util.CustomError
	// 模块从下位机收到数据的通道
	receiveDataChannel chan *SerialMessage
	// 模块发送讯息到下位机
	sendDataChannel chan *SerialMessage
	// 中止发送数据通道
	stopSendDataChannel chan int
}
