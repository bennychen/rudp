package rudp

import (
	"encoding/binary"
	"fmt"
)

// the algorithm is based on http://blog.codingnow.com/2016/03/reliable_udp.html
// source c code is at https://github.com/cloudwu/rudp

const MaxPackageSize = (0x7fff - 4)

const (
	TypeHeartbeat = iota // provider sends heartbeat to consumer to keep alive
	TypeCorrupt          // abnormal corrupted message
	TypeRequest          // consumer requests provider to resend message
	TypeMissing          // provider tells consumer that some message is missing
	TypeNormal           // provider sends normal message to consumer
)

type RUDP struct {
	SendDelay   int // after how long we should send messages
	ExpiredTime int // after how long messages in history should be cleared

	mtu         int // maximum transmission unit size, recommended value 512
	sendQueue   messageQueue
	recvQueue   messageQueue
	sendHistroy messageQueue // keep message history in case we need to resend
	messagePool *message

	sendPackage *RUDPPackage // returned by RUDP::Update
	sendAgain   []uint16     // package ids to send again

	corrupt          bool
	currentTick      int
	lastSendTick     int
	lastExpiredTick  int
	currentSendID    uint16
	currentRecvIDMin uint16
	currentRecvIDMax uint16
}

type RUDPPackage struct {
	Next   *RUDPPackage
	Buffer []byte
	Size   int
}

func Create(sendDelay int, expiredTime int, mtu int) *RUDP {
	u := &RUDP{}
	u.mtu = mtu
	if u.mtu < 128 {
		u.mtu = 128
	}
	u.SendDelay = sendDelay
	u.ExpiredTime = expiredTime
	u.sendAgain = make([]uint16, 0)
	return u
}

// Send sends a new package out
func (u *RUDP) Send(buffer []byte, sz int) {
	if sz > MaxPackageSize {
		fmt.Println("package size is too large.")
		return
	}
	if sz > len(buffer) {
		sz = len(buffer)
	}
	m := u.createMessage(buffer, sz)
	m.id = u.currentSendID
	u.currentSendID++
	m.tick = u.currentTick
	u.sendQueue.push(m)
}

// Recv receives package and returns the size of the new package
// 0 = no new package
// -1 = corrupted connection
func (u *RUDP) Recv(buffer []byte) int {
	if u.corrupt {
		u.corrupt = false
		return -1
	}
	m := u.recvQueue.pop(u.currentRecvIDMin)
	if m == nil {
		return 0
	}
	u.currentRecvIDMin++
	if m.sz > 0 {
		copy(buffer, m.buffer)
	}
	u.deleteMessage(m)
	return m.sz
}

// Update should be called every frame with the time tick,
// or when a new package is coming.
// received is the actual udp package we received
// sz is the size of the package
// the package returned from this function should be sent out.
func (u *RUDP) Update(received []byte, sz int, deltaTick int) *RUDPPackage {
	u.currentTick += deltaTick
	u.clearOutPackage()
	if sz > len(received) {
		sz = len(received)
	}
	u.extractPackages(received, sz)

	if u.currentTick >= u.lastExpiredTick+u.ExpiredTime {
		u.clearSendExpired(u.lastExpiredTick)
		u.lastExpiredTick = u.currentTick
	}
	if u.currentTick >= u.lastSendTick+u.SendDelay {
		u.sendPackage = u.genOutPackage()
		u.lastSendTick = u.currentTick
		return u.sendPackage
	}
	return nil
}

func (u *RUDP) DebugGetPoolSize() int {
	n := 0
	m := u.messagePool
	for m != nil {
		n++
		m = m.next
	}
	return n
}

type message struct {
	next   *message
	buffer []byte
	sz     int
	id     uint16
	tick   int
}

type messageQueue struct {
	head *message
	tail *message
}

func (q *messageQueue) push(m *message) {
	if q.tail == nil {
		q.head = m
		q.tail = m
	} else {
		q.tail.next = m
		q.tail = m
	}
}

func (q *messageQueue) pop(id uint16) *message {
	if q.head == nil {
		return nil
	}
	m := q.head
	if m.id != id {
		return nil
	}
	q.head = m.next
	m.next = nil
	if q.head == nil {
		q.tail = nil
	}
	return m
}

func (u *RUDP) clearOutPackage() {
	u.sendPackage = nil
}

// createMessage is called in following 2 cases
// 1. when sending message, we create message and push to sendQueue
// 2. when message received, we create messages and push to recvQueue
func (u *RUDP) createMessage(buffer []byte, sz int) *message {
	msg := u.messagePool
	if msg != nil {
		u.messagePool = msg.next
		if len(msg.buffer) < sz {
			msg.buffer = make([]byte, sz)
		}
	} else {
		msg = &message{}
		if sz > 0 {
			msg.buffer = make([]byte, sz)
		} else {
			msg.buffer = make([]byte, u.mtu)
		}
	}

	msg.sz = sz
	if buffer != nil {
		copy(msg.buffer, buffer)
	}
	msg.tick = 0
	msg.id = 0
	msg.next = nil
	return msg
}

func (u *RUDP) deleteMessage(m *message) {
	m.next = u.messagePool
	u.messagePool = m
}

func (u *RUDP) clearSendExpired(tick int) {
	m := u.sendHistroy.head
	var last *message
	for m != nil {
		if m.tick >= tick {
			break
		}
		last = m
		m = m.next
	}

	if last != nil {
		// free all the messages before tick
		last.next = u.messagePool
		u.messagePool = u.sendHistroy.head
	}
	u.sendHistroy.head = m
	if m == nil {
		u.sendHistroy.tail = nil
	}
}

func compareID(srcID uint16, destID uint16) int {
	src := int(srcID)
	dest := int(destID)
	if src-dest > 0x8000 || src-dest < -0x8000 {
		return dest - src
	}
	return src - dest
}

func (u *RUDP) getID(buffer []byte) uint16 {
	return binary.BigEndian.Uint16(buffer)
}

func (u *RUDP) addRequest(id uint16) {
	u.sendAgain = append(u.sendAgain, id)
}

func (u *RUDP) addMissing(id uint16) {
	u.insertMessageToRecvQueue(id, nil, -1)
}

func (u *RUDP) insertMessageToRecvQueue(id uint16, buffer []byte, sz int) {
	if compareID(id, u.currentRecvIDMin) < 0 {
		fmt.Printf(
			"Failed to insert msg with id %v as it's less than current min id.\n", id)
		return
	}
	if compareID(id, u.currentRecvIDMax) > 0 || u.recvQueue.head == nil {
		m := u.createMessage(buffer, sz)
		m.id = id
		u.recvQueue.push(m)
		u.currentRecvIDMax = id
	} else {
		m := u.recvQueue.head
		last := &u.recvQueue.head
		for {
			if compareID(m.id, id) > 0 {
				tmp := u.createMessage(buffer, sz)
				tmp.id = id
				tmp.next = m
				*last = tmp
				return
			} else if m.id == id {
				// Duplicated message
				return
			}
			last = &m.next
			m = m.next

			if m == nil {
				fmt.Println("Error::Should never be here unless bug.")
				break
			}
		}
	}
}

func (u *RUDP) extractPackages(buffer []byte, sz int) {
	for sz > 0 {
		tag := uint16(buffer[0])
		// if tag is at [128, 32K], tag is 2 bytes
		// otherwise tag is 1 byte
		if tag > 127 {
			if sz <= 1 {
				u.corrupt = true
				return
			}
			tag = binary.BigEndian.Uint16(buffer) - 0x8000
			buffer = buffer[2:]
			sz -= 2
		} else {
			buffer = buffer[1:]
			sz--
		}

		switch tag {
		case TypeHeartbeat:
			if len(u.sendAgain) == 0 {
				// request next package id
				u.sendAgain = append(u.sendAgain, u.currentRecvIDMin)
			}
		case TypeCorrupt:
			u.corrupt = true
			return
		case TypeRequest, TypeMissing:
			// | tag (1 byte) | id (2 bytes) |
			if sz < 2 {
				u.corrupt = true
				return
			}
			id := u.getID(buffer)
			if tag == TypeRequest {
				u.addRequest(id)
			} else {
				u.addMissing(id)
			}
			buffer = buffer[2:]
			sz -= 2
		default:
			// | tag (1~2 bytes) | id (2 bytes) | data |
			// data is at least 1 byte, so general msg's tag starts from 1
			dataLength := int(tag - TypeNormal)
			if sz < dataLength+2 {
				u.corrupt = true
				return
			}
			id := u.getID(buffer)
			u.insertMessageToRecvQueue(id, buffer[2:], dataLength)
			buffer = buffer[dataLength+2:]
			sz -= dataLength + 2
		}
	}
}

type tmpBuffer struct {
	buffer []byte
	sz     int
	head   *RUDPPackage
	tail   *RUDPPackage
}

func (tmp *tmpBuffer) createEmptyPackage(sz int) *RUDPPackage {
	p := &RUDPPackage{}
	p.Next = nil
	p.Buffer = make([]byte, sz)
	p.Size = sz
	if tmp.tail == nil {
		tmp.tail = p
		tmp.head = p
	} else {
		tmp.tail.Next = p
		tmp.tail = p
	}
	return p
}

func (tmp *tmpBuffer) createPackageFromBuffer() *RUDPPackage {
	p := tmp.createEmptyPackage(tmp.sz)
	copy(p.Buffer, tmp.buffer[:tmp.sz])
	tmp.sz = 0
	return p
}

/*
	1. request missing ( lookup RUDP::recvQueue )
	2. reply request ( RUDP::sendAgain )
	3. send message ( RUDP::sendQueue )
	4. send heartbeat
*/
func (u *RUDP) genOutPackage() *RUDPPackage {
	tmp := &tmpBuffer{}
	tmp.buffer = make([]byte, u.mtu)

	u.requestMissing(tmp)
	u.replyRequest(tmp)
	u.sendMessage(tmp)

	if tmp.head == nil && tmp.sz == 0 {
		tmp.buffer[0] = TypeHeartbeat
		tmp.sz = 1
	}
	if tmp.sz > 0 {
		tmp.createPackageFromBuffer()
	}
	return tmp.head
}

// consumer requests missing packets
func (u *RUDP) requestMissing(tmp *tmpBuffer) {
	id := u.currentRecvIDMin
	m := u.recvQueue.head
	for m != nil {
		if compareID(m.id, id) > 0 {
			for i := id; compareID(i, m.id) < 0; i++ {
				u.packRequest(tmp, i, TypeRequest)
			}
		}
		id = m.id + 1
		m = m.next
	}
}

// provider replies missing packets requests from consumer
func (u *RUDP) replyRequest(tmp *tmpBuffer) {
	history := u.sendHistroy.head
	for i := 0; i < len(u.sendAgain); i++ {
		id := u.sendAgain[i]
		for {
			if history == nil || compareID(id, history.id) < 0 {
				// expired
				u.packRequest(tmp, id, TypeMissing)
				break
			} else if id == history.id {
				u.packMessage(tmp, history)
				break
			}
			history = history.next
		}
	}

	u.sendAgain = make([]uint16, 0)
}

func (u *RUDP) sendMessage(tmp *tmpBuffer) {
	m := u.sendQueue.head
	for m != nil {
		u.packMessage(tmp, m)
		m = m.next
	}

	if u.sendQueue.head != nil {
		if u.sendHistroy.tail == nil {
			u.sendHistroy = u.sendQueue
		} else {
			u.sendHistroy.tail.next = u.sendQueue.head
			u.sendHistroy.tail = u.sendQueue.tail
		}
		u.sendQueue.head = nil
		u.sendQueue.tail = nil
	}
}

func (u *RUDP) packRequest(tmp *tmpBuffer, id uint16, tag int) {
	sz := u.mtu - tmp.sz
	if sz < 3 {
		tmp.createPackageFromBuffer()
	}
	buffer := tmp.buffer[tmp.sz:]
	tmp.sz += u.fillHeader(buffer, tag, id)
}

func (u *RUDP) packMessage(tmp *tmpBuffer, m *message) {
	if m.sz > u.mtu-4 {
		if tmp.sz > 0 {
			tmp.createPackageFromBuffer()
		}
		// big package
		sz := 4 + m.sz
		p := tmp.createEmptyPackage(sz)
		p.Next = nil
		p.Buffer = make([]byte, sz)
		p.Size = sz
		u.fillHeader(p.Buffer, m.sz+TypeNormal, m.id)
		copy(p.Buffer[4:], m.buffer[:m.sz])
		return
	}
	// the remaining size is not enough to hold the message
	if u.mtu-tmp.sz < 4+m.sz {
		tmp.createPackageFromBuffer()
	}
	buf := tmp.buffer[tmp.sz:]
	length := u.fillHeader(buf, m.sz+TypeNormal, m.id)
	copy(buf[length:], m.buffer[:m.sz])
	tmp.sz += length + m.sz
}

func (u *RUDP) fillHeader(buffer []byte, length int, id uint16) int {
	var sz int
	if length < 128 {
		buffer[0] = byte(length)
		sz = 1
	} else {
		binary.BigEndian.PutUint16(buffer, uint16(length)+0x8000)
		sz = 2
	}
	binary.BigEndian.PutUint16(buffer[sz:], uint16(id))
	return sz + 2
}
