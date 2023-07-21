package idgen

import (
	"errors"
	"math/rand"
	"testing"
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
	const shortForm = "2006-01-02 15:04:03 +0800 CST"
	for i := 0; i < 0xFF-1; i++ {
		id2, _ = iw.NextID()
	}
	if id2 != id+0xFF {
		t.Errorf("id产生错误,id:%v,id2:%v\n", id, id2)
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
	//const shortForm = "2006-01-02 15:04:03 +0800 CST"

	for i := 0; i < 0xFF-1; i++ {
		id2, _ = iw.NextID()
	}
	if id2 != id+0xFF {
		t.Errorf("id产生错误,id:%v,id2:%v\n", id, id2)
	}

}

func BenchmarkIDWorker_NextID(b *testing.B) {
	var myWorkID int64 = 7
	//var id int64 = 0
	var id int64
	iw, _ := NewCustomNodeID(int64(myWorkID))
	//b.StopTimer()
	//b.StartTimer()
	for n := 0; n < b.N; n++ {
		id, _ = iw.NextID()
		if id < 1 {
			b.Errorf("id产生错误:%v", id)
		}
	}
}

//func TestTimeBack (t *testing.T){
//   iw, _:= NewNodeIDByIpAndTimeBackInterval(int64(6000))// 60 secondes
//   for i:=0;i<1e7;i++{
//     _,err:=iw.NextID()
//     if err!=nil{
//        t.Error(err)
//        return
//     }
//   }
//}
