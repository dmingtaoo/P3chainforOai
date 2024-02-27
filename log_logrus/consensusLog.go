package loglogrus

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
)

// 将上下层共识消息单独写入到一个日志文件中
type ConsensusHook struct {
	tpbftWriter io.Writer
	raftWriter  io.Writer
}

func (hook *ConsensusHook) Fire(entry *logrus.Entry) error {
	line, err := entry.String()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to read entry, %v", err)
		return err
	}
	if strings.Contains(line, "[TPBFT Consensus]") {
		hook.tpbftWriter.Write([]byte(line))
	}
	if strings.Contains(line, "[RAFT Consensus]") {
		hook.raftWriter.Write([]byte(line))
	}

	return nil
}

// 不针对特定等级的日志消息
func (hook *ConsensusHook) Levels() []logrus.Level {
	return logrus.AllLevels
}
