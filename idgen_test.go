package idgen

import (
	"errors"
	"math/rand"
	"testing"
)

func TestDecode64Str(t *testing.T) {
	str62 := "7m85Y0n8LzA"
	rs, err := Decode64Str(str62)
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

func TestEncodeAndDecode(t *testing.T) {
	length := 10000
	var err error
	for i := 0; i < length; i++ {
		err = EncodeAndDecode(rand.Int63())
	}
	if err != nil {
		t.Errorf("%s", err)
	} else {
		t.Logf("随机编码解码%v个数字成功", length)
	}

}

func EncodeAndDecode(longNum int64) (err error) {
	rsStr, err := Encode62Str(longNum)
	if err != nil {
		err = errors.New("编码失败")
		return
	}
	rsNum, err := Decode64Str(rsStr)
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
func TestIdWorker_FromIp(t *testing.T) {
	iw, err := NewNodeIdByIp()
	if err != nil {
		t.Errorf("%s\n", err)
	}
	myWorkId := iw.nodeId
	if err != nil {
		t.Errorf("%s\n", err)
	}
	id, err := iw.NextId()
	if err != nil {
		t.Errorf("%s\n", err)
	}
	_, _, workId, seq1, _ := ParseId(id)
	if myWorkId != workId {
		t.Errorf("id产生和解释不一致")
	}
	id2, _ := iw.NextId()
	time1, lastTimeStamp, workId, seq2, _ := ParseId(id2)
	if myWorkId != workId {
		t.Errorf("id产生和解释不一致")
	}
	if seq2 != seq1+1 {
		t.Errorf("id产生和解释不一致")
	}
	const shortForm = "2006-01-02 15:04:03 +0800 CST"
	//local, _ := time.LoadLocation("Asia/Shanghai") //上海
	//t.Logf("lastTimeStamp:%v,当前时间:%s\n",lastTimeStamp,time1.In(local).Format(shortForm))
	t.Logf("lastTimeStamp:%v,当前时间:%s\n", lastTimeStamp, time1.Format(shortForm))
	t.Logf("workId:%v,seq1:%v,seq2:%v\n", workId, seq1, seq2)
	for i := 0; i < 0x1FF-1; i++ {
		id2, _ = iw.NextId()
	}
	if id2 != id+0x1FF {
		t.Errorf("id产生错误,id:%v,id2:%v\n", id, id2)
	}
	id2, _ = iw.NextId()
	_, lastTimeStamp3, workId3, seq3, isCustomWorkId := ParseId(id2)
	if lastTimeStamp3 != (lastTimeStamp+1) || workId3 != workId || seq3 != 0 {
		t.Errorf("id产生错误,lastTimeStamp3:%v,workId3:%v,seq3:%v,isCustomWorkId:%v\n", lastTimeStamp3, workId3, seq3, isCustomWorkId)
	}
	t.Logf("lastTimeStamp3:%v,workId3:%v,seq3:%v,isCustomWorkId:%v\n", lastTimeStamp3, workId3, seq3, isCustomWorkId)
}
func TestIdWorker_NextId(t *testing.T) {
	var myWorkId int64 = 5
	iw, err := NewCustomNodeId(myWorkId)
	if err != nil {
		t.Errorf("%s\n", err)
	}
	id, err := iw.NextId()
	if err != nil {
		t.Errorf("%s\n", err)
	}
	_, _, workId, seq1, _ := ParseId(id)
	if myWorkId != workId {
		t.Errorf("id产生和解释不一致")
	}
	id2, _ := iw.NextId()
	time1, lastTimeStamp, workId, seq2, _ := ParseId(id2)
	if myWorkId != workId {
		t.Errorf("id产生和解释不一致,myWorkId:%v,workId:%v\n", myWorkId, workId)
	}
	if seq2 != seq1+1 {
		t.Errorf("id产生和解释不一致")
	}
	const shortForm = "2006-01-02 15:04:03 +0800 CST"
	//local, _ := time.LoadLocation("Asia/Shanghai") //上海
	//t.Logf("lastTimeStamp:%v,当前时间:%s\n",lastTimeStamp,time1.In(local).Format(shortForm))
	t.Logf("lastTimeStamp:%v,当前时间:%s\n", lastTimeStamp, time1.Format(shortForm))
	t.Logf("workId:%v,seq1:%v,seq2:%v\n", workId, seq1, seq2)
	for i := 0; i < 0x3FFF-1; i++ {
		id2, _ = iw.NextId()
	}
	if id2 != id+0x3FFF {
		t.Errorf("id产生错误,id:%v,id2:%v\n", id, id2)
	}
	id2, _ = iw.NextId()
	_, lastTimeStamp3, workId3, seq3, isCustomWorkId := ParseId(id2)
	if lastTimeStamp3 != (lastTimeStamp+1) || workId3 != workId || seq3 != 0 {
		t.Errorf("id产生错误,lastTimeStamp3:%v,workId3:%v,seq3:%v,isCustomWorkId:%v\n", lastTimeStamp3, workId3, seq3, isCustomWorkId)
	}
	t.Logf("lastTimeStamp3:%v,workId3:%v,seq3:%v,isCustomWorkId:%v\n", lastTimeStamp3, workId3, seq3, isCustomWorkId)
	id2Str, _ := Encode62Str(id2)
	t.Logf("id2 to62str:%v\n", id2Str)

}

func BenchmarkIdWorker_NextId(b *testing.B) {
	var myWorkId int64 = 7
	var id int64 = 0
	iw, _ := NewCustomNodeId(int64(myWorkId))
	b.StopTimer()
	b.StartTimer()
	id, _ = iw.NextId()
	if id < 1 {
		b.Errorf("id产生错误:%v", id)
	}
}

//func TestTimeBack (t *testing.T){
//   iw, _:= NewNodeIdByIpAndTimeBackInterval(int64(6000))// 60 secondes
//   for i:=0;i<1e7;i++{
//     _,err:=iw.NextId()
//     if err!=nil{
//        t.Error(err)
//        return
//     }
//   }
//}
