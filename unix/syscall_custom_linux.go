package unix

import "unsafe"

func CreateMsg(fd int, p, oob []byte, to Sockaddr) (n *Msghdr, err error) {
	var ptr unsafe.Pointer
	var salen _Socklen
	if to != nil {
		var err error
		ptr, salen, err = to.sockaddr()
		if err != nil {
			return nil, err
		}
	}
	var msg Msghdr
	msg.Name = (*byte)(ptr)
	msg.Namelen = uint32(salen)
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
				return 0, err
			}
			// send at least one normal byte
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

	return &msg, nil
}

func SendMultiMsg(fd int, msgs []*Msghdr, flags int) (n int, err error) {
	if n, err = sendmmsg(fd, msgs, flags); err != nil {
		return 0, err
	}
	return n, nil
}

func sendmmsg(s int, msgs []*Msghdr, flags int) (n int, err error) {
	var _p0 unsafe.Pointer
	if len(msgs) > 0 {
		_p0 = unsafe.Pointer(&msgs[0])
	} else {
		_p0 = unsafe.Pointer(&_zero)
	}
	r0, _, e1 := Syscall6(SYS_SENDMSG, uintptr(s), uintptr(_p0), uintptr(len(msgs)), uintptr(flags), 0, 0)
	n = int(r0)
	if e1 != 0 {
		err = errnoErr(e1)
	}
	return
}