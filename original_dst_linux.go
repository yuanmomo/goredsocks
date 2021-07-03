// +build linux

package main

import (
	"net"
	"os"
	"syscall"
	"unsafe"
)

const (
	// SO_ORIGINAL_DST is a Linux getsockopt optname.
	SO_ORIGINAL_DST = 80

	// IP6T_SO_ORIGINAL_DST a Linux getsockopt optname.
	IP6T_SO_ORIGINAL_DST = 80
)

// GetOriginalDST retrieves the original destination address from
// NATed connection.  Currently, only Linux iptables using DNAT/REDIRECT
// is supported.  For other operating systems, this will just return
// conn.LocalAddr().
//
// Note that this function only works when nf_conntrack_ipv4 and/or
// nf_conntrack_ipv6 is loaded in the kernel.
func GetOriginalDST(conn *net.TCPConn) (*net.TCPAddr, error) {
	f, err := conn.File()
	if err != nil {
		return nil, err
	}
	defer f.Close()

	fd := f.Fd()
	fdint := int(fd)
	// revert to non-blocking mode.
	// see http://stackoverflow.com/a/28968431/1493661
	if err = syscall.SetNonblock(fdint, true); err != nil {
		return nil, os.NewSyscallError("setnonblock", err)
	}

	v6 := conn.LocalAddr().(*net.TCPAddr).IP.To4() == nil
	if v6 {
		return ipv6_getorigdst(fd)
	} else {
		return getorigdst(fd)
	}
}

func getorigdst(fd uintptr) (*net.TCPAddr, error) {
	// IPv4
	raw := syscall.RawSockaddrInet4{}
	siz := unsafe.Sizeof(raw)
	if err := socketcall(GETSOCKOPT, fd, syscall.IPPROTO_IP, SO_ORIGINAL_DST, uintptr(unsafe.Pointer(&raw)), uintptr(unsafe.Pointer(&siz)), 0); err != nil {
		return nil, os.NewSyscallError("getsockopt", err)
	}
	ip := make([]byte, 4)
	for i, b := range raw.Addr {
		ip[i] = b
	}
	pb := *(*[2]byte)(unsafe.Pointer(&raw.Port))
	return &net.TCPAddr{
		IP:   ip,
		Port: int(pb[0])*256 + int(pb[1]),
	}, nil
}

func ipv6_getorigdst(fd uintptr) (*net.TCPAddr, error) {
	raw := syscall.RawSockaddrInet6{}
	siz := unsafe.Sizeof(raw)
	if err := socketcall(GETSOCKOPT, fd, syscall.IPPROTO_IPV6, IP6T_SO_ORIGINAL_DST, uintptr(unsafe.Pointer(&raw)), uintptr(unsafe.Pointer(&siz)), 0); err != nil {
		return nil, os.NewSyscallError("socketcall", err)
	}
	ip := make([]byte, 16)
	for i, b := range raw.Addr {
		ip[i] = b
	}
	pb := *(*[2]byte)(unsafe.Pointer(&raw.Port))
	return &net.TCPAddr{
		IP:   ip,
		Port: int(pb[0])*256 + int(pb[1]),
	}, nil
}
