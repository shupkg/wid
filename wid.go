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

func SetDefault(node, epoch uint64) {
	Default.mu.Lock()
	defer Default.mu.Unlock()

	Default.node = node
	if epoch <= 0 {
		epoch = 1515000000
	}
	Default.epoch = epoch
}

func New(node uint64) *ID {
	return (&ID{}).Set(node, 0)
}

//53位， 兼容js
type ID struct {
	node  uint64 //节点
	epoch uint64 //纪元时间（序号时间起始）

	time     uint64 //时间
	sequence uint64 //序列号

	mu sync.Mutex
}

func (s *ID) Set(node uint64, epoch uint64) *ID {
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

func (s *ID) Generate() uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := uint64(time.Now().Unix())
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

func (s *ID) Build(unixTs uint64, seq uint64, node uint64) uint64 {
	return (unixTs-s.epoch)<<timeShift | seq<<sequenceShift | node
}

func (s *ID) Parse(id uint64) (unixTs uint64, seq uint64, node uint64) {
	unixTs = id>>timeShift + s.epoch
	seq = id>>sequenceShift - id>>timeShift<<sequenceBits
	node = id - id>>sequenceShift<<sequenceShift
	return
}

func (s *ID) ParseString(sid string, radix int) (unixTs uint64, seq uint64, node uint64) {
	if radix < 2 {
		radix = 36
	}
	i, _ := strconv.ParseUint(sid, radix, 64)
	return s.Parse(i)
}

func (s *ID) Format(id uint64, radix int) string {
	if radix < 2 {
		radix = 36
	}
	return strconv.FormatUint(id, radix)
}

func (s *ID) FormatHuman(id uint64) string {
	unixTs, seq, node := s.Parse(id)
	return fmt.Sprintf("%s%05d%02d", time.Unix(int64(unixTs), 0).In(locCST).Format(timeFormat), seq, node)
}

func (s *ID) ParseHumanString(hid string) (unixTs uint64, seq uint64, node uint64) {
	if len(hid) == 21 {
		tim, _ := time.ParseInLocation(timeFormat, hid[:14], locCST)
		unixTs = uint64(tim.Unix())
		seq, _ = strconv.ParseUint(hid[14:19], 10, 64)
		node, _ = strconv.ParseUint(hid[19:], 10, 64)
	}
	return
}
