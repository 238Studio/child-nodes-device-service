package deviceService

// Start 启动模块服务
// 传入：启动参数
// 传出：无
func (app *SerialApp) Start() {
	app.StartAllListenMessage()
	app.isAlive = true
}

// Stop 中止模块服务
// 传入：无
// 传出：无
func (app *SerialApp) Stop() {
	app.isAlive = false
	app.StopAllListenMessage()
}

// GetApp 获取App
// 传入：无
// 传出：该模块App的指针
func (app *SerialApp) GetApp() *interface{} {
	var value interface{}
	value = app
	return &value
}

// IsAlive 是否在服务
// 传入：无
// 传出：是否在服务
func (app *SerialApp) IsAlive() bool {
	return app.isAlive
}
