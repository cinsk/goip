package main

import (
	"encoding/binary"
	"fmt"
	"net"
)

type IP4Range struct {
	Begin uint32
	End   uint32
}

func NewIP4Range(cidr string) (IP4Range, error) {
	addr, net, err := net.ParseCIDR(cidr)
	if err != nil {
		return IP4Range{}, err
	}

	begin := binary.BigEndian.Uint32(addr[len(addr)-4:])
	mask := binary.BigEndian.Uint32(net.Mask)
	begin = begin & mask
	end := begin | ^mask

	return IP4Range{Begin: begin, End: end}, nil
}

func (r IP4Range) String() string {
	return fmt.Sprintf("%s-%s", int2ip(r.Begin), int2ip(r.End))
}

func ip2int(ip net.IP) uint32 {
	if len(ip) == 16 {
		return binary.BigEndian.Uint32(ip[12:16])
	}
	return binary.BigEndian.Uint32(ip)
}

func int2ip(nn uint32) net.IP {
	ip := make(net.IP, 4)
	binary.BigEndian.PutUint32(ip, nn)
	return ip
}
