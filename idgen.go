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
 * time clash 防止时间回拨，默认允许时间回拨1000毫秒,适应闰秒的情况或者电脑时间误差。
 * 潜在乱序:由数据结构可以看出，在同10毫秒内，跨节点上产生的id不是严格递增的。
 * 时间 37 bit,10毫秒单位,以北京时间 2023-07-21T00:00:00+08:00 为标准差，44年左右
 * 以ip后16bit为nodeId,每秒可以产生 2**8*100=25600=2.56万个id,一般用于程序本地生产id。
 * 自定义nodeId(10bit,0-1023),每秒可以产生 2**14*100=1638400=163.84万个id,一般用于某个数据中心的远程公共id生产服务。
 * condition 1:
 * nodeID.isCustom==0
 * nodeId: 16bit ip as nodeId,(example: ip 172.16.1.16 nodeId: 0x010F)
 *  * +------+-----------------+----------+--------+----------+----------+
 *  * | sign |  delta seconds  | isCustom |16bit ip|time clash| sequence |
 *  * +------+-----------------+----------+--------+----------+----------+
 *  * | 1bit      37bits       |  1bit:0  | 16bits |   1bit   |  8bits   |
 *  * +------+-----------------+----------+--------+----------+----------+
 * condition 2:
 * nodeID.isCustom==1
 * nodeId: 0~1023
 *  * +------+-----------------+----------+--------+----------+----------+
 *  * | sign |  delta seconds  | isCustom |node id |time clash| sequence |
 *  * +------+-----------------+----------+--------+----------+----------+
 *  * | 1bit      37bits       |  1bit:1  | 10bits |   1bit   |  14bits  |
 *  * +------+-----------------+----------+--------+----------+----------+
 *
 * twitter origin snowflake
 *     1bit   41bits     10bits       12bits
 * total 64 bit
 * *
 */
const (
	//CEpoch        = 146516436600 //北京时间 2016/6/6 6:6:6 CST ,10毫秒单位
	CEpoch        = 168986880000 //2023-07-21T00:00:00+08:00 ,10毫秒单位
	flakeTimeUnit = 1e7          // nsec, i.e. 10 msec
	ALPHABET      = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	ALPHABET82    = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz!@$^*()_-{}[]<>,.`~|"
)

// IDWorker id worker
type IDWorker struct {
	nodeID           int64 //节点id
	lastTimeStamp    int64 //最后时间戳
	sequence         int64 //序列号
	timeBackTag      int64 //time back tag
	timeBackInterval int64 // time back interval(*10ms)
	timeBackStamp    int64 // the last time happened of time back
	lock             *sync.Mutex
	isCustom         bool // is or not coustom node id
}

// IDDetail  Detail of ID
type IDDetail struct {
	isCustom    bool      //是否定制
	nodeID      int64     //节点
	ts          time.Time //时间
	sequence    int64     //序列号
	isTimeClash bool      //time back tag
}

// NewNodeIDByIPAndTimeBackInterval define the ms of backinterval that is allow
func NewNodeIDByIPAndTimeBackInterval(timeBackInterval int64) (iw *IDWorker, err error) {
	iw = new(IDWorker)
	ip16bit, err := lower16BitPrivateIP()
	if err != nil {
		return
	}
	iw.nodeID = int64(ip16bit & 0xFFFF)
	iw.lastTimeStamp = -1
	iw.lock = new(sync.Mutex)
	iw.isCustom = false
	iw.timeBackInterval = timeBackInterval //default 50 ms (5*flakeTimeUnit)
	return iw, nil
}

// NewNodeIDByIP 以IPv4的后16bit 作为nodeId
func NewNodeIDByIP() (iw *IDWorker, err error) {
	return NewNodeIDByIPAndTimeBackInterval(100)
}

// NewCustomNodeIDAndTimeBackInterval custom nodeid and the interval of timeback
func NewCustomNodeIDAndTimeBackInterval(nodeid, timeBackInterval int64) (iw *IDWorker, err error) {
	iw = new(IDWorker)
	if nodeid > 1023 || nodeid < 0 {
		return nil, errors.New("worker not fit,must between 0 and 1023")
	}
	iw.nodeID = nodeid
	iw.lastTimeStamp = -1
	iw.isCustom = true
	iw.lock = new(sync.Mutex)
	iw.timeBackInterval = timeBackInterval //default 50 ms (5*flakeTimeUnit)
	return iw, nil
}

// NewCustomNodeID Func: Generate NewCustomNodeID with Given workerid
func NewCustomNodeID(nodeid int64) (iw *IDWorker, err error) {
	return NewCustomNodeIDAndTimeBackInterval(nodeid, 100)
}

// NextID Func: Generate next id
func (iw *IDWorker) NextID() (id int64, err error) {
	iw.lock.Lock()
	defer iw.lock.Unlock()
	currTime := iw.timeGen()
	delta := iw.lastTimeStamp - currTime
	if delta == 0 {
		if !(iw.isCustom == false && iw.sequence < 0xFF) && !(iw.isCustom == true && iw.sequence < 0x3FFF) {
			//sleep 1-10 ms
			time.Sleep(time.Duration(int64(flakeTimeUnit-time.Now().Nanosecond()%flakeTimeUnit)) * time.Nanosecond)
			currTime = iw.timeGen()
			if currTime-1 < iw.lastTimeStamp {
				delta = iw.lastTimeStamp - currTime + 1
			}
			iw.sequence = 0
		} else {
			iw.sequence++
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
			iw.sequence = 0
		} else {
			if backDelta <= iw.timeBackInterval {
				//sleep backDelta*10+10 ms
				time.Sleep(time.Duration(backDelta*10+10) * time.Millisecond)
				currTime = iw.timeGen()
				//check clock again
				if currTime < iw.timeBackStamp {
					errStr := fmt.Sprintf("Clock moved backwards and more than timeBackInterval:%d ms, Refuse gen id", iw.timeBackInterval*10)
					err = errors.New(errStr)
				} else {
					iw.sequence = 0
				}
			} else {
				// Avoid DDOS attacks
				time.Sleep(time.Duration(iw.timeBackInterval*10) * time.Millisecond)
				err = errors.New("- Clock moved backwards and more than timeBackInterval, Refuse gen id")
			}
			if err != nil {
				return 0, err
			}

		}
	}

	iw.lastTimeStamp = currTime
	if iw.timeBackStamp < currTime {
		iw.timeBackStamp = currTime
	}
	if iw.isCustom == false {
		id = (currTime-CEpoch)<<26 | iw.nodeID<<9 | iw.timeBackTag<<8 | iw.sequence
	} else {
		id = (currTime-CEpoch)<<26 | 1<<25 | iw.nodeID<<15 | iw.timeBackTag<<14 | iw.sequence
	}
	return id, nil
}

// ParseID  parse int64 to struct
func ParseID(id int64) (idDetail IDDetail, err error) {
	if id < 83886079 {
		return idDetail, errors.New("id illegal,should> 83886079")
	}
	if id>>25&0x1 != 1 {
		idDetail.isCustom = false
		idDetail.sequence = id & 0xFF //8bit
		idDetail.ts = time.Unix(0, ((id>>26)+CEpoch)*flakeTimeUnit)
		idDetail.isTimeClash = (id&0x100 == 0x100)
		idDetail.nodeID = (id >> 9) & 0xffff
	} else {
		idDetail.isCustom = true
		idDetail.sequence = id & 0x3FFF //14bit
		idDetail.ts = time.Unix(0, ((id>>26)+CEpoch)*flakeTimeUnit)
		idDetail.isTimeClash = ((id>>14)&0x1 == 1)
		idDetail.nodeID = (id >> 15) & 0x3ff //10bit
	}
	return
}

func (idDetail IDDetail) String() string {
	var nodeStr string
	if idDetail.isCustom {
		nodeStr = fmt.Sprintf("%d", idDetail.nodeID)
	} else {
		nodeStr = fmt.Sprintf("%d.%d", idDetail.nodeID>>8&0xFF, idDetail.nodeID&0xFF)
	}
	return fmt.Sprintf("IDDetail-[isCustom:%v,nodeId:%d,nodeID_Str:%s,ts:%s,sequence:%d,isTimeClash:%v]", idDetail.isCustom, idDetail.nodeID, nodeStr, idDetail.ts.Format("2006-01-02 15:04:05.000"), idDetail.sequence, idDetail.isTimeClash)
}

// return 10ms unit
func (iw *IDWorker) timeGen() int64 {
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

// Encode62Str Func:int64 encode to string
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

// Decode62Str Func:string decode to int64
func Decode62Str(str62 string) (longNum int64, err error) {
	// max_int64  7m85Y0n8LzA
	length := len(str62)
	if length == 0 || length > 12 {
		err = errors.New("the length of input string must > 0")
		return
	}
	for index, char := range []byte(str62) {
		longNum += int64(strings.Index(ALPHABET, string(char))) * int64(math.Pow(62, float64(index)))
	}
	return
}

// Encode82Str Func:encode int64 to string
func Encode82Str(longNum int64) (str string, err error) {
	if longNum < 0 {
		err = errors.New("input num must >=0")
		return
	}
	result := make([]byte, 0)
	for longNum > 0 {
		round := longNum / 82
		remain := longNum % 82
		result = append(result, ALPHABET82[remain])
		longNum = round
	}
	if len(result) < 1 {
		err = errors.New("string join error")
		return
	}
	str = string(result)
	return
}

// Decode82Str Func:decode string to int64
func Decode82Str(str82 string) (longNum int64, err error) {
	// max_int64  7m85Y0n8LzA
	length := len(str82)
	if length == 0 || length > 11 {
		err = errors.New("the length of input string must > 0")
		return
	}
	for index, char := range []byte(str82) {
		longNum += int64(strings.Index(ALPHABET82, string(char))) * int64(math.Pow(82, float64(index)))
	}
	return
}
