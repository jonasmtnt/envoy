package main

import (
	"fmt"
	"net"
	"slices"

	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"
	"github.com/envoyproxy/envoy/contrib/golang/filters/network/source/go/pkg/network"
	"github.com/envoyproxy/envoy/examples/golang-sni-routing/simple/postgres"
)

func init() {
	network.RegisterNetworkFilterConfigFactory("simple", cf)
}

var cf = &configFactory{}

type configFactory struct{}

func (f *configFactory) CreateFactoryFromConfig(config interface{}) network.FilterFactory {
	/*
		a := config.(*anypb.Any)
		configStruct := &xds.TypedStruct{}
		_ = a.UnmarshalTo(configStruct)

		v := configStruct.Value.AsMap()["echo_server_addr"]
		addr, err := net.LookupHost(v.(string))
		if err != nil {
			fmt.Printf("fail to resolve: %v, err: %v\n", v.(string), err)
			return nil
		}
		upAddr := addr[0] + ":1025"

		return &filterFactory{
			upAddr: upAddr,
		}
	*/
	return &filterFactory{}
}

type filterFactory struct {
	//upAddr string
}

func (f *filterFactory) CreateFilter(cb api.ConnectionCallback) api.DownstreamFilter {
	return &downFilter{
		//upAddr: f.upAddr,
		cb: cb,
	}
}

type downFilter struct {
	api.EmptyDownstreamFilter

	cb       api.ConnectionCallback
	upAddr   string
	upFilter *upFilter
}

func (f *downFilter) OnNewConnection() api.FilterStatus {
	localAddr, _ := f.cb.StreamInfo().UpstreamLocalAddress()
	remoteAddr, _ := f.cb.StreamInfo().UpstreamRemoteAddress()
	fmt.Println("----------------------------------------------------")
	fmt.Printf("OnNewConnection, local: %v, remote: %v\n", localAddr, remoteAddr)
	f.upFilter = &upFilter{
		downFilter: f,
		ch:         make(chan []byte, 1),
	}
	//network.CreateUpstreamConn(f.upAddr, f.upFilter)
	return api.NetworkFilterContinue
}

func (f *downFilter) OnData(buffer []byte, endOfStream bool) api.FilterStatus {
	remoteAddr, _ := f.cb.StreamInfo().UpstreamRemoteAddress()
	fmt.Printf("(downFilter) OnData, addr: %v, buffer: %v, endOfStream: %v\n", remoteAddr, string(buffer), endOfStream)

	if slices.Compare(buffer, []byte{0, 0, 0, 8, 4, 210, 22, 48}) == 0 {
		fmt.Println("first tcp packet")
	}

	if slices.Compare(buffer, postgres.PostgresStartTLSMsg) == 0 {
		fmt.Println("second tcp packet received --> PostgresStartTLSMsg")
	}

	/*
		host := ""
		if len(buffer) > 200 {
			for i := 153; buffer[i] != 0; i++ {
				host = fmt.Sprintf("%s%s", host, string(buffer[i:i+1]))
			}
			fmt.Printf("Host: %s\n", host)
		}

		port := 5432
		if host == "" {
			host = "echo_service"
			port = 1025
		}

		ips, err := net.LookupIP(host)
		if err != nil {
			fmt.Printf("%v\n", err)
		}
		f.upAddr = fmt.Sprintf("%s:%d", ips[0].String(), port)
		fmt.Println(f.upAddr)
		network.CreateUpstreamConn(f.upAddr, f.upFilter)

		//TODO: send manually the two start messages
		//[]byte{0, 0, 0, 8, 4, 210, 22, 48}
		//postgres.PostgresStartTLSMsg
	*/
	ips, err := net.LookupIP("postgres1")
	if err != nil {
		fmt.Printf("%v\n", err)
	}
	f.upAddr = ips[0].String() + ":5432"
	network.CreateUpstreamConn(f.upAddr, f.upFilter)
	f.upFilter.ch <- buffer
	return api.NetworkFilterContinue
}

func (f *downFilter) OnEvent(event api.ConnectionEvent) {
	remoteAddr, _ := f.cb.StreamInfo().UpstreamRemoteAddress()
	fmt.Printf("OnEvent, addr: %v, event: %v\n", remoteAddr, event)
}

func (f *downFilter) OnWrite(buffer []byte, endOfStream bool) api.FilterStatus {
	fmt.Printf("OnWrite, buffer: %v, endOfStream: %v\n", string(buffer), endOfStream)
	return api.NetworkFilterContinue
}

type upFilter struct {
	api.EmptyUpstreamFilter

	cb         api.ConnectionCallback
	downFilter *downFilter
	ch         chan []byte
}

func (f *upFilter) OnPoolReady(cb api.ConnectionCallback) {
	f.cb = cb
	localAddr, _ := f.cb.StreamInfo().UpstreamLocalAddress()
	remoteAddr, _ := f.cb.StreamInfo().UpstreamRemoteAddress()
	fmt.Printf("OnPoolReady, local: %v, remote: %v\n", localAddr, remoteAddr)
	go func() {
		for {
			buf, ok := <-f.ch
			if !ok {
				return
			}
			f.cb.Write(buf, false)
		}
	}()
}

func (f *upFilter) OnPoolFailure(poolFailureReason api.PoolFailureReason, transportFailureReason string) {
	fmt.Printf("OnPoolFailure, reason: %v, transportFailureReason: %v\n", poolFailureReason, transportFailureReason)
}

func (f *upFilter) OnData(buffer []byte, endOfStream bool) {
	remoteAddr, _ := f.cb.StreamInfo().UpstreamRemoteAddress()
	fmt.Printf("(upFilter) OnData, addr: %v, buffer: %v, endOfStream: %v\n", remoteAddr, string(buffer), endOfStream)

	/*
		if slices.Compare(buffer, []byte{0, 0, 0, 8, 4, 210, 22, 48}) == 0 {
			f.downFilter.cb.Write(postgres.PostgresStartReply, endOfStream)
			return
		}

		if slices.Compare(buffer, postgres.PostgresStartTLSMsg) == 0 {
			f.downFilter.cb.Write(postgres.PostgresStartTLSReply, endOfStream)
			return
		}

		// l o c a l h o s t
		// 108 111 99 97 108 104 111 115 116 	--> end with 0 (NUL)

		// p o s t g r e s 1
		// 112 111 115 116 103 114 101 115 49 --> end with 0 (NUL)

		// byte 155
		if len(buffer) > 200 {
			s := ""
			for i := 153; buffer[i] != 0; i++ {
				s = fmt.Sprintf("%s%s", s, string(buffer[i:i+1]))
			}
			fmt.Printf("Host: %s\n", s)
		}
	*/
	f.downFilter.cb.Write(buffer, endOfStream)
}

func (f *upFilter) OnEvent(event api.ConnectionEvent) {
	remoteAddr, _ := f.cb.StreamInfo().UpstreamRemoteAddress()
	fmt.Printf("OnEvent, addr: %v, event: %v\n", remoteAddr, event)
	if event == api.LocalClose || event == api.RemoteClose {
		// will set ok to false for a closed and empty channel
		_, ok := <-f.ch
		if ok {
			close(f.ch)
		}
	}
}

func main() {}
