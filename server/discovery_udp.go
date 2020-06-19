package server

import (
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"net"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

const discoveryServerType string = "udp4"

type UdpIpDiscoveryOptions struct {
	RemoteHost             string
	RemotePort             int
	LocalHost              string
	LocalPort              int
	ReadBufferLength       int
	PingTimeInterval       time.Duration
	ConnectTimeoutInterval time.Duration
}

type UdpIpDiscovery struct {
	options       UdpIpDiscoveryOptions
	remoteAddr    *net.UDPAddr
	conn          *net.UDPConn
	leaved        *int32
	addresses     map[string]*JoinAddress
	addressesLock *sync.RWMutex
	OnJoin        func(address string)
	OnLeave       func(address string)
	OnPing        func(address string)
}

type JoinAddress struct {
	address     string
	refreshTime time.Time
}

func (d *UdpIpDiscovery) Join() {
	//创建连接
	remoteAddr, remoteAddrErr := net.ResolveUDPAddr(discoveryServerType, d.options.RemoteHost+":"+strconv.Itoa(d.options.RemotePort))
	if remoteAddrErr != nil {
		panic(remoteAddrErr)
	}
	localAddr, localAddrErr := net.ResolveUDPAddr(discoveryServerType, d.options.LocalHost+":"+strconv.Itoa(d.options.LocalPort))
	if localAddrErr != nil {
		panic(localAddrErr)
	}

	conn, connErr := net.ListenUDP(discoveryServerType, localAddr)
	if connErr != nil {
		panic(connErr)
	}

	//设置结构体
	d.remoteAddr = remoteAddr
	d.conn = conn
	atomic.StoreInt32(d.leaved, 1)
	log.Println("udp ip discovery Join " + remoteAddr.String())
	//设置消息监听
	go d.readMessage()
	//通知加入集群
	go d.loopPing()
	//连接超时移除节点
	go d.timeoutPing()
}

func (d *UdpIpDiscovery) readMessage() {
	for atomic.LoadInt32(d.leaved) == 1 {
		data := make([]byte, d.options.ReadBufferLength)
		count, _, err := d.conn.ReadFromUDP(data)
		if err != nil {
			continue
		}
		var message IpPingMessage
		unmarshalErr := json.Unmarshal(data[:count], &message)
		if unmarshalErr != nil {
			log.Println(unmarshalErr)
		}
		if message.Status == IpPingMessageJoin {
			d.addressesLock.RLock()
			if _, ok := d.addresses[message.MessageAddress]; !ok {
				d.addressesLock.RUnlock()
				d.addressesLock.Lock()
				d.addresses[message.MessageAddress] = &JoinAddress{message.MessageAddress,
					time.Now()}
				d.addressesLock.Unlock()
				if d.OnJoin != nil {
					go d.OnJoin(message.MessageAddress)
				}
			} else {
				d.addressesLock.RUnlock()
				d.addressesLock.Lock()
				d.addresses[message.MessageAddress].refreshTime = time.Now()
				d.addressesLock.Unlock()
				if d.OnPing != nil {
					go d.OnPing(message.MessageAddress)
				}
			}
		}
		if message.Status == IpPingMessageLeave {
			d.addressesLock.RLock()
			if _, ok := d.addresses[message.MessageAddress]; ok {
				d.addressesLock.RUnlock()
				d.addressesLock.Lock()
				delete(d.addresses, message.MessageAddress)
				d.addressesLock.Unlock()
				if d.OnLeave != nil {
					go d.OnLeave(message.MessageAddress)
				}
			} else {
				d.addressesLock.RUnlock()
			}
		}
	}
}

func (d *UdpIpDiscovery) timeoutPing() {
	timeoutInterval := d.options.ConnectTimeoutInterval

	for atomic.LoadInt32(d.leaved) == 1 {
		time.Sleep(timeoutInterval)
		now := time.Now()
		d.addressesLock.RLock()
		for k, joinAddress := range d.addresses {
			if joinAddress.refreshTime.Add(timeoutInterval).Before(now) {
				delete(d.addresses, k)
				if d.OnLeave != nil {
					go d.OnLeave(k)
				}
			}
		}
		d.addressesLock.RUnlock()
	}
}

func (d *UdpIpDiscovery) Leave() {
	message := IpPingMessage{MessageAddress: getIpV4Address(), Status: IpPingMessageLeave}
	writeIpMessage(message, d)

	closeErr := d.conn.Close()
	if closeErr != nil {
		log.Error(closeErr)
	}
	atomic.StoreInt32(d.leaved, 0)

	log.Println("udp ip discovery leave " + d.options.RemoteHost + ":" + strconv.Itoa(d.options.RemotePort))
}

func (d *UdpIpDiscovery) Ping() {
	message := IpPingMessage{MessageAddress: getIpV4Address(), Status: IpPingMessageJoin}

	writeIpMessage(message, d)
}

func writeIpMessage(message IpPingMessage, d *UdpIpDiscovery) {
	data, dataErr := json.Marshal(message)
	if dataErr != nil {
		panic(dataErr)
	}

	_, e := d.conn.WriteToUDP(data, d.remoteAddr)
	if e != nil {
		log.Error(e)
	}
}

func (d *UdpIpDiscovery) loopPing() {
	for atomic.LoadInt32(d.leaved) == 1 {
		d.Ping()
		time.Sleep(d.options.PingTimeInterval)
	}
}

func (d *UdpIpDiscovery) GetAddresses() []string {
	addresses := make([]string, len(d.addresses))
	for k, _ := range d.addresses {
		addresses = append(addresses, k)
	}
	return addresses
}

func (d *UdpIpDiscovery) SetOnJoin(ls func(address string)) {
	d.OnJoin = ls
}

func (d *UdpIpDiscovery) SetOnLeave(ls func(address string)) {
	d.OnLeave = ls
}

func (d *UdpIpDiscovery) SetOnPing(ls func(address string)) {
	d.OnPing = ls
}

func (d *UdpIpDiscovery) AutoLeave() {
	c := make(chan os.Signal, 0)
	signal.Notify(c)

	<-c
	d.Leave()
}

func NewUdpIpDiscovery() IpDiscovery {
	options := LoadOptions(&UdpIpDiscoveryOptions{RemoteHost: "172.17.15.255", RemotePort: 8000, LocalHost: "0.0.0.0",
		LocalPort:        8000,
		ReadBufferLength: 1024, PingTimeInterval: time.Second * 10, ConnectTimeoutInterval: time.Second * 15},
		"discovery_udp")
	var leaved int32 = 1
	udp := &UdpIpDiscovery{options: *options.(*UdpIpDiscoveryOptions), leaved: &leaved,
		addresses: make(map[string]*JoinAddress), addressesLock: new(sync.RWMutex),
	}
	return udp
}
