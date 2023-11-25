# API 文档
# `appAPI.go`
appAPI.go这个文件记录的是关于serialApp的方法。
## `(app *SerialApp) PutDeviceIntoSerialApp(device *SerialDevice)`

### 描述
将一个下位机注册到串口应用中 实现从COM口到串口设备的映射。在这里，下位机的结构体被称为“串口设备”，因为在这里的下位机特指的是串口设备。
另外，这个下位机是已经初始化好的，这个只负责注册而不负责别的工作。如果有重名（COM号）的下位机则新的下位机实例代替旧的，这个过程也仅仅是在注册层面代替，而不进行释放一类的操作。
### 输入
- 类型：`*SerialDevice`
- 串口设备对象指针

### 输出
- 无

## `(app *SerialApp) RemoveDeviceFromSerialApp(COM string)`

## 描述
将一个注册好的下位机从表中移除，这里并不会释放下位机，它仅仅是从表中移除。
另外 它同时会移除这个下位机的功能模块映射。
### 输入
- 类型：`string`
- COM序号
### 输出
- 无

## `(app *SerialApp) OpenPort(COM string) error`

## 描述
打开某个指定的硬件端口（COM口）
### 输入
- 类型：`string`
### 输出
- 类型：`error`
- 错误

## `(app *SerialApp) ClosePort(COM string) error`

## 描述
关闭某个指定的硬件端口（COM口）
### 输入
- 类型：`string`
- COM口序号
### 输出
- 类型：`error`
- 错误

## `RegisterSubModulesWithDevice(moduleID []uint32, COM string)`

## 描述
注册下位机关联模块 下位机功能模块->下位机集合 实现映射。
在这里，每个下位机都是
作为一系列功能模块的集合存在的，每个下位机都对应了一系列功能，这些功能的名称是全局唯一的，
也就是说，同一个功能可以有很多实例，但是不同功能不能重名。
### 输入
- 类型：`[]uint32`
- 该下位机所具有的模块ID
- 类型：`string`
- 该下位机的COM口序号
### 输出
无

## `(app *SerialApp) DeregisterSubModulesWithDevice(COM string)`

## 描述
取消这个下位机在模块功能中的注册。
### 输入
- 类型：`string`
- 该下位机的COM口序列
### 输出
无

## `(app *SerialApp) GetSerialMessageChannel(nodeModuleID uint32) *SerialChannel`

## 描述
获取并注册消息通道。
在这里获取的的消息通道（SerialChannel）包含了传递给下位机的管道，从下位机传回数据的管道，
以及终止该消息通道继续发送数据的通知管道。

## `(app *SerialApp) RemoveSerialChannel(nodeModuleID uint32)`

## 描述
取消注册一个管道，注意，这里只是取消注册，不包含释放等一系列操作。

## 传入
- 类型：`uint32`
- 节点模块ID

## 传出
- 类型：无

## `serialAPI.go`
操作串口的API的代码文件。

## `Uint32ToBytes(num uint32) []byte`

## 描述
将Uint32转为4位byte。

## 传入
- 类型：`uint32`
- 需要转换的32位无符号整数

## `BytesToUint32(bytes []byte) uint32`

## 描述
将4位bytes转为Uint32

## 传入
- 类型：`[]byte`
- 需要转换的bytes

## 传出
- 类型：`uint32`
- uint32

## `(app *SerialApp) send(channel *SerialChannel, targetModuleID uint32, targetFunction string, data *[]byte) error`

## 描述
将指定数据发送到指定功能模块。可能有多个下位机拥有相同的功能模块，此时
这些数据会被发送到这些全部下位机。
注意，这个方法是带锁的，也就是说，当一个上位机的模块在向下位机传输数据的时候，别的
上位机模块是不能向下位机传输数据的。
## 传入：
- 类型：`channel *SerialChannel`
- 发送数据的管道
- 类型：`targetModuleID uint32`
- 目标模块 也就是下位机的模块名
- 类型：`targetFunction string`
- 目标功能 也就是下位机这个模块需要执行的特定功能
- 类型：`data *[]byte`
- 数据数组的指针

## `(app *SerialApp) sendToDevice(targetModuleID uint32, targetFunction string, COM string, data *[]byte) error`

## 描述
将指定数据发送到特定的下位机。