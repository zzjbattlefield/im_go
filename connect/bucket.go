package connect

import (
	"sync"

	"github.com/zzjbattlefield/IM_GO/proto"
)

type Bucket struct {
	lock       sync.RWMutex
	rooms      map[int]*Room
	chs        map[int]*Channel
	option     BucketOption
	routines   []chan *proto.PushRoomMessageReqeust //发送群组消息的channel 接收到消息后直接丢进redis里
	routineNum int64
	broadcast  chan []byte
}

type BucketOption struct {
	ChannelSize    int //chs的size
	RoomSize       int
	routinueAmount uint64 // 存放投递push消息的通道切片的数量
	routinueSize   int    // push消息通道的缓冲区大小
}

func NewBucket(option *BucketOption) (bucket *Bucket) {
	bucket = &Bucket{
		rooms:    make(map[int]*Room),
		chs:      make(map[int]*Channel, option.ChannelSize),
		option:   *option,
		routines: make([]chan *proto.PushRoomMessageReqeust, option.routinueAmount),
	}
	for i := uint64(0); i < option.routinueAmount; i++ {
		messageChan := make(chan *proto.PushRoomMessageReqeust, option.routinueSize)
		bucket.routines[i] = messageChan
		go bucket.pushRoom(messageChan)
	}
	return
}

// 将消息发送到指定的房间
func (b *Bucket) pushRoom(ch chan *proto.PushRoomMessageReqeust) {
	for {
		var (
			arg  *proto.PushRoomMessageReqeust
			room *Room
		)
		arg = <-ch
		if room = b.Room(arg.RoomID); room != nil {
			room.Push(&arg.Msg)
		}
	}

}

func (b *Bucket) Room(rid int) (room *Room) {
	b.lock.RLock()
	defer b.lock.RUnlock()
	return b.rooms[rid]
}

func (bucket *Bucket) DeleteChannel(ch *Channel) {
	var (
		ok   bool
		room *Room
	)
	bucket.lock.RLock()
	defer bucket.lock.RUnlock()
	if ch, ok = bucket.chs[ch.UserID]; ok {
		room = ch.Room
		delete(bucket.chs, ch.UserID)
	}
	if room != nil && room.DeleteChannel(ch) {
		//也要把room里的这个channel给删掉
		if room.drop {
			delete(bucket.rooms, room.ID)
		}
	}
}

func (bucket *Bucket) Put(userID int, roomID int, ch *Channel) (err error) {
	var (
		room *Room
		ok   bool
	)
	bucket.lock.Lock()
	if roomID != noRoom {
		if room, ok = bucket.rooms[roomID]; !ok {
			//创建新房间
			room = NewRoom(roomID)
			bucket.rooms[roomID] = room
		}
		ch.Room = room
	}
	ch.UserID = userID
	bucket.chs[userID] = ch
	bucket.lock.Unlock()

	if room != nil {
		room.Put(ch)
	}
	return
}
