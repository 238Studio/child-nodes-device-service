package deviceService_test

import (
	"github.com/UniversalRobotDriveTeam/child-nodes-basic/robotBasicAPI/deviceService"
	"testing"
	"time"
)

func TestName(t *testing.T) {
	app := deviceService.InitSerialApp(9600, time.Millisecond, 16)
	println(app)

}
