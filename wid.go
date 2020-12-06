package wid

import (
	"fmt"
	"strconv"
	"sync"
	"time"
)

/*
关于节点位长度取值问题(不借秒的情况下)
节点位  支持节点数  每节点最大秒并发
 0     1         2097152
 1     2         1048576
 2     4         524288
 3     8         262144
 4     16        131072
 5     32        65536
 6     64        32768
 7     128       16384
 8     256       8192
 9     512       4096
10     1024      2048
*/

//32位秒 + 17位自增 + 4位机器 = 53位
const (
	timeShift     = 21
	nodeBits      = 5
	sequenceBits  = timeShift - nodeBits
	sequenceShift = nodeBits

	nodeMask     = -1 ^ (-1 << nodeBits)
	sequenceMask = -1 ^ (-1 << sequenceBits)

	timeFormat = "20060102150405"
)

var locCST = time.FixedZone("CST", 28800)

var Default = New(0)

func SetDefault(node, epoch int64) {
	Default.mu.Lock()
	defer Default.mu.Unlock()

	Default.node = node
	if epoch <= 0 {
		epoch = 1515000000
	}
	Default.epoch = epoch
}

func New(node int64) *ID {
	return (&ID{}).Set(node, 0)
}

//53位， 兼容js
type ID struct {
	node  int64 //节点
	epoch int64 //纪元时间（序号时间起始）

	time     int64 //时间
	sequence int64 //序列号

	mu sync.Mutex
}

func (s *ID) Set(node, epoch int64) *ID {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.node = node & nodeMask //如果超限设为0
	if epoch <= 0 {
		//2018-01-04T01:20:00+08:00 //为何取这个时间，想在2018年初找一个好记的时间戳, 这个点挺好，1515+6个0
		epoch = 1515000000
	}
	s.epoch = epoch
	return s
}

func (s *ID) Generate() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().Unix()
	seq := s.sequence
	if now <= s.time {
		if now < s.time {
			now = s.time
		}
		//达到最大值，借秒
		if seq = (seq + 1) & sequenceMask; seq == 0 {
			now++
		}
	} else {
		seq = 0
	}
	s.time = now
	s.sequence = seq
	return s.Build(now, seq, s.node)
}

func (s *ID) Build(unixTs int64, seq int64, node int64) int64 {
	return (unixTs-s.epoch)<<timeShift | seq<<sequenceShift | node
}

func (s *ID) Parse(id int64) (unixTs int64, seq int64, node int64) {
	unixTs = id>>timeShift + s.epoch
	seq = id>>sequenceShift - id>>timeShift<<sequenceBits
	node = id - id>>sequenceShift<<sequenceShift
	return
}

func (s *ID) ParseString(sid string, radix int) (unixTs int64, seq int64, node int64) {
	if radix < 2 {
		radix = 36
	}
	i, _ := strconv.ParseInt(sid, radix, 64)
	return s.Parse(i)
}

func (s *ID) Format(id int64, radix int) string {
	if radix < 2 {
		radix = 36
	}
	return strconv.FormatInt(id, radix)
}

func (s *ID) FormatHuman(id int64) string {
	unixTs, seq, node := s.Parse(id)
	return fmt.Sprintf("%s%05d%02d", time.Unix(unixTs, 0).In(locCST).Format(timeFormat), seq, node)
}

func (s *ID) ParseHumanString(hid string) (unixTs int64, seq int64, node int64) {
	if len(hid) == 21 {
		tim, _ := time.ParseInLocation(timeFormat, hid[:14], locCST)
		unixTs = tim.Unix()
		seq, _ = strconv.ParseInt(hid[14:19], 10, 64)
		node, _ = strconv.ParseInt(hid[19:], 10, 64)
	}
	return
}
