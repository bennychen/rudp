package rudp_test

import (
	"fmt"
	"testing"

	"github.com/bennychen/rudp"
)

func dumpRecv(u *rudp.RUDP) {
	tmp := make([]byte, rudp.MaxPackage)
	n := u.Recv(tmp)
	for n != 0 {
		if n < 0 {
			fmt.Println("CORRUPT")
			break
		}

		fmt.Println("RECV ")
		for i := 0; i < n; i++ {
			fmt.Println(tmp[i])
		}
		fmt.Println("")

		n = u.Recv(tmp)
	}
}

var idx int = 0

func dump(p *rudp.RUDPPackage) {
	idx++
	fmt.Printf("%v: ", idx)
	for p != nil {
		fmt.Printf("(")
		for i := 0; i < p.Size; i++ {
			fmt.Printf("%02x ", p.Buffer[i])
		}
		fmt.Printf(") ")
		p = p.Next
	}
	fmt.Printf("\n")
}

func TestRudp(t *testing.T) {
	a := 0xaa
	fmt.Printf("negation of %b = %b\n", a, ^uint(a))

	U := rudp.CreateRudp(1, 5)

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
	}
	t4 := []byte{4, 3, 2, 1}

	U.Send(t1, len(t1))
	U.Send(t2, len(t2))
	dump(U.Update(nil, 0, 1))
	dump(U.Update(nil, 0, 1))
	U.Send(t3, len(t3))
	U.Send(t4, len(t4))
	dump(U.Update(nil, 0, 1))

	r1 := []byte{02, 00, 00, 02, 00, 03}
	dump(U.Update(r1, len(r1), 1))
	dumpRecv(U)
	r2 := []byte{5, 0, 1, 1,
		5, 0, 3, 3,
	}
	dump(U.Update(r2, len(r2), 1))
	dumpRecv(U)
	r3 := []byte{5, 0, 0, 0,
		5, 0, 5, 5,
	}
	dump(U.Update(r3, len(r3), 0))
	r4 := []byte{5, 0, 2, 2}
	dump(U.Update(r4, len(r4), 1))

	dumpRecv(U)

	U.Delete()
}
