package device

import (
	"strconv"
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
		sendBuffer:           make(map[string]*map[uint32]*SendDataBuffer),
		readySendBuffer:      make(map[string]*map[uint32]*SendDataBuffer),
		sendBufferWaitTime:   make(map[string]*map[uint32]int64),
		i:                    0,
		j:                    0xFFF,
		sendFuncStopChannels: make(map[string]*chan struct{}),
		app:                  app,
	}
	app.frameFeedbackChannel = app.GetSerialMessageChannel(_const.FeedbackModule)
	app.initDeviceChannel = app.GetSerialMessageChannel(_const.InitModule)
	sc := make(chan struct{})
	app.stopInitDeviceChannel = &sc
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

// AutoInitAllDevices 自动和下位机沟通并初始化各个映射参数和COM参数
// 传入：无
// 传出：无
func (app *SerialApp) AutoInitAllDevices() *[]error {
	errs := make([]error, 0)
	// 获取COM口并初始化，注册这些COM口
	ports, err := serial.GetPortsList()
	if err != nil {
		errs = append(errs, err)
	}
	for _, COM := range ports {
		println("发现串口:" + COM)
		err := app.AutoInitPerDevice(COM)
		if err != nil {
			errs = append(errs, err)
		}
	}
	return &errs
}

// AutoInitPerDevice 自动初始化一个COM口
// 传入：COM
// 传出：无
func (app *SerialApp) AutoInitPerDevice(COM string) error {
	// 生成串口配置
	serialDevice := new(SerialDevice)
	serialDevice.COM = COM
	serialDevice.isConnected = false
	serialDevice.SubModuleID = make([]uint32, 0)
	serialDevice.serialConfig = serial_.Config{
		Name:        COM,
		Baud:        app.Baud,
		ReadTimeout: app.ReadTimeout,
	}
	// 将设备加入设备列表
	app.PutDeviceIntoSerialApp(serialDevice)
	// 给设备发送其COM号
	// 启动COM口
	err := app.OpenPort(COM)
	if err != nil {
		return err
	}
	COM_, _ := strconv.ParseInt(COM[3:], 10, 8)
	buffer := []byte{byte(COM_)}
	_, err = serialDevice.portIO.Write(buffer)
	if err != nil {
		return err
	}
	return nil
}

// StartAutoInit 开启自动初始化 分析从initChannel传回的数据报 来获得下位机支持的模块
// 传入：无
// 传出：无
func (app *SerialApp) StartAutoInit() {
	go func() {
		for {
			select {
			case <-*app.stopInitDeviceChannel:
				break
			case msg := <-(*app.initDeviceChannel.ReceiveDataChannel):
				println("收到:" + string(msg.Data))
				if msg.TargetFunction == "InitData" {
					i := 0
					n := (len(msg.Data) - 1) / 4
					modules := make([]uint32, 0)
					COM := "COM" + string(msg.Data[0])
					for i < n {
						modules = append(modules, BytesToUint32(msg.Data[i*4+1:i*4+5]))
						i++
					}
					app.RegisterSubModulesWithDevice(modules, COM)
				}
			default:
				continue
			}
		}
	}()
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
