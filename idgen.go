package idgen

import (
	"errors"
	"fmt"
	"math"
	"net"
	"strings"
	"sync"
	"time"
)

/**
 * 以ip后16位掩码作为nodeId, 理论节点数为65536个。足以在k8s这样的环境内保证产生的ID全局唯一。
 * time clash 防止时间回拨，默认允许时间回拨1500毫秒,适应闰秒的情况或者电脑时间误差。
 * 潜在乱序:由数据结构可以看出，在同10毫秒内，跨节点上产生的id不是严格递增的。
 * 时间 37 bit,10毫秒单位,以北京时间 cst 2016/6/6 6:6:6为标准差，44年左右,id可以持续到 cst 2059/12/25 12:58:20
 * 以ip后16bit为nodeId,每秒可以产生 2**8*100=25600=2.56万个id,一般用于程序本地生产id。
 * 自定义nodeId(10bit,0-1023),每秒可以产生 2**14*100=1638400=163.84万个id,一般用于某个数据中心的远程公共id生产服务。
 * 情况1:
 * isCustom==0
 * nodeId: 16bit ip as nodeId,(example: ip 172.16.1.16 nodeId: 0x010F)
 *  * +------+-----------------+----------+--------+----------+----------+
 *  * | sign |  delta seconds  | isCustom |16bit ip|time clash| sequence |
 *  * +------+-----------------+----------+--------+----------+----------+
 *  * | 1bit      37bits       |  1bit:0  | 16bits |   1bit   |  8bits   |
 *  * +------+-----------------+----------+--------+----------+----------+
 * 情况2:
 * isCustom==1
 * nodeId: 0~1023
 *  * +------+-----------------+----------+--------+----------+----------+
 *  * | sign |  delta seconds  | isCustom |node id |time clash| sequence |
 *  * +------+-----------------+----------+--------+----------+----------+
 *  * | 1bit      37bits       |  1bit:1  | 10bits |   1bit   |  14bits  |
 *  * +------+-----------------+----------+--------+----------+----------+
 *
 * origin snowflake
 *     1bit   41bits     10bits       12bits
 * total 64 bit
 * *
 */
const (
	CEpoch        = 146516436600 //北京时间 2016/6/6 6:6:6 CST ,10毫秒单位
	flakeTimeUnit = 1e7          // nsec, i.e. 10 msec
	ALPHABET      = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
)

type IdWorker struct {
	nodeId           int64
	lastTimeStamp    int64
	sequence         int64
	timeBackTag      int64 //time back tag
	timeBackInterval int64 // time back interval(*10ms)
	timeBackStamp    int64 // the last time happened of time back
	lock             *sync.Mutex
	isCustom         bool // is or not coustom node id
}

func NewNodeIdByIpAndTimeBackInterval(timeBackInterval int64) (iw *IdWorker, err error) {
	iw = new(IdWorker)
	ip16bit, err := lower16BitPrivateIP()
	if err != nil {
		return
	}
	iw.nodeId = int64(ip16bit & 0xFFFF)
	iw.lastTimeStamp = -1
	iw.lock = new(sync.Mutex)
	iw.isCustom = false
	iw.timeBackInterval = timeBackInterval //default 50 ms (5*flakeTimeUnit)
	return iw, nil
}

//以IPv4的后16bit 作为nodeId
func NewNodeIdByIp() (iw *IdWorker, err error) {
	return NewNodeIdByIpAndTimeBackInterval(150)
}
func NewCustomNodeIdAndTimeBackInterval(nodeid, timeBackInterval int64) (iw *IdWorker, err error) {
	iw = new(IdWorker)
	if nodeid > 0x7 || nodeid < 0 {
		return nil, errors.New("worker not fit,must between 0 and 7")
	}
	iw.nodeId = nodeid
	iw.lastTimeStamp = -1
	iw.isCustom = true
	iw.lock = new(sync.Mutex)
	iw.timeBackInterval = timeBackInterval //default 50 ms (5*flakeTimeUnit)
	return iw, nil
}

// NewCustomNodeId Func: Generate NewCustomNodeId with Given workerid
func NewCustomNodeId(nodeid int64) (iw *IdWorker, err error) {
	return NewCustomNodeIdAndTimeBackInterval(nodeid, 150)
}

// NewId Func: Generate next id
func (iw *IdWorker) NextId() (id int64, err error) {
	iw.lock.Lock()
	defer iw.lock.Unlock()
	currTime := iw.timeGen()
	delta := iw.lastTimeStamp - currTime
	if delta == 0 {
		if (iw.isCustom == false && iw.sequence < 0xFF) || (iw.isCustom == true && iw.sequence < 0x3FFF) {
			iw.sequence++
		} else {
			//sleep:=time.Duration(int64(flakeTimeUnit-time.Now().Nanosecond()%flakeTimeUnit)) * time.Nanosecond
			//fmt.Printf("sleep %s\n",sleep.String())
			//sleep 1-10 ms
			time.Sleep(time.Duration(int64(flakeTimeUnit-time.Now().Nanosecond()%flakeTimeUnit)) * time.Nanosecond)
			currTime = iw.timeGen()
			if currTime-1 < iw.lastTimeStamp {
				delta = iw.lastTimeStamp - currTime + 1
			}
			iw.sequence = 0
		}
	}
	if delta < 0 {
		iw.sequence = 0
	}
	if delta > 0 {
		backDelta := iw.timeBackStamp - currTime
		if backDelta < 0 {
			if iw.timeBackTag == 0 {
				iw.timeBackTag = 1
			}
			if iw.timeBackTag == 1 {
				iw.timeBackTag = 0
			}
			iw.timeBackStamp = currTime
			iw.sequence = 0
		} else {
			if backDelta <= iw.timeBackInterval {
				//sleep backDelta*10+10 ms
				//fmt.Printf("--- time back, sleep %s\n",(time.Duration(backDelta*10+10) * time.Millisecond).String())
				time.Sleep(time.Duration(backDelta*10+10) * time.Millisecond)
				currTime = iw.timeGen()
				//check clock again
				if currTime < iw.timeBackStamp {
					//fmt.Printf("currTime:%d , timeBackStamp %d, delta: %d ms\n", currTime,iw.timeBackStamp, (currTime-iw.lastTimeStamp)*10)
					//fmt.Printf("backDelta:%d ms\n",backDelta*10)
					errStr := fmt.Sprintf("Clock moved backwards and more than timeBackInterval:%d ms, Refuse gen id", iw.timeBackInterval*10)
					//err = errors.New("Clock moved backwards and more than timeBackInterval, Refuse gen id")
					err = errors.New(errStr)
					return
				} else {
					iw.timeBackStamp = currTime
					iw.sequence = 0
				}
			} else {
				err = errors.New("- Clock moved backwards and more than timeBackInterval, Refuse gen id")
				return
			}

		}
	}

	iw.lastTimeStamp = currTime
	if iw.timeBackStamp > 0 {
		iw.timeBackStamp = currTime
	}
	if iw.isCustom == false {
		id = (currTime-CEpoch)<<26 | iw.nodeId<<9 | iw.timeBackTag<<8 | iw.sequence
	} else {
		id = (currTime-CEpoch)<<26 | 1<<25 | iw.nodeId<<15 | iw.timeBackTag<<14 | iw.sequence
	}
	return id, nil
}

// ParseId Func: reverse uid to timestamp, workid, seq
func ParseId(id int64) (t time.Time, ts int64, workerId int64, seq int64, isCustomNodeId bool) {
	if id>>25&0x1 != 1 {
		//println("-id", id)
		isCustomNodeId = false
		seq = id & 0xFF //8bit
		workerId = (id >> 9) & 0xffff
		ts = (id >> 26) + CEpoch
		t = time.Unix(0, ts*flakeTimeUnit)
	} else {
		isCustomNodeId = true
		seq = id & 0x3fff // 14bit
		workerId = (id >> 15) & 0x3FF
		ts = (id >> 26) + CEpoch
		t = time.Unix(0, ts*flakeTimeUnit)
	}
	return
}

// return 10ms unit
func (iw *IdWorker) timeGen() int64 {
	return time.Now().UnixNano() / flakeTimeUnit
}

func lower16BitPrivateIP() (uint16, error) {
	ip, err := privateIPv4()
	if err != nil {
		return 0, err
	}
	return uint16(ip[2])<<8 + uint16(ip[3]), nil
}

func privateIPv4() (net.IP, error) {
	as, err := net.InterfaceAddrs()
	if err != nil {
		return nil, err
	}

	for _, a := range as {
		ipnet, ok := a.(*net.IPNet)
		if !ok || ipnet.IP.IsLoopback() {
			continue
		}

		ip := ipnet.IP.To4()
		if isPrivateIPv4(ip) {
			return ip, nil
		}
	}
	return nil, errors.New("no private ip address")
}

func isPrivateIPv4(ip net.IP) bool {
	return ip != nil &&
		(ip[0] == 10 || ip[0] == 172 && (ip[1] >= 16 && ip[1] < 32) || ip[0] == 192 && ip[1] == 168)
}

/**
 * int64 encode to string
 */
func Encode62Str(longNum int64) (str string, err error) {
	if longNum < 0 {
		err = errors.New("input num must >=0")
		return
	}
	result := make([]byte, 0)
	for longNum > 0 {
		round := longNum / 62
		remain := longNum % 62
		result = append(result, ALPHABET[remain])
		longNum = round
	}
	if len(result) < 1 {
		err = errors.New("string join error")
		return
	}
	str = string(result)
	return
}

/**
 * string decode to int64
 */
func Decode64Str(str62 string) (longNum int64, err error) {
	// max_int64  7m85Y0n8LzA
	length := len(str62)
	if length == 0 || length > 11 {
		err = errors.New("the length of input string must > 0")
		return
	}
	for index, char := range []byte(str62) {
		longNum += int64(strings.Index(ALPHABET, string(char))) * int64(math.Pow(62, float64(index)))
	}
	return
}
