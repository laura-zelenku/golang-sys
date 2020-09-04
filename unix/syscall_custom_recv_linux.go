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

func Recvmmsg(fd int, rrs []*ReceiveResp, flags int) (n int, err error) {
	msgs := make([]Mmsghdr, len(rrs))
	var rr *ReceiveResp

	for i := 0; i < len(rrs); i++ {
		rr = rrs[i]
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

		msgs[i].Msghdr = msg
	}

	if n, err = recvmmsg(fd, msgs, flags); err != nil {
		return
	}

	for i := 0; i < len(rrs); i++ {
		rr = rrs[i]
		rr.Size = msgs[i].Msglen
		msg := msgs[i].Msghdr

		oobn := int(msg.Controllen)
		recvflags := int(msg.Flags)
		// source address is only specified if the socket is unconnected
		var from Sockaddr
		rsa := (*RawSockaddrAny)(unsafe.Pointer(msg.Name))
		if rsa.Addr.Family != AF_UNSPEC {
			from, err = anyToSockaddr(fd, rsa)
		}

		rr.Oobn = oobn
		rr.Recvflags = recvflags
		rr.Err = err
		rr.From = from
	}

	return
}

type Mmsghdr struct {
	Msghdr Msghdr /* Message header */
	Msglen int    /* Number of received bytes for header */
}

func recvmmsg(s int, hs []Mmsghdr, flags int) (int, error) {
	n, _, errno := syscall.Syscall6(SYS_RECVMMSG, uintptr(s), uintptr(unsafe.Pointer(&hs[0])), uintptr(len(hs)), uintptr(flags), 0, 0)
	return int(n), errnoErr(errno)
}
