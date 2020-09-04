package unix

import (
	"syscall"
	"unsafe"
)

const (
	MaxSegmentSize = (1 << 16) - 1 // largest possible UDP datagram
)

type ReceiveResp struct {
	Oob       []byte
	P         []byte
	Size      int
	From      Sockaddr
	Err       error
	Recvflags int
	Oobn      int
}

func Recvmsgs2(fd int, rr *ReceiveResp, flags int) (n, oobn int, recvflags int, from Sockaddr, err error) {
	return Recvmsgs(fd, rr.P, rr.Oob, flags)
}

func Recvmsgs3(fd int, rr *ReceiveResp, flags int) (n int, err error) {
	msgs := make([]Mmsghdr, 1)
	var msg Msghdr
	var rsa RawSockaddrAny
	msg.Name = (*byte)(unsafe.Pointer(&rsa))
	msg.Namelen = uint32(SizeofSockaddrAny)
	var iov Iovec
	if len(rr.P) > 0 {
		iov.Base = &rr.P[0]
		iov.SetLen(len(rr.P))
	}
	var dummy byte
	if len(rr.Oob) > 0 {
		if len(rr.P) == 0 {
			var sockType int
			sockType, err = GetsockoptInt(fd, SOL_SOCKET, SO_TYPE)
			if err != nil {
				return
			}
			// receive at least one normal byte
			if sockType != SOCK_DGRAM {
				iov.Base = &dummy
				iov.SetLen(1)
			}
		}
		msg.Control = &rr.Oob[0]
		msg.SetControllen(len(rr.Oob))
	}
	msg.Iov = &iov
	msg.Iovlen = 1
	msgs[0].Msghdr = msg

	if n, err = recvmmsg(fd, msgs, flags); err != nil {
		return
	}
	rr.Size = msgs[0].Msglen

	oobn := int(msg.Controllen)
	recvflags := int(msg.Flags)
	// source address is only specified if the socket is unconnected
	var from Sockaddr
	if rsa.Addr.Family != AF_UNSPEC {
		from, err = anyToSockaddr(fd, &rsa)
	}

	rr.Oobn = oobn
	rr.Recvflags = recvflags
	rr.Err = err
	rr.From = from
	return
}

func Recvmsgs(fd int, p, oob []byte, flags int) (n, oobn int, recvflags int, from Sockaddr, err error) {
	msgs := make([]Mmsghdr, 1)
	var msg Msghdr
	var rsa RawSockaddrAny
	msg.Name = (*byte)(unsafe.Pointer(&rsa))
	msg.Namelen = uint32(SizeofSockaddrAny)
	var iov Iovec
	if len(p) > 0 {
		iov.Base = &p[0]
		iov.SetLen(len(p))
	}
	var dummy byte
	if len(oob) > 0 {
		if len(p) == 0 {
			var sockType int
			sockType, err = GetsockoptInt(fd, SOL_SOCKET, SO_TYPE)
			if err != nil {
				return
			}
			// receive at least one normal byte
			if sockType != SOCK_DGRAM {
				iov.Base = &dummy
				iov.SetLen(1)
			}
		}
		msg.Control = &oob[0]
		msg.SetControllen(len(oob))
	}
	msg.Iov = &iov
	msg.Iovlen = 1
	msgs[0].Msghdr = msg
	if n, err = recvmmsg(fd, msgs, flags); err != nil {
		return
	}
	n = msgs[0].Msglen
	oobn = int(msg.Controllen)
	recvflags = int(msg.Flags)
	// source address is only specified if the socket is unconnected
	if rsa.Addr.Family != AF_UNSPEC {
		from, err = anyToSockaddr(fd, &rsa)
	}
	return
}

// from net package

//func recvMsgs(ms []Message, flags int) (int, error) {
//	for i := range ms {
//		ms[i].raceWrite()
//	}
//	hs := make(mmsghdrs, len(ms))
//	var parseFn func([]byte, string) (net.Addr, error)
//	if c.network != "tcp" {
//		parseFn = parseInetAddr
//	}
//	if err := hs.pack(ms, parseFn, nil); err != nil {
//		return 0, err
//	}
//	var operr error
//	var n int
//	fn := func(s uintptr) bool {
//		n, operr = recvmmsg(s, hs, flags)
//		if operr == syscall.EAGAIN {
//			return false
//		}
//		return true
//	}
//	if err := c.c.Read(fn); err != nil {
//		return n, err
//	}
//	if operr != nil {
//		return n, os.NewSyscallError("recvmmsg", operr)
//	}
//	if err := hs[:n].unpack(ms[:n], parseFn, c.network); err != nil {
//		return n, err
//	}
//	return n, nil
//}

type Mmsghdr struct {
	Msghdr Msghdr /* Message header */
	Msglen int    /* Number of received bytes for header */
}

func recvmmsg(s int, hs []Mmsghdr, flags int) (int, error) {
	n, _, errno := syscall.Syscall6(SYS_RECVMMSG, uintptr(s), uintptr(unsafe.Pointer(&hs[0])), uintptr(len(hs)), uintptr(flags), 0, 0)
	return int(n), errnoErr(errno)
}
