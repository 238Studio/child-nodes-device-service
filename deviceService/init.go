package deviceService

import (
	_const "github.com/UniversalRobotDriveTeam/child-nodes-assist/const"
	"go.bug.st/serial"
	"strconv"
	"strings"
	"sync"
	"time"
)
import serial_ "github.com/tarm/serial"

// InitSerialApp 初始化SerialApp
// 传入：COM口，波特率，超时时间
// 传出：未启动的串口
func InitSerialApp(Baud int, ReadTimeout time.Duration, sendCacheLength int) *SerialApp {
	app := new(SerialApp)
	app.mu = new(sync.Mutex)
	app.isAlive = false
	app.serialDevicesByCOM = make(map[string]*SerialDevice)
	app.serialDevicesBySubModuleID = make(map[byte]map[string]*SerialDevice)
	app.serialChannelByNodeModulesID = make(map[byte]*SerialChannel, 1)
	app.stopListenSubMessageChannel = make(map[string]chan int, 1)
	app.serialDevicesFunctionByModuleID = make(map[byte][]string, 1)
	app.dataCache = make(map[string][]*SerialMessage, sendCacheLength)
	app.maxDataCache = sendCacheLength
	app.dataCacheIDNow = 0
	app.Baud = Baud
	app.ReadTimeout = ReadTimeout
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
		serialDevice.SubModuleID = make([]byte, 0)
		serialDevice.serialConfig = serial_.Config{
			Name:        COM,
			Baud:        app.Baud,
			ReadTimeout: app.ReadTimeout,
		}
		app.serialDevicesByCOM[COM] = serialDevice
	}
	// 启动COM口
	for COM, _ := range app.serialDevicesByCOM {
		app.OpenPort(COM)
	}
	// 初始化初始化模块
	initMessageChannel := app.GetSerialMessageChannel(_const.InitModule)
	initSerialModuleApp := new(InitSerialModuleApp)
	initSerialModuleApp.channel = initMessageChannel
	initSerialModuleApp.dataProcessor = new(InitSerialDataProcessor)
	initSerialModuleApp.dataProcessor.app = app
	initSerialModuleApp.dataProcessor.rightDevices = make([]string, 0)
	initSerialModuleApp.serialDevicesBySubModuleID = make(map[byte]map[string]*SerialDevice)
	// 给下位机发送初始化验证讯号
	for COM, _ := range app.serialDevicesByCOM {
		d := initSerialModuleApp.dataProcessor.ProcessSendData(COM)
		app.sendToDevice(_const.InitModule, "", COM, &d)
	}
	// 开始启动消息管道监听
	go app.StartAllListenMessage()
	// 开始监听 并监听一段时间
	app.StartAllListenMessage()
	time.Sleep(delayTime)
	initSerialModuleApp.channel.stopSendDataChannel <- 0
	for COM, _ := range app.serialDevicesByCOM {
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

// ProcessReadData 从串口里读取 并处理数据
// 传入：数据
// 传出：无
func (processor *InitSerialDataProcessor) ProcessReadData(data []byte) {
	COMLen := data[0]
	COM := string(data[1:COMLen])
	modules_ := strings.Split(string(data[COMLen:]), "%")
	for i := range modules_ {
		ms := strings.Split(modules_[i], "&")
		fs := strings.Split(ms[1], ",")
		subModuleID, err := strconv.ParseInt(ms[0], 10, 8)
		if err != nil {
			return
		}
		// 数据格式错误则直接返回
		processor.app.serialDevicesFunctionByModuleID[byte(subModuleID)] = fs
		_, ok := processor.app.serialDevicesBySubModuleID[byte(subModuleID)]
		if !ok {
			processor.app.serialDevicesBySubModuleID[byte(subModuleID)] = make(map[string]*SerialDevice)
		}
		processor.app.serialDevicesBySubModuleID[byte(subModuleID)][COM] = processor.app.serialDevicesByCOM[COM]
	}
	processor.rightDevices = append(processor.rightDevices, COM)
}

// ProcessSendData 把数据转为byte[]
// 将字符串转为bytes
func (processor *InitSerialDataProcessor) ProcessSendData(data interface{}) []byte {
	data_ := ""
	data_ = data.(string)
	data__ := []byte(data_)
	return data__
}
