package nat

import (
	"fmt"
	"io/ioutil"
	"net"
	"time"
)

type addrReflector struct {
	filePath    string //存储自身对外公网IP地址的文件路径
	staticExtIP net.IP //存储从文件获取的net.IP格式的IP地址
}

// These methods manage a mapping between a port on the local
// machine to a port that can be connected to from the internet.
//
// protocol is "UDP" or "TCP". Some implementations allow setting
// a display name for the mapping. The mapping may be removed by
// the gateway when its lifetime ends.
// 形参protocol可以是UDP也可是TCP. 形参name表示在映射时可以设置映射的显示名称。
// lifetime表示映射存在生存期(当生存期结束时，映射关系会被NAT设备删除)
func (sr *addrReflector) AddMapping(protocol string, extport int, intport int, name string, lifetime time.Duration) error {
	return nil
}

func (sr *addrReflector) DeleteMapping(protocol string, extport int, intport int) error {
	return nil
}

// This method should return the external (Internet-facing)
// address of the gateway device(read from file).
// 本方法负责返回NAT设备的外网地址(从文件读取)
func (sr *addrReflector) ExternalIP() (net.IP, error) {

	if sr.staticExtIP != nil {
		return sr.staticExtIP, nil
	}
	content, err := ioutil.ReadFile(sr.filePath) //从文件中读取自己的公网IP
	if err != nil {
		fmt.Errorf("Read external ip from %s is failed\n", sr.filePath)
		return nil, err
	}
	ExIP := net.ParseIP(string(content))
	sr.staticExtIP = ExIP
	fmt.Printf("Read External IP:%s\n", content)
	return sr.staticExtIP, nil
}

// Should return name of the method. This is used for logging.
func (sr *addrReflector) String() string {
	return fmt.Sprintf("Get ExIP from : %s", sr.filePath)
}
