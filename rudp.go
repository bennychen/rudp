package rudp

import (
	"encoding/binary"
	"fmt"
)

// the algorithm is based on http://blog.codingnow.com/2016/03/reliable_udp.html
// source c code is at https://github.com/cloudwu/rudp

const MaxPackageSize = (0x7fff - 4)
const MTU = 128 //512

type RUDP struct {
	sendQueue   messageQueue
	recvQueue   messageQueue
	sendHistroy messageQueue // keep message history in case we need to resend

	sendPackage *RUDPPackage // returned by rudp_update

	freeList  *message // recyclable messages
	sendAgain []int    // package id needs to send again

	corrupt         int
	currentTick     int
	lastSendTick    int
	lastExpiredTick int
	sendID          int
	recvIDMin       int
	recvIDMax       int
	sendDelay       int
	expired         int
}

type RUDPPackage struct {
	Next   *RUDPPackage
	Buffer []byte
	Size   int
}

func CreateRudp(sendDelay int, expiredTime int) *RUDP {
	u := &RUDP{}
	u.sendDelay = sendDelay
	u.expired = expiredTime
	u.sendAgain = make([]int, 0)
	return u
}

func (u *RUDP) Delete() {
	u.sendQueue.head = nil
	u.recvQueue.head = nil
	u.sendHistroy.head = nil
	u.freeList = nil
	u.sendAgain = nil
	u.clearOutPackage()
}

// Send sends a new package out
func (u *RUDP) Send(buffer []byte, sz int) {
	if sz > MaxPackageSize {
		fmt.Println("package size is too large.")
		return
	}
	m := u.createMessage(buffer, sz)
	u.sendID++
	m.id = u.sendID
	m.tick = u.currentTick
	u.sendQueue.push(m)
}

// Recv receives package and returns the size of the new package
// 0 = no new package
// -1 = corrupt connection
func (u *RUDP) Recv(buffer []byte) int {
	if u.corrupt != 0 {
		u.corrupt = 0
		return -1
	}
	m := u.recvQueue.pop(u.recvIDMin)
	if m == nil {
		return 0
	}
	u.recvIDMin++
	if m.sz > 0 {
		copy(buffer, m.buffer)
	}
	u.deleteMessage(m)
	return m.sz
}

// Update should be called every frame with the time tick,
// or when a new package is coming.
// buffer is the actual udp package we received
// sz is the size of the package
// the package returned from this function should be sent out.
func (u *RUDP) Update(received []byte, sz int, deltaTick int) *RUDPPackage {
	u.currentTick += deltaTick
	u.clearOutPackage()
	u.extractPackages(received, sz)

	if u.currentTick >= u.lastExpiredTick+u.expired {
		u.clearSendExpired(u.lastExpiredTick)
		u.lastExpiredTick = u.currentTick
	}
	if u.currentTick >= u.lastSendTick+u.sendDelay {
		u.sendPackage = u.genOutPackage()
		u.lastSendTick = u.currentTick
		return u.sendPackage
	}
	return nil
}

type message struct {
	next   *message
	buffer []byte
	sz     int
	id     int
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

func (q *messageQueue) pop(id int) *message {
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

func (u *RUDP) createMessage(buffer []byte, sz int) *message {
	msg := u.freeList
	if msg != nil {
		u.freeList = msg.next
		if len(msg.buffer) < sz {
			msg.buffer = make([]byte, sz)
		}
	}
	if msg == nil {
		msg = &message{}
		msg.buffer = make([]byte, sz)
	}

	msg.sz = sz
	copy(msg.buffer, buffer)
	msg.tick = 0
	msg.id = 0
	msg.next = nil
	return msg
}

func (u *RUDP) deleteMessage(m *message) {
	m.next = u.freeList
	u.freeList = m
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
		last.next = u.freeList
		u.freeList = u.sendHistroy.head
	}
	u.sendHistroy.head = m
	if m == nil {
		u.sendHistroy.tail = nil
	}
}

// TODO: understand and make sure this function works
func (u *RUDP) getID(buffer []byte) int {
	id := uint(binary.BigEndian.Uint16(buffer))
	// max id is 64k, if over 64k, id is back to zero
	id |= (uint(u.recvIDMax) & (^uint(0xffff)))
	// if the diff of id is larger than 32K(0x8000), then adjust the id
	if id < uint(u.recvIDMax)-0x8000 {
		id += 0x10000
	} else if id > uint(u.recvIDMax)+0x8000 {
		id -= 0x10000
	}
	return int(id)
}

func (u *RUDP) addRequest(id int) {
	u.sendAgain = append(u.sendAgain, id)
}

func (u *RUDP) addMissing(id int) {
	// TODO: understand this
	u.insertMessage(id, nil, -1)
}

func (u *RUDP) insertMessage(id int, buffer []byte, sz int) {
	if id < u.recvIDMin {
		fmt.Printf(
			"Failed to insert msg with id %v as it's less than current min id.", id)
		return
	}
	if id > u.recvIDMax || u.recvQueue.head == nil {
		m := u.createMessage(buffer, sz)
		m.id = id
		u.recvQueue.push(m)
		u.recvIDMax = id
	} else {
		m := u.recvQueue.head
		last := &u.recvQueue.head
		for {
			if m.id > id {
				tmp := u.createMessage(buffer, sz)
				tmp.id = id
				tmp.next = m
				*last = tmp
				return
			}
			last = &m.next
			m = m.next

			if m == nil {
				break
			}
		}
	}
}

const (
	TypeIgnore = iota
	TypeCorrupt
	TypeRequest
	TypeMissing
	TypeNormalHeaderSize
)

func (u *RUDP) extractPackages(buffer []byte, sz int) {
	for sz > 0 {
		tag := uint(buffer[0])
		// if tag is at [128, 32K], tag is 2 bytes
		// otherwise tag is 1 byte
		if tag > 127 {
			if sz <= 1 {
				u.corrupt = 1
				return
			}
			tag = uint(binary.BigEndian.Uint16(buffer)) & 0x7fff
			buffer = buffer[2:]
			sz -= 2
		} else {
			buffer = buffer[1:]
			sz--
		}

		switch tag {
		case TypeIgnore:
			if len(u.sendAgain) == 0 {
				u.sendAgain = append(u.sendAgain, u.recvIDMin)
			}
		case TypeCorrupt:
			u.corrupt = 1
			return
		case TypeRequest:
		case TypeMissing:
			// | tag | id |
			if sz < 2 {
				u.corrupt = 1
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
			// | data length (2 bytes) | id (2 bytes) | data |
			dataLength := int(tag - TypeNormalHeaderSize)
			if sz < dataLength+2 {
				u.corrupt = 1
				return
			}
			id := u.getID(buffer)
			u.insertMessage(id, buffer[2:], dataLength)
			buffer = buffer[2:]
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

func (u *RUDP) createPackage(tmp *tmpBuffer) {
	p := &RUDPPackage{}
	p.Next = nil
	p.Buffer = make([]byte, tmp.sz)
	p.Size = tmp.sz
	copy(p.Buffer, tmp.buffer[:tmp.sz])
	if tmp.tail == nil {
		tmp.tail = p
		tmp.head = p
	} else {
		tmp.tail.Next = p
		tmp.tail = p
	}
}

/*
	1. request missing ( lookup U->recv_queue )
	2. reply request ( U->send_again )
	3. send message ( U->send_queue )
	4. send heartbeat
*/
func (u *RUDP) genOutPackage() *RUDPPackage {
	tmp := &tmpBuffer{}
	tmp.buffer = make([]byte, MTU)

	u.requestMissing(tmp)
	u.replyRequest(tmp)
	u.sendMessage(tmp)

	if tmp.head == nil && tmp.sz == 0 {
		tmp.buffer[0] = TypeIgnore
		tmp.sz = 1
	}
	u.createPackage(tmp)
	return tmp.head
}

func (u *RUDP) requestMissing(tmp *tmpBuffer) {
	id := u.recvIDMin
	m := u.recvQueue.head
	for m != nil {
		if m.id > id {
			for i := id; i < m.id; i++ {
				u.packRequest(tmp, i, TypeRequest)
			}
		}
		id = m.id + 1
		m = m.next
	}
}

func (u *RUDP) replyRequest(tmp *tmpBuffer) {
	history := u.sendHistroy.head
	for i := 0; i < len(u.sendAgain); i++ {
		id := u.sendAgain[i]
		if id < u.recvIDMin {
			// already received, ignore
			continue
		}
		for {
			if history == nil || id < history.id {
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

	u.sendAgain = make([]int, 0)
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

func (u *RUDP) packRequest(tmp *tmpBuffer, id int, tag int) {
	sz := MTU - tmp.sz
	if sz < 3 {
		u.createPackage(tmp)
	}
	buffer := tmp.buffer[tmp.sz:]
	tmp.sz += u.fillHeader(buffer, tag, id)
}

func (u *RUDP) packMessage(tmp *tmpBuffer, m *message) {
	sz := MTU - tmp.sz
	if m.sz > MTU-4 {
		if tmp.sz > 0 {
			u.createPackage(tmp)
		}
		// big package
		sz = 4 + m.sz
		p := &RUDPPackage{}
		p.Next = nil
		p.Buffer = make([]byte, sz)
		p.Size = sz
		u.fillHeader(p.Buffer, m.sz+TypeNormalHeaderSize, m.id)
		copy(p.Buffer[4:], m.buffer[:m.sz])
		if tmp.tail == nil {
			tmp.head = p
			tmp.tail = p
		} else {
			tmp.tail.Next = p
			tmp.tail = p
		}
		return
	}
	if sz < 4+m.sz {
		u.createPackage(tmp)
	}
	buf := tmp.buffer[tmp.sz:]
	length := u.fillHeader(buf, m.sz+TypeNormalHeaderSize, m.id)
	tmp.sz += length + m.sz
	copy(buf[length:], m.buffer[:m.sz])
}

func (u *RUDP) fillHeader(buffer []byte, length int, id int) int {
	var sz int
	if length < 128 {
		buffer[0] = byte(length)
		sz = 1
	} else {
		buffer[0] = byte(((length & 0x7f00) >> 8) | 0x80)
		buffer[1] = byte(length & 0xff)
		sz = 2
	}
	binary.BigEndian.PutUint16(buffer[sz:], uint16(id))
	return sz + 2
}
