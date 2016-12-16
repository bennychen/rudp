package rudp_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/bennychen/rudp"
)

func dumpRecv(u *rudp.RUDP) string {
	tmp := make([]byte, rudp.MaxPackageSize)
	n := u.Recv(tmp)
	str := ""
	for n != 0 {
		if n < 0 {
			str += "CORRUPT\n"
			break
		}

		str += "RECV "
		for i := 0; i < n; i++ {
			str += fmt.Sprintf("%v", tmp[i])
			if i < n-1 {
				str += " "
			}
		}
		str += "\n"

		n = u.Recv(tmp)
	}
	if str != "" {
		fmt.Printf(str)
	}
	return str
}

var idx int = 0

func dump(p *rudp.RUDPPackage) {
	fmt.Printf("%v: send ", idx)
	for p != nil {
		fmt.Printf("(")
		for i := 0; i < p.Size; i++ {
			fmt.Printf("%02x ", p.Buffer[i])
		}
		fmt.Printf(")")
		p = p.Next
	}
	fmt.Printf("\n")
	idx++
}

func TestCorrectPath(t *testing.T) {
	fmt.Println("=======================TestCorrectPath======================")
	U := rudp.Create(1, 5)
	U.MTU = 128

	t1 := []byte{1, 2, 3, 4}
	t2 := []byte{5, 6, 7, 8}
	t3 := []byte{
		2, 1, 1, 1, 1, 1, 1, 3, 2, 1, 1, 1, 1, 1, 1, 3,
		2, 1, 1, 1, 1, 1, 1, 3, 2, 1, 1, 1, 1, 1, 1, 3,
		2, 1, 1, 1, 1, 1, 1, 3, 2, 1, 1, 1, 1, 1, 1, 3,
		2, 1, 1, 1, 1, 1, 1, 3, 2, 1, 1, 1, 1, 1, 1, 3,
		2, 1, 1, 1, 1, 1, 1, 3, 2, 1, 1, 1, 1, 1, 1, 3,
		2, 1, 1, 1, 1, 1, 1, 3, 2, 1, 1, 1, 1, 1, 1, 3,
		2, 1, 1, 1, 1, 1, 1, 3, 2, 1, 1, 1, 1, 1, 1, 3,
		2, 1, 1, 1, 1, 1, 1, 3, 2, 1, 1, 1, 1, 1, 1, 3,
		2, 1, 1, 1, 1, 1, 1, 3, 2, 1, 1, 1, 1, 1, 1, 3,
		2, 1, 1, 1, 1, 1, 1, 3, 2, 1, 1, 1, 1, 1, 1, 3,
		2, 1, 1, 1, 1, 1, 1, 3, 2, 1, 1, 1, 1, 1, 1, 3,
		2, 1, 1, 1, 1, 1, 1, 3, 2, 1, 1, 1, 1, 1, 1, 3,
		2, 1, 1, 1, 1, 1, 1, 3, 2, 1, 1, 1, 1, 1, 1, 3,
		2, 1, 1, 1, 1, 1, 1, 3, 2, 1, 1, 1, 1, 1, 1, 3,
		2, 1, 1, 1, 1, 1, 1, 3, 2, 1, 1, 1, 1, 1, 1, 3,
		2, 1, 1, 1, 1, 1, 1, 3, 2, 1, 1, 1, 10, 11, 12, 13,
	} // 256 bytes
	t4 := []byte{4, 3, 2, 1}

	U.Send(t1, len(t1))
	U.Send(t2, len(t2))
	p := U.Update(nil, 0, 1)
	if p == nil || p.Next != nil {
		t.Error("RUDP::Update error, should only send one package.")
	}
	if bytes.Compare(
		p.Buffer, []byte{8, 0, 0, 1, 2, 3, 4, 8, 0, 1, 5, 6, 7, 8}) != 0 {
		t.Error("RUDP::Update error, data package to send is wrong.")
	}
	dump(p)
	p = U.Update(nil, 0, 1)
	if p == nil || p.Next != nil {
		t.Error("RUDP::Update error, should only send one package.")
	}
	if bytes.Compare(
		p.Buffer, []byte{0}) != 0 {
		t.Error("RUDP::Update error, should send a heartbeat package.")
	}
	dump(p)

	U.Send(t3, len(t3))
	U.Send(t4, len(t4))
	p = U.Update(nil, 0, 1)
	if p.Next == nil || p.Next.Next != nil {
		t.Error("RUDP::Update error, should send two packages.")
	}
	if p.Size != 260 || p.Buffer[3] != 2 {
		t.Error("RUDP::Update error, should send a package with id 2 and size 260.")
	}
	if bytes.Compare(
		p.Next.Buffer, []byte{8, 0, 3, 4, 3, 2, 1}) != 0 {
		t.Error("RUDP::Update error, data package to send is wrong.")
	}
	dump(p)

	r1 := []byte{
		02, 00, 00,
		02, 00, 03,
	}
	p = U.Update(r1, len(r1), 1)
	if p == nil || p.Next != nil {
		t.Error("RUDP::Update error, should only send one package.")
	}
	if bytes.Compare(
		p.Buffer, []byte{8, 0, 0, 1, 2, 3, 4, 8, 0, 3, 4, 3, 2, 1}) != 0 {
		t.Error("RUDP::Update error, data package to resend is wrong.")
	}
	dump(p)
	recvResult := dumpRecv(U)
	if recvResult != "" {
		t.Error("RUDP::Recv error, should receive nothing.")
	}

	r2 := []byte{
		5, 0, 1, 1,
		5, 0, 3, 3,
	}
	p = U.Update(r2, len(r2), 1)
	if p == nil || p.Next != nil {
		t.Error("RUDP::Update error, should only send one package.")
	}
	if bytes.Compare(
		p.Buffer, []byte{2, 0, 0, 2, 0, 2}) != 0 {
		t.Error("RUDP:Update error: should send 2 TypeRequest messages.")
	}
	dump(p)
	recvResult = dumpRecv(U)
	if recvResult != "" {
		t.Error("RUDP::Recv error, should receive nothing.")
	}

	r3 := []byte{
		5, 0, 0, 0,
		5, 0, 5, 5,
	}
	p = U.Update(r3, len(r3), 0)
	if p != nil {
		t.Error("RUDP::Update error, should send 0 package.")
	}
	dump(p)

	r4 := []byte{5, 0, 6, 6}
	p = U.Update(r4, len(r4), 1)
	if p == nil || p.Next != nil {
		t.Error("RUDP::Update error, should only send one package.")
	}
	if bytes.Compare(
		p.Buffer, []byte{2, 0, 2, 2, 0, 4}) != 0 {
		t.Error("RUDP:Update error: should send 2 TypeRequest message.")
	}
	dump(p)

	r5 := []byte{5, 0, 2, 2}
	p = U.Update(r5, len(r5), 1)
	if p == nil || p.Next != nil {
		t.Error("RUDP::Update error, should only send one package.")
	}
	if bytes.Compare(
		p.Buffer, []byte{2, 0, 4}) != 0 {
		t.Error("RUDP:Update error: should send 1 TypeRequest message.")
	}
	dump(p)

	recvResult = dumpRecv(U)
	if recvResult != "RECV 0\nRECV 1\nRECV 2\nRECV 3\n" {
		t.Error("RUDP:Recv error: should receive 0~3 messages.")
	}

	U = nil
}

func TestLargePackage(t *testing.T) {
	fmt.Println("=======================TestLargePackage======================")
	U := rudp.Create(1, 5)
	U.MTU = 128

	buf := make([]byte, rudp.MaxPackageSize)
	U.Send(buf, len(buf))
	p := U.Update(nil, 0, 1)
	if p == nil || p.Size != 0x7fff {
		t.Error("RUDP::Update error, should send a normal package.")
	}

	buf = make([]byte, rudp.MaxPackageSize+1)
	U.Send(buf, len(buf))
	p = U.Update(nil, 0, 1)
	if p == nil || p.Next != nil || bytes.Compare(p.Buffer, []byte{0}) != 0 {
		t.Error("RUDP::Update error, should send a heartbeat package.")
	}
}

func TestRecvHeartbeat(t *testing.T) {
	fmt.Println("=======================TestRecvHeartbeat======================")
	U := rudp.Create(1, 5)
	r := []byte{0}
	U.Update(r, len(r), 1)
	dumpRecv(U)
}

func TestRecvBigMessage(t *testing.T) {
	fmt.Println("=======================TestRecvBigMessage======================")

	r := []byte{
		0x81, 4, 0, 0,
		2, 1, 1, 1, 1, 1, 1, 3, 2, 1, 1, 1, 1, 1, 1, 3,
		2, 1, 1, 1, 1, 1, 1, 3, 2, 1, 1, 1, 1, 1, 1, 3,
		2, 1, 1, 1, 1, 1, 1, 3, 2, 1, 1, 1, 1, 1, 1, 3,
		2, 1, 1, 1, 1, 1, 1, 3, 2, 1, 1, 1, 1, 1, 1, 3,
		2, 1, 1, 1, 1, 1, 1, 3, 2, 1, 1, 1, 1, 1, 1, 3,
		2, 1, 1, 1, 1, 1, 1, 3, 2, 1, 1, 1, 1, 1, 1, 3,
		2, 1, 1, 1, 1, 1, 1, 3, 2, 1, 1, 1, 1, 1, 1, 3,
		2, 1, 1, 1, 1, 1, 1, 3, 2, 1, 1, 1, 1, 1, 1, 3,
		2, 1, 1, 1, 1, 1, 1, 3, 2, 1, 1, 1, 1, 1, 1, 3,
		2, 1, 1, 1, 1, 1, 1, 3, 2, 1, 1, 1, 1, 1, 1, 3,
		2, 1, 1, 1, 1, 1, 1, 3, 2, 1, 1, 1, 1, 1, 1, 3,
		2, 1, 1, 1, 1, 1, 1, 3, 2, 1, 1, 1, 1, 1, 1, 3,
		2, 1, 1, 1, 1, 1, 1, 3, 2, 1, 1, 1, 1, 1, 1, 3,
		2, 1, 1, 1, 1, 1, 1, 3, 2, 1, 1, 1, 1, 1, 1, 3,
		2, 1, 1, 1, 1, 1, 1, 3, 2, 1, 1, 1, 1, 1, 1, 3,
		2, 1, 1, 1, 1, 1, 1, 3, 2, 1, 1, 1, 10, 11, 12, 13,
	}

	U := rudp.Create(1, 5)
	U.Update(r, len(r), 1)
	dumpRecv(U)
}
