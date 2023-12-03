package device

import (
	"sync"
	"time"

	_const "github.com/238Studio/child-nodes-assist/const"
	"go.bug.st/serial"
)
import serial_ "github.com/tarm/serial"

// InitSerialApp 初始化SerialApp
// 传入：COM口，波特率，超时时间
// 传出：未启动的串口
func InitSerialApp(baud int, readTimeout time.Duration, maxResendTimes int, RevBufferWaitTimeOut int64, SendBufferWaitTimeOut int64) *SerialApp {
	app := new(SerialApp)
	app.mu = new(sync.Mutex)
	app.isAlive = false
	app.SendBufferWaitTimeOut = SendBufferWaitTimeOut
	app.RevBufferWaitTimeOut = RevBufferWaitTimeOut
	app.serialDevicesByCOM = make(map[string]*SerialDevice)
	app.serialDevicesBySubModuleID = make(map[uint32]*map[string]*SerialDevice)
	app.serialChannelByNodeModulesID = make(map[uint32]*SerialChannel)
	app.revBuffer = &RevBuffer{
		revBuffer:              make(map[string]*map[uint32]*[]*[]byte),
		revFuncStopChannels:    make(map[string]chan struct{}),
		revBufferHangingPeriod: make(map[string]*map[uint32]int64),
		revBufferResidue:       make(map[string]*map[uint32]uint32),
		app:                    app,
	}
	app.maxResendTimes = maxResendTimes

	app.Baud = baud
	app.ReadTimeout = readTimeout
	app.sendBuffer = &SendBuffer{
		sendBuffer:           make(map[string]*map[*SerialChannel]*map[uint32]*SendDataBuffer),
		readySendBuffer:      make(map[string]*map[*SerialChannel]*map[uint32]*SendDataBuffer),
		sendBufferWaitTime:   make(map[string]*map[*SerialChannel]*map[uint32]int64),
		i:                    0,
		j:                    0xFFF,
		sendFuncStopChannels: make(map[string]chan struct{}),
		app:                  app,
	}
	app.frameFeedbackChannel = app.GetSerialMessageChannel(0xf)
	// todo 常量化
	return app
}

/*
自动初始化的具体的含义是 在启动串口服务后
初始化一个初始化模块 然后获取全部COM口 并向疑似下位机的初始化模块发送讯息 讯息内包括了下位机的COM口
下位机会返回其具备的模块->功能 映射表 COM口
COM口长度 COM %模块&功能,功能,功能...%模块...
此处为了方便处理 模块编号是字符串
*/

// AutoInitAndStartApp 自动和下位机沟通并初始化各个映射参数和COM参数
// 传入：无
// 传出：无
func (app *SerialApp) AutoInitAndStartApp(delayTime time.Duration) error {
	// 获取COM口并初始化，注册这些COM口
	ports, err := serial.GetPortsList()
	if err != nil {
		return err
	}
	for _, COM := range ports {
		serialDevice := new(SerialDevice)
		serialDevice.COM = COM
		serialDevice.isConnected = false
		serialDevice.SubModuleID = make([]uint32, 0)
		serialDevice.serialConfig = serial_.Config{
			Name:        COM,
			Baud:        app.Baud,
			ReadTimeout: app.ReadTimeout,
		}
		app.serialDevicesByCOM[COM] = serialDevice
	}
	// 启动COM口
	for COM := range app.serialDevicesByCOM {
		err := app.OpenPort(COM)
		if err != nil {
			return err
		}
		//todo:err
	}
	// 初始化初始化模块
	initMessageChannel := app.GetSerialMessageChannel(_const.InitModule)
	initSerialModuleApp := new(InitSerialModuleApp)
	initSerialModuleApp.channel = initMessageChannel
	initSerialModuleApp.dataProcessor = new(InitSerialDataProcessor)
	initSerialModuleApp.dataProcessor.app = app
	initSerialModuleApp.dataProcessor.rightDevices = make([]string, 0)
	initSerialModuleApp.serialDevicesBySubModuleID = make(map[byte]*map[string]*SerialDevice)
	// 给下位机发送初始化验证讯号
	for COM := range app.serialDevicesByCOM {
		d := initSerialModuleApp.dataProcessor.ProcessSendData(COM)
		err := app.sendToDevice(COM, &d)
		if err != nil {
			//TODO:err
		}
	}
	// 开始启动消息管道监听
	go app.StartAllListenMessage()
	// 开始监听 并监听一段时间
	app.StartAllListenMessage()
	time.Sleep(delayTime)
	initSerialModuleApp.channel.stopSendDataChannel <- struct{}{}
	for COM := range app.serialDevicesByCOM {
		isIn := false
		for _, COM_ := range initSerialModuleApp.dataProcessor.rightDevices {
			if COM == COM_ {
				isIn = true
			}
		}
		if !isIn {
			delete(app.serialDevicesByCOM, COM)
		}
	}
	return nil
}

// 开始监听初始模块管道信息
// 传入：无
// 传出：无
func (app *InitSerialModuleApp) startListenInitModule() {
	for {
		select {
		case <-app.channel.stopSendDataChannel:
			break
		case message := <-app.channel.receiveDataChannel:
			app.dataProcessor.ProcessReadData(message.data)
		}
	}
}

// ProcessReadData 从串口里读取 并处理数据 来完成下位机和它的功能模块的映射关系
/*传回数据的格式是
COMLen COM modulesNum modules
*/
// 传入：数据
// 传出：无
func (processor *InitSerialDataProcessor) ProcessReadData(data []byte) {
	// 获得下位机COM口
	COMLen := data[0]
	COM := string(data[1:COMLen])
	modulesNum := BytesToUint32(data[2:6])
	// 获得下位机支持的功能模块并进行注册
	var i uint32 = 0
	for i < modulesNum {
		i++
		module := BytesToUint32(data[6+i*4 : 10+i*4])
		_, ok := processor.app.serialDevicesBySubModuleID[module]
		if !ok {
			k := make(map[string]*SerialDevice)
			processor.app.serialDevicesBySubModuleID[module] = &k
		}
		(*processor.app.serialDevicesBySubModuleID[module])[COM] = processor.app.serialDevicesByCOM[COM]
	}
	processor.rightDevices = append(processor.rightDevices, COM)
}

// ProcessSendData 把字符串转为[]byte
// 传入：数据
// 传出：字符串
func (processor *InitSerialDataProcessor) ProcessSendData(data interface{}) []byte {
	data_ := ""
	data_ = data.(string)
	data__ := []byte(data_)
	return data__
}

// InitRevDataBuffer 根据数据报初始化RevDataBuffer
// 传入：一个数据报
// 传出：*RevDataBuffer
func InitRevDataBuffer(data *[]byte) *RevDataBuffer {
	rev := new(RevDataBuffer)
	frameID := BytesToUint32((*data)[0:4])
	bufferID := BytesToUint32((*data)[4:8])
	frameNum := BytesToUint32((*data)[8:12])
	exactLength := BytesToUint32((*data)[12:16])
	rev.frameID = frameID
	rev.bufferID = bufferID
	rev.frameNum = frameNum
	var d []byte
	//深拷贝
	copy(d, (*data)[16:16+exactLength])
	rev.data = &d
	return rev
}
