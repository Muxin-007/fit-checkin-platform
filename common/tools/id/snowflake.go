package id

import (
	"os"
	"strconv"
	"sync"
)

var idInstance IDGenerator
var nodeID uint16

func MachineID() (uint16, error) {
	return nodeID, nil
}

var once = new(sync.Once)

func InitSonyflake(_nodeID uint16) {
	nodeID = _nodeID
	once.Do(func() {
		var err error
		idInstance, err = NewSonySnowFlake(MachineID) // 创建snowflake实例

		if err != nil {
			os.Exit(1)
		}
	})
}

// GenID 生成全局唯一ID
func GenID() string {
	return strconv.FormatUint(idInstance.GenID(), 10)
}
