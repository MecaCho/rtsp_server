package rtspclient

import (
	"net"
	"time"
	"glog"
	"fmt"
)


func DecKeytool(value string) (string, error){
	return "", nil
}

// NewRTSPClient ...
func NewRTSPClient(address string) (net.Conn, error) {
	con, err := net.DialTimeout("tcp", address, 10*time.Second)
	if err != nil {
		glog.Errorf("connect to remote camera err: %s.", err.Error())
	}
	return con, err
}

func CheckMain(address, CameraURL string) (string, error){
	checkingLog := fmt.Sprintf("")

	return checkingLog, nil

}
