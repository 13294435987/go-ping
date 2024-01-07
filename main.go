package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"log"
	"math"
	"net"
	"os"
	"time"
)

var (
	timeout      int64
	size         int
	count        int
	typ          uint8 = 8
	code         uint8 = 0
	sendCount    int
	successCount int
	failCount    int
	minTimeUse   int64 = math.MaxInt32
	maxTimeUse   int64
	totalTimeUse int64
)

// ICMP 定义头部信息
type ICMP struct {
	Type        uint8
	Code        uint8
	CheckSum    uint16
	ID          uint16
	SequenceNum uint16
}

func main() {
	getCommandArgs()
	// 目标IP地址，取命令行最后一个参数
	targetIp := os.Args[len(os.Args)-1]
	// 创建一个连接
	conn, err := net.DialTimeout("ip:icmp", targetIp, time.Duration(timeout)*time.Millisecond)
	if err != nil {
		log.Fatal(err)
		return
	}

	// 关闭连接
	defer func(conn net.Conn) {
		_ = conn.Close()
	}(conn)

	fmt.Printf("正在 Ping %s [%s] 具有 %d 字节的数据:\n", targetIp, conn.RemoteAddr(), size)
	for i := 0; i < count; i++ {
		sendCount++
		icmp := &ICMP{
			Type:        typ,
			Code:        code,
			CheckSum:    0,
			ID:          1,
			SequenceNum: 1,
		}

		data := make([]byte, size)
		var buffer bytes.Buffer
		timeStart := time.Now()
		_ = binary.Write(&buffer, binary.BigEndian, icmp)
		buffer.Write(data)
		data = buffer.Bytes()
		checkSum := checkSum(data)
		// 高8位
		data[2] = byte(checkSum >> 8)
		// 低8位
		data[3] = byte(checkSum)
		_ = conn.SetDeadline(time.Now().Add(time.Duration(timeout) * time.Millisecond))
		n, err := conn.Write(data)
		if err != nil {
			failCount++
			log.Println(err)
			continue
		}
		buf := make([]byte, 65535)
		n, err = conn.Read(buf)
		if err != nil {
			failCount++
			log.Println(err)
			continue
		}
		successCount++
		timeUse := time.Since(timeStart).Milliseconds()
		if minTimeUse > timeUse {
			minTimeUse = timeUse
		}
		if maxTimeUse < timeUse {
			maxTimeUse = timeUse
		}
		totalTimeUse += timeUse
		fmt.Printf("来自 %d.%d.%d.%d 的回复: 字节=%d 时间=%d ms TTL=%d\n",
			buf[12], buf[13], buf[14], buf[15], n-28, timeUse, buf[8])
		time.Sleep(time.Second)
	}

	fmt.Printf("%s 的 Ping 统计信息:                          \n"+
		"    数据包: 已发送 = %d，已接收 = %d，丢失 = %d (%.2f%% 丢失)，\n"+
		"往返行程的估计时间(以毫秒为单位):                       \n    "+
		"最短 = %dms，最长 = %dms，平均 = %dms ",
		conn.RemoteAddr(), sendCount, successCount, failCount, float64(failCount)/float64(sendCount)*100, minTimeUse, maxTimeUse, totalTimeUse/int64(sendCount))
}

// getCommandArgs 获取命令行的参数
func getCommandArgs() {
	flag.Int64Var(&timeout, "w", 1000, "请求超时时长，单位毫秒")
	flag.IntVar(&size, "l", 32, "请求发送缓冲区大小，单位字节")
	flag.IntVar(&count, "n", 4, "发送请求数")
	flag.Parse()
}

func checkSum(data []byte) uint16 {
	length := len(data)
	index := 0
	var sum uint32 = 0

	for length > 1 {
		sum += uint32(data[index])<<8 + uint32(data[index+1])
		length -= 2
		index += 2
	}

	if length != 0 {
		sum += uint32(data[index])
	}
	highOrder16 := sum >> 16
	for highOrder16 != 0 {
		sum = highOrder16 + uint32(uint16(sum))
		highOrder16 = sum >> 16
	}
	return uint16(^sum)
}
