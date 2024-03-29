package idgen

import (
	"errors"
	"math/rand"
	"testing"
	"time"
)

var idWorker *IDWorker

func init() {
	idWorker, _ = NewNodeIDByIP()
}

func TestDecode62Str(t *testing.T) {
	str62 := "7m85Y0n8LzA"
	rs, err := Decode62Str(str62)
	if err != nil {
		t.Errorf("解码失败")
	}
	if rs != int64(^uint(0)>>1) {
		t.Errorf("解码失败")
	}
	t.Logf("62进制字符串 %s 10进制解码为 %v\n", str62, rs)

	rsStr, err := Encode62Str(rs)
	if err != nil {
		t.Errorf("编码失败")
	}

	if rsStr != str62 {
		t.Errorf("编码失败")
	}
	t.Logf("10进制数字 %v 编码为62进制字符串 %s\n", rs, rsStr)

}
func TestEncode82Str(t *testing.T) {
	rs, _ := idWorker.NextID()
	rsStr, err := Encode82Str(rs)
	if err != nil {
		t.Errorf("编码失败")
	}
	t.Logf("10进制数字 %v 编码为82进制字符串 %s\n", rs, rsStr)

}

func TestEncodeAndDecode(t *testing.T) {
	length := 10000
	var err error
	for i := 0; i < length; i++ {
		err = encodeAndDecode(rand.Int63())
	}
	if err != nil {
		t.Errorf("%s", err)
	} else {
		t.Logf("随机编码解码%v个数字成功", length)
	}
}

func encodeAndDecode(longNum int64) (err error) {
	rsStr, err := Encode62Str(longNum)
	if err != nil {
		err = errors.New("编码失败")
		return
	}
	rsNum, err := Decode62Str(rsStr)
	if err != nil {
		err = errors.New("解码失败")
		return
	}
	if rsNum != longNum {
		err = errors.New("编码解码不匹配")
		return
	}
	return
}
func encodeAndDecode82(longNum int64) (err error) {
	rsStr, err := Encode82Str(longNum)
	if err != nil {
		err = errors.New("编码失败")
		return
	}
	rsNum, err := Decode82Str(rsStr)
	if err != nil {
		err = errors.New("解码失败")
		return
	}
	if rsNum != longNum {
		err = errors.New("编码解码不匹配")
		return
	}
	return
}
func TestIDWorker_FromIp(t *testing.T) {
	iw, err := NewNodeIDByIP()
	if err != nil {
		t.Errorf("%s\n", err)
	}
	myWorkID := iw.nodeID
	if err != nil {
		t.Errorf("%s\n", err)
	}
	id, err := iw.NextID()
	if err != nil {
		t.Errorf("%s\n", err)
	}

	iDDetail, _ := ParseID(id)
	seq1 := iDDetail.sequence

	if myWorkID != iDDetail.nodeID {
		t.Errorf("id产生和解释不一致")
	}
	t.Logf("iDDetail 1:%v", iDDetail)
	id2, _ := iw.NextID()
	iDDetail, _ = ParseID(id2)
	t.Logf("iDDetail 2:%v", iDDetail)
	if myWorkID != iDDetail.nodeID {
		t.Errorf("id产生和解释不一致")
	}
	if iDDetail.sequence != seq1+1 {
		t.Errorf("id产生和解释不一致")
	}
}
func TestIDWorker_NextID(t *testing.T) {
	var myWorkID int64 = 5
	iw, err := NewCustomNodeID(myWorkID)
	if err != nil {
		t.Errorf("%s\n", err)
	}

	id, err := iw.NextID()
	if err != nil {
		t.Errorf("%s\n", err)
	}

	iDDetail, _ := ParseID(id)
	seq1 := iDDetail.sequence

	if myWorkID != iDDetail.nodeID {
		t.Errorf("id产生和解释不一致")
	}
	t.Logf("iDDetail 1:%v", iDDetail)
	id2, _ := iw.NextID()
	iDDetail, _ = ParseID(id2)
	t.Logf("iDDetail 2:%v", iDDetail)
	if myWorkID != iDDetail.nodeID {
		t.Errorf("id产生和解释不一致")
	}
	if iDDetail.sequence != seq1+1 {
		t.Errorf("id产生和解释不一致")
	}
}

var iw *IDWorker

func TestIDGenByNodeIP(t *testing.T) {
	//var id int64
	iw, _ = NewNodeIDByIP()
	start := time.Now()
	for i := 0; i < 100000; i++ {
		iw.NextID()
	}
	cost := time.Since(start).Milliseconds()
	t.Logf("id gen 1000000 times,cost:%dms,speed:%d/s", cost, 100000*1000/cost)
}
func BenchmarkIDWorker_Custom(b *testing.B) {
	var myWorkID int64 = 7
	//var id int64 = 0
	var id int64
	b.StopTimer()
	iw, _ = NewCustomNodeID(int64(myWorkID))
	b.StartTimer()
	id, _ = iw.NextID()
	if id < MinNum {
		b.Errorf("id产生错误:%v", id)
	}
	// b.StopTimer()
}
func BenchmarkIDWorker_IP(b *testing.B) {
	//var id int64 = 0
	var id int64
	b.StopTimer()
	iw, _ = NewNodeIDByIP()
	b.StartTimer()
	id, _ = iw.NextID()
	if id < MinNum {
		b.Errorf("id产生错误:%v", id)
	}
	// b.StopTimer()
}

//:!go test -bench="^BenchmarkID" -benchtime=5s -benchmem
//:!go test -bench="^BenchmarkID" -count=10 -benchmem
