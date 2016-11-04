package connection

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"golang.org/x/net/websocket"
)

const (
	host       = "127.0.0.1:1883"
	route      = "/chat"
	fromserver = "hello client"
	fromclient = "hello server"
)

var (
	doneChan = make(chan bool)
	f        = func(data []byte) {
		fmt.Println(string(data))
		doneChan <- true
	}
	ff = func(data []byte) {
		fmt.Println(string(data))
		doneChan <- true
	}
)

func TestNormal(t *testing.T) {
	go func() {
		time.Sleep(1e3)
		client := &NormalClient{
			Addr:    host,
			Network: "tcp",
		}
		cnn, err := client.Dial()
		if err != nil {
			t.Error(err)
		}
		ccon := NewConn(cnn)
		ccon.SetHandler(f)
		ccon.Send([]byte(fromclient), time.Time{})
		// ccon.Close()
	}()
	go func() {
		h := func(c net.Conn) {
			scnn := NewConn(c)
			scnn.SetHandler(ff)
			scnn.Send([]byte(fromserver), time.Time{})
			// scnn.Close()
		}
		server := &NormalServer{host}
		server.Run(h)
	}()
	<-doneChan
	<-doneChan
}

func TestWebsocket(t *testing.T) {
	go func() {
		time.Sleep(1e3)
		client := &WebsocketClient{
			UrlAddress: "ws://" + host + route,
			UrlOrigin:  "http://" + host,
		}
		cnn, err := client.Dial()
		if err != nil {
			t.Error(err)
			return
		}
		ccon := NewConn(cnn)
		ccon.SetHandler(f)
		ccon.Send([]byte(fromclient), time.Time{})
		// ccon.Close()
	}()
	go func() {
		h := func(c net.Conn) {
			scnn := NewConn(c)
			scnn.SetHandler(ff)
			scnn.Send([]byte(fromserver), time.Time{})
			// scnn.Close()
		}
		server := &WebsocketServer{Route: route}
		server.Run(h)
		err := http.ListenAndServe(":1883", nil)
		if err != nil {
			t.Error(err)
		}
	}()
	<-doneChan
	<-doneChan
}

var caCertPEM = `-----BEGIN CERTIFICATE-----
MIIC/zCCAeegAwIBAgIJANPk8nOgXBfsMA0GCSqGSIb3DQEBCwUAMBYxFDASBgNV
BAMMC3RvbnliYWkuY29tMB4XDTE2MTAyMTAxMTgwOVoXDTMwMDYzMDAxMTgwOVow
FjEUMBIGA1UEAwwLdG9ueWJhaS5jb20wggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAw
ggEKAoIBAQCnJ6t0oOQJcfdEo26SuQW5XAKlddZyHQXX6wnUK9LbDvEYcv0B07KC
ADrbZXXK/BEQjeKlY1sR7qa8vYmoHBp+rrZlfeGak0qRtxsn9pVyAYa3qpmJiD/0
e9TpR1wVyyoTSk+yeFRmW8mx6hSuFyuTSbJb5a6jksm7sBy6648hvo21IVobZKad
x5OkDLgyTwrRDQtskLx8JpD2Ssma3rNCHbNhxz3esDvKtwfOQ1z0vh/ZPqNLpm8Y
USh/8H3kbphtwkN1Xk933D4ciNg6fN/NIDlXqyZj7wHdi+ztT6nMz+vAMLrUum+c
JpnVzZCKYx2AkHo+5WSjElq/t6EeAPB/AgMBAAGjUDBOMB0GA1UdDgQWBBShmPjU
kQYvJhscBT0zqlPC9vfJpTAfBgNVHSMEGDAWgBShmPjUkQYvJhscBT0zqlPC9vfJ
pTAMBgNVHRMEBTADAQH/MA0GCSqGSIb3DQEBCwUAA4IBAQCUoOtTDOV88MJ6YZIt
cx24UlJUTuEibvSISEok7u/bOT4vYD4+NsQsrnmcGF6bIWxz+wQHf0050Nyi6uN9
1okv+KqWhUYavkdy3MMMqCnBXMUql2E7E9La4+2/8vkGMiCSfkWd21kcvcf9tySe
Ep/dSWHvEMU2QYGLwSSgMugfyBzhCyVa3FSBF6rf9AhYJqud+xhLX5ny8yvKWJfS
m4jNWQC7VkVxE0Y4fjkHcyDK5Dg7B8rBdZt51G0NykfAwW4R31uYMlYoC9k67Sn1
+H9Gymn0oTdc6R5qOVM/waVI4zghwzRiZPFAazhLBFIzOl+NFZHVHTWqNvGX9lB3
lQcd
-----END CERTIFICATE-----
`

var serverCertPEM = `-----BEGIN CERTIFICATE-----
MIICpjCCAY4CCQC3XTI2I9k0ozANBgkqhkiG9w0BAQsFADAWMRQwEgYDVQQDDAt0
b255YmFpLmNvbTAeFw0xNjEwMjEwMTIyMDJaFw0zMDA2MzAwMTIyMDJaMBQxEjAQ
BgNVBAMMCWxvY2FsaG9zdDCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEB
ANCzQ/QFpR/NnJx+HyXWmiz/BLVSca5Q3nFs0y+HJLjR/ram4GQXfZHNNa8DfV9K
Z0RmBkDufsTmwdR1eoQxWJhsStPRRCm/F8JxENkREkw59wJDh1A9+eH2ZOjNF2YO
qFW12USG0Fxe9O3G5YxkHVmyaeP2zKFbZIlZpTuXvry4+RXc3ReOs+LT8UwR7qnn
G05NhOZJpqOQz4pZTyYdP/YmB0+F8NkhYohiJBd5ttzOiBbhAyxfv8K/b3ZTLgMn
dPTuWAHaFlfaKHt6i5/IHu6vlt1QqC/O5h2eFAW4VXoEyNxT9bsleS5Z1B3gqj0Z
9mcH9prPMdiw6c/YMz9qlj8CAwEAATANBgkqhkiG9w0BAQsFAAOCAQEAXbXr+VtV
Co9PBCWluaOg6tWMLbRKiyxw8ZSFXPqpv7Cv42GKyFuN0sBaXY2JbnoZAYAEiCKH
Oj/Q9/hSAF8tChTAMpnuIKVWILR05Wx9jktYEYUsMO8DYfoFPm0V2eQO0VimxIwe
V+zpPk+g7gXzaPCk0tMzPEhQI1hRBnd3ctjw2W84i04zL/YSXwPqXM5m+psLS3mp
2jxI2nxHhas5xWagPRiMi2/zKGeklCF4xPpVul3IDazovp+QLhvitYEMptmXZywp
9lE2b/BQpjkmahvg1DAO8GarUqwveJCpjsigAmb+ssMFgsq5oE39OgmmJogijEgK
MYI74dctKWbKwA==
-----END CERTIFICATE-----
`

var serverKeyPEM = `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEA0LND9AWlH82cnH4fJdaaLP8EtVJxrlDecWzTL4ckuNH+tqbg
ZBd9kc01rwN9X0pnRGYGQO5+xObB1HV6hDFYmGxK09FEKb8XwnEQ2RESTDn3AkOH
UD354fZk6M0XZg6oVbXZRIbQXF707cbljGQdWbJp4/bMoVtkiVmlO5e+vLj5Fdzd
F46z4tPxTBHuqecbTk2E5kmmo5DPillPJh0/9iYHT4Xw2SFiiGIkF3m23M6IFuED
LF+/wr9vdlMuAyd09O5YAdoWV9ooe3qLn8ge7q+W3VCoL87mHZ4UBbhVegTI3FP1
uyV5LlnUHeCqPRn2Zwf2ms8x2LDpz9gzP2qWPwIDAQABAoIBAHV7NFl9hAJvICYH
6UjHkWqa8++ORWg3JcD73bX3gXh43cW8ErzFbG5S8lFeyYiGvbMHK6YJm1sD+0C6
EQmxLYpAC69JMfG+HIXcV/uZXI+EYcPod5d4C95jcVKLgQx2W9RD1dZ5fXlCCuZ+
/GEXSl5fzLQDDhYW+HVI/XaaaUiaYyPeg4d8v7DgRRt0e4k3gw+iK4O9LdLjo8s7
20jO5iqB8Q39vItnenslhuJLFXTb9FKqYjMfF1tg4KvcxvWRHBKh7pew0cFg6x+D
SYH/vN1Y3sVdCV/WLhepl3xWehxELXidhhjwlVyzw/qpjU+n9kqcCcMRvdwclXsv
s02OkqECgYEA87aVfauLSmr49Qt1PzPoIVGJnfZ54lkz4o0MdQmgAWsZpZpaB+oc
UL7623E0OimLlXywQcQwyjOvPJPLVfvI1jZCTKUJF/0OQpoo5L0txjGR0AKB1U+L
1a1Ne9uluLMQIknLdWM5yZO/FLiLNH1Bzat0w3kHQKcripLav9opL4sCgYEA2zjL
6kZYdtXsUczpPs/9ZHfNWGSURfgjM3GaahvAne7jcY6nU4oKcPI7xmN5pMgDg/Ip
D5ZOXNMJrPQlZieRIDtJI4cCuTmqCASUu5UuTDaDFoA9OIrw5Bk3bMuFEGR8nmDw
YDY7sUnE58PVsjHxfMo24N4Hxq9hcZ1hfxKHCp0CgYEA7YlUAvyj8pB4vj3AtS1e
XrseKnwFJ/xRecqIDbqn3ToJ8UKT5Yrktj6uOhwID3hWuAijsjYKZ8ogTiau1Mtn
YIHZ9OIWDu9yaq/ek7NfXgEKYXdQHBw/6q/TCZl32KGOZB0Q1QU1WadLYmLMKwi6
jj9CuxLHYtiMs6+Wzc9QvSkCgYEAxUFT6telXjibTfeNoOFR5gcOlIzcBiGCuxVr
ljKmnPWJXnK4CSlav8qXEqoLmMQItb8+RzI+DLQwLpn41PShV1lBNGCViMlkP5av
piJT6GvchHCbpbcPjo9KGfD/KILixzf+0vO5vorcTJcgQBlEa78gpjHi2VqR2cN9
KPQo4tkCgYBrIctdUF2DpyQRSy8/uu3KKSATtVwivRYeeCRIcb6jJsvhpnEhKwCO
mfco7yeqVhaj3tq8UryWlU3RAmcDRlzz7mLw8BdzITkyR4jbYX4HMUQpez3GEzPM
A0Ac35A/Ggc0y25QaT1j3wa4JK4HozNW8xVyQlXtXCatVTYBiJqaDg==
-----END RSA PRIVATE KEY-----
`
var clientCertPEM = `-----BEGIN CERTIFICATE-----
MIICpzCCAY8CCQC3XTI2I9k0pDANBgkqhkiG9w0BAQsFADAWMRQwEgYDVQQDDAt0
b255YmFpLmNvbTAeFw0xNjEwMjEwMTQ1MjBaFw0zMDA2MzAwMTQ1MjBaMBUxEzAR
BgNVBAMMCnRvbnliYWlfY24wggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIB
AQD0mzvCAi1Vm4+VZ5wTNo1Dz+PAXfvyi8woOeUelWjK66gs7Z3yTLb/wzcvI0Tf
4ktyUoEwvL5OBj3osN0vbcFb+1YmjKFAML1xqkXrxNzweVkf3H6a1vnajEJ+3D9W
eIO4Og+Szc4PugY5UjaQs0zItIU85Rb2yLJYxAFWmj0sf1v38ekjp9URsOBQ1cam
cKNjaQ8UE+/84N1kuDbDNXHqhE3aeGpD1JBj6jtzYZ1qav2uU1JJtPkBuTA8QfVW
UdVxpmqfxUCKARprpaP2jiI4XesYh+qZITJbxRTfti69cNDsiT1qWkTD2GVi9wjp
r9Qn7w7yYJhjXFA0de3R2E5FAgMBAAEwDQYJKoZIhvcNAQELBQADggEBAAF+sBb5
X21aYBIJwhIStS+GrgwEtiKEkwjqFp1BpnI6NLbgkeqk4VoucX0ygWKYnPgUJ7lL
GrdG0MqTJj6q4VMywu8TlJfHRt0RACErHe6DQYVgFawzwU/v4QyWx2uBjZ0xfuMd
V4pr03K3XZTRGNU8VzgaNnmUN8Zhxm+hUgZC9JPoidsdUnTI/F45b6uXa05bU246
3FzubhhmK233xWN5Z/oOuU68orlTCZSE1IorPf93nQoc/Qw+qH1z2Hg60v9msD0h
/FjFBzQoGQFZAYue8KkYfl+22cl2EVXlL77nQlJH4inWkE3Ur08PWIrIH0TKdDEV
s5rJZod4qmBsQLU=
-----END CERTIFICATE-----
`

var clientKeyPEM = `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEA9Js7wgItVZuPlWecEzaNQ8/jwF378ovMKDnlHpVoyuuoLO2d
8ky2/8M3LyNE3+JLclKBMLy+TgY96LDdL23BW/tWJoyhQDC9capF68Tc8HlZH9x+
mtb52oxCftw/VniDuDoPks3OD7oGOVI2kLNMyLSFPOUW9siyWMQBVpo9LH9b9/Hp
I6fVEbDgUNXGpnCjY2kPFBPv/ODdZLg2wzVx6oRN2nhqQ9SQY+o7c2Gdamr9rlNS
SbT5AbkwPEH1VlHVcaZqn8VAigEaa6Wj9o4iOF3rGIfqmSEyW8UU37YuvXDQ7Ik9
alpEw9hlYvcI6a/UJ+8O8mCYY1xQNHXt0dhORQIDAQABAoIBAEmPiZgdyX5uSu72
esR4amFTWr3WRde6XQpX5uScsCgeWLQyPYbJtVsTXTwq0iK2TlQyJbH/gQe77tBU
qCAEVNsRqeXr8u53+fD98QsnZ/5VsAzZ5aUxM3CbL8AxAkdB9PLpWxeeFvM50XdZ
lxlLIrNVPqCCkLZkwuXZBEIEHpOLnxzR/p/wa8rOrC7u+mwC6lX/P+8uEHjc33lF
350CFhDE0/cPUavPT98REejeSUUm9YH2bA0AgumC3MI0tAJNyDWe/qXnfc6k3Ugc
GyED7hyLp48G5UQu8dPWPbtR8RdJAYqdRtVSv1TpUqqaA9yPi9L+bpsLTT/bJVx2
PFbZQZ0CgYEA+xP3+5N4hfcAaipDi6grv/MShrrByXf7SgI4YlEDa47H3yAgApDH
XGm8I1asPebeDMrSrgKdJrYElnr5aYSKUHdYxKLnmLxZ3Zng9HQn5vwPUKgv2+Vr
nxwXnSa69XzjKqeu7CfKvR40uUAJdssHeSBWOOyUSW0I+3KjhnTbHs8CgYEA+WbJ
eSghIOZCe9719+EXaNLqtj8VSeBIZ5Itaoxilvd2YujNolskbnY2ckhSrlNLIVNd
VmPlNFlsvfRWTHjYsnxl/HuVqUhGCrv0asHPkgjpo3UUPPRq7WUptoRd/+PMNOtB
0iRXOiIXYkGMFRffQUqd64AQjH0vX+7IMm6YJqsCgYEAospjst3++v0Xa75TZS14
kFAN5wUXuITujSG7ZSOZ0BcXSHgPyRoN6ME1lsQPkWMq/ahTyTcwpXTGrLq3E883
zsxS5cup1cHpkmC/FkBpzr4HQAiMX0r06IjSVrZR6fE7aOCn7b4vGUeIb8QxXrBs
/AAXZ3kc/C6R8FZ36CnEGMkCgYEA7/e4jzzPc2paOfmzzUflUFTwxV45S5Xj1NPP
ox9klUGRxWWexkLP8QEJLjjsZRN7zJr7ye3gUdhhSvxYNMhqKIKNVrxKqlECI73p
VPcak7bDpyU1zdiXMcgOtLD0CQzUJW7fLCUPUL4QpdfPw1Pu2lHDVR8pzSN+e92y
/uNuvnUCgYBt787XPshPz/RAIByH5fayboMXE42Oh36Wwr+7oCDaANxdkcRgg456
HYC4FIyffaGHu4j1hI4PAoSkpOwdOZh9vSyFhnuRUDhPj82rmvDLz/dA0exbZgMM
4SL45ATOnw0QPvLjCXjLpCQFUQ4po+vp30WMY3VTMEPlJJSXCxIUsw==
-----END RSA PRIVATE KEY-----
`

var (
	pool          = x509.NewCertPool()
	serverCert, _ = tls.X509KeyPair([]byte(serverCertPEM), []byte(serverKeyPEM))
	clientCert, _ = tls.X509KeyPair([]byte(clientCertPEM), []byte(clientKeyPEM))
)

func TestTls(t *testing.T) {
	pool.AppendCertsFromPEM([]byte(caCertPEM))
	go func() {
		time.Sleep(1e3)
		cfg := &tls.Config{
			Certificates:       []tls.Certificate{clientCert},
			RootCAs:            pool,
			InsecureSkipVerify: true,
		}
		client := &TlsClient{
			Config:  cfg,
			Network: "tcp",
			Addr:    host,
		}
		cnn, err := client.Dial()
		if err != nil {
			t.Error(err)
			return
		}
		ccon := NewConn(cnn)
		ccon.SetHandler(f)
		ccon.Send([]byte(fromclient), time.Time{})
	}()

	go func() {
		h := func(c net.Conn) {
			scnn := NewConn(c)
			scnn.SetHandler(ff)
			scnn.Send([]byte(fromserver), time.Time{})
		}
		cfg := &tls.Config{
			Certificates: []tls.Certificate{serverCert},
			ClientAuth:   tls.RequireAndVerifyClientCert,
			ClientCAs:    pool,
		}
		server := &TlsServer{
			Config: cfg,
			Addr:   host,
		}
		server.Run(h)
	}()
	<-doneChan
	<-doneChan
}

//test fail: server has no response, connect fail
func TestWss(t *testing.T) {
	pool.AppendCertsFromPEM([]byte(caCertPEM))
	go func() {
		time.Sleep(1e3)
		cfg, err := websocket.NewConfig("wss://"+host+route, "https://"+host)
		if err != nil {
			fmt.Println(err)
			return
		}
		cfg.TlsConfig = &tls.Config{
			Certificates:       []tls.Certificate{clientCert},
			RootCAs:            pool,
			InsecureSkipVerify: true,
		}
		client := &WebsocketClient{
			Config: cfg,
		}
		cnn, err := client.Dial()
		if err != nil {
			fmt.Println(err)
			return
		}
		ccon := NewConn(cnn)
		ccon.SetHandler(f)
		ccon.Send([]byte(fromclient), time.Time{})
	}()

	go func() {
		h := func(c net.Conn) {
			scnn := NewConn(c)
			scnn.SetHandler(ff)
			scnn.Send([]byte(fromserver), time.Time{})
		}
		cfg := new(websocket.Config)
		cfg.TlsConfig = &tls.Config{
			Certificates: []tls.Certificate{serverCert},
			ClientAuth:   tls.RequireAndVerifyClientCert,
			ClientCAs:    pool,
		}
		server := &WebsocketServer{
			Config: cfg,
			Route:  route,
		}
		server.Run(h)

		s := &http.Server{
			Addr: ":1883",
			TLSConfig: &tls.Config{
				ClientCAs:  pool,
				ClientAuth: tls.RequireAndVerifyClientCert,
			},
		}
		// err := s.ListenAndServe()
		err := s.ListenAndServeTLS("testdata/server.crt", "testdata/server.key")
		if err != nil {
			fmt.Println(err)
		}
	}()
	<-doneChan
	<-doneChan
}
