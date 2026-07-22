package id

import (
	"context"
	"fmt"
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/redis/go-redis/v9"
	"github.com/sony/sonyflake"
)
// sonyflakeEpoch 与 github.com/sony/sonyflake 在 StartTime 为零时的默认纪元一致。
var sonyflakeEpoch = time.Date(2014, 9, 1, 0, 0, 0, 0, time.UTC)

type IDGenerator interface {
	GenID() uint64
}

type SnowFlake struct {
	node *snowflake.Node
}

func (s *SnowFlake) GenID() uint64 {
	return uint64(s.node.Generate().Int64())
}

func GenNodeID(cli redis.Cmdable, svc string) (int64, error) {
	key := fmt.Sprintf("id:%s", svc)
	ctx := context.Background()

	machineID, err := cli.Incr(ctx, key).Result()
	if err != nil {
		return 0, err
	}

	// 如果机器ID超过1023，重置为0
	if machineID >= 1024 {
		machineID = 0
		_, err := cli.Set(ctx, key, machineID, 0).Result()
		if err != nil {
			return 0, err
		}
	}

	return machineID, nil
}

func NewIDGenerator(nodeID int64) (IDGenerator, error) {
	var err error
	node, err := snowflake.NewNode(nodeID)
	if err != nil {
		return nil, err
	}
	return &SnowFlake{node}, nil
}

type SonySnowFlake struct {
	node *sonyflake.Sonyflake
}

func NewSonySnowFlake(machineId func() (uint16, error)) (IDGenerator, error) {
	settings := sonyflake.Settings{
		MachineID: machineId,
		StartTime: sonyflakeEpoch,
	}
	node := sonyflake.NewSonyflake(settings)
	return &SonySnowFlake{node}, nil
}

func (s *SonySnowFlake) GenID() uint64 {
	v, _ := s.node.NextID()
	return v
}
