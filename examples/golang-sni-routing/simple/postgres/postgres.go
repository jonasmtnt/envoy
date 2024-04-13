package postgres

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"

	"github.com/traefik/traefik/v2/pkg/tcp"
)

const defaultBufSize = 4096

var (
	PostgresStartTLSMsg   = []byte{0, 0, 0, 8, 4, 210, 22, 47} // int32(8) + int32(80877103)
	PostgresStartTLSReply = []byte{83}                         // S
	PostgresStartReply    = []byte{71}                         // G
)

type router struct{}

// isPostgres determines whether the buffer contains the Postgres STARTTLS message.
func isPostgres(br *bufio.Reader) (bool, error) {
	// Peek the first 8 bytes individually to prevent blocking on peek
	// if the underlying conn does not send enough bytes.
	// It could happen if a protocol start by sending less than 8 bytes,
	// and expect a response before proceeding.
	for i := 1; i < len(PostgresStartTLSMsg)+1; i++ {
		peeked, err := br.Peek(i)
		if err != nil {
			fmt.Printf("Error while Peeking first bytes: %s\n", err)
			return false, err
		}

		if !bytes.Equal(peeked, PostgresStartTLSMsg[:i]) {
			return false, nil
		}
	}
	return true, nil
}

// servePostgres serves a connection with a Postgres client negotiating a STARTTLS session.
// It handles TCP TLS routing, after accepting to start the STARTTLS session.
func servePostgres(conn tcp.WriteCloser) {
	_, err := conn.Write(PostgresStartTLSReply)
	if err != nil {
		conn.Close()
		return
	}

	br := bufio.NewReader(conn)

	b := make([]byte, len(PostgresStartTLSMsg))
	_, err = br.Read(b)
	if err != nil {
		conn.Close()
		return
	}

	hello, err := clientHelloInfo(br)
	if err != nil {
		conn.Close()
		return
	}

	if !hello.isTLS {
		conn.Close()
		return
	}

	return
}

// clientHelloInfo returns various data from the clientHello handshake,
// without consuming any bytes from br.
// It returns an error if it can't peek the first byte from the connection.
func clientHelloInfo(br *bufio.Reader) (*clientHello, error) {
	hdr, err := br.Peek(1)
	if err != nil {
		return nil, err
	}

	// No valid TLS record has a type of 0x80, however SSLv2 handshakes start with an uint16 length
	// where the MSB is set and the first record is always < 256 bytes long.
	// Therefore, typ == 0x80 strongly suggests an SSLv2 client.
	const recordTypeSSLv2 = 0x80
	const recordTypeHandshake = 0x16
	fmt.Printf("hdr[0]: %v\n", hdr[0])
	if hdr[0] != recordTypeHandshake {
		if hdr[0] == recordTypeSSLv2 {
			// we consider SSLv2 as TLS, and it will be refused by real TLS handshake.
			return &clientHello{
				isTLS:  true,
				peeked: getPeeked(br),
			}, nil
		}
		return &clientHello{
			peeked: getPeeked(br),
		}, nil // Not TLS.
	}

	const recordHeaderLen = 5
	hdr, err = br.Peek(recordHeaderLen)
	if err != nil {
		return nil, err
	}

	recLen := int(hdr[3])<<8 | int(hdr[4]) // ignoring version in hdr[1:3]

	if recordHeaderLen+recLen > defaultBufSize {
		br = bufio.NewReaderSize(br, recordHeaderLen+recLen)
	}

	helloBytes, err := br.Peek(recordHeaderLen + recLen)
	if err != nil {
		return nil, err
	}

	sni := ""
	var protos []string

	server := tls.Server(helloSniffConn{r: bytes.NewReader(helloBytes)}, &tls.Config{
		GetConfigForClient: func(hello *tls.ClientHelloInfo) (*tls.Config, error) {
			sni = hello.ServerName
			protos = hello.SupportedProtos
			return nil, nil
		},
	})
	_ = server.Handshake()

	return &clientHello{
		serverName: sni,
		isTLS:      true,
		peeked:     getPeeked(br),
		protos:     protos,
	}, nil
}

func getPeeked(br *bufio.Reader) string {
	peeked, err := br.Peek(br.Buffered())
	if err != nil {
		log.Fatal(err)
		return ""
	}
	return string(peeked)
}

// helloSniffConn is a net.Conn that reads from r, fails on Writes,
// and crashes otherwise.
type helloSniffConn struct {
	r        io.Reader
	net.Conn // nil; crash on any unexpected use
}

type clientHello struct {
	serverName string   // SNI server name
	protos     []string // ALPN protocols list
	isTLS      bool     // whether we are a TLS handshake
	peeked     string   // the bytes peeked from the hello while getting the info
}

// GetConn creates a connection proxy with a peeked string.
func GetConn(conn tcp.WriteCloser, peeked string) tcp.WriteCloser {
	// FIXME should it really be on Router ?
	conn = &Conn{
		Peeked:      []byte(peeked),
		WriteCloser: conn,
	}

	return conn
}

// Conn is a connection proxy that handles Peeked bytes.
type Conn struct {
	// Peeked are the bytes that have been read from Conn for the
	// purposes of route matching, but have not yet been consumed
	// by Read calls. It set to nil by Read when fully consumed.
	Peeked []byte

	// Conn is the underlying connection.
	// It can be type asserted against *net.TCPConn or other types
	// as needed. It should not be read from directly unless
	// Peeked is nil.
	tcp.WriteCloser
}
