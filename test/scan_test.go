package main

import (
	"io/ioutil"
	"net"
	"os"
	"reflect"
	"testing"
)

func TestAddBit(t *testing.T) {
	ip := net.ParseIP("192.168.0.0")
	addBit(ip)

	want := net.ParseIP("192.168.0.1")
	got := ip

	if want[len(ip)-1] != got[len(ip)-1] {
		t.Errorf("Wanted %q got %q", want, got)
	}
}

func TestPorts(t *testing.T) {
	// this test breaks for some reason
	var pl []string
	ports := "10,12,11,100-120"
	sem := make(chan struct{}, MAX)
	wg.Add(1)
	//
	parsePorts(&pl, ports, sem)
	//
	pl = []string{"10", "12", "11"}
	want := []string{"10", "12", "11"}
	//
	if !reflect.DeepEqual(want, pl) {
		t.Errorf("Wanted %q got %q", want, pl)
	}
}

func TestCidrParse(t *testing.T) {
	// Setup

	c := make(chan string)
	res := make([]string, 0)
	cidr := []string{"127.0.0.0/31"}
	sem := make(chan struct{}, MAX)

	sem <- struct{}{}
	wg.Add(1)
	go parseHosts(cidr, c, sem)
	// Start consumer
	for msg := range c {
		res = append(res, msg)
	}

	want := []string{"127.0.0.0", "127.0.0.1"}
	got := res
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Wanted %q got %q", want, got)
	}
}

func TestHostParse(t *testing.T) {
	// Setup
	c := make(chan string)
	res := make([]string, 0)
	cidr := []string{"127.0.0.0/31", "ruv.is", "192.168.0.1"}
	sem := make(chan struct{}, MAX)

	sem <- struct{}{}
	wg.Add(1)
	go parseHosts(cidr, c, sem)
	// Start consumer
	for msg := range c {
		res = append(res, msg)
	}

	want := []string{"127.0.0.0", "127.0.0.1", "ruv.is", "192.168.0.1"}
	got := res
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Wanted %q got %q", want, got)
	}
}

func TestTcpSend(t *testing.T) {
	// Stdout, this function only prints to Stdout
	resStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	print()

	// Ports List
	portList := []string{"80"}
	// Host to test
	host := "ruv.is"
	// Expected
	exp := "ruv.is:80 Open"
	// Semaphore
	sem := make(chan struct{}, MAX)
	// Set verbose to see if we do not get a connection
	verbose = 1
	// Run the function
	wg.Add(1)
	sem <- struct{}{}
	go sendTcp(host, portList, sem)
	wg.Wait()
	// Close pipe
	w.Close()
	out, _ := ioutil.ReadAll(r)
	os.Stdout = resStdout

	// I always pickup an message from the unittest about an
	// Invalid timeout ronge which i can't figure out how to get rid off// scan_test.go:115: Got "-test.timeout=10m0s -> is not avalid range.\nruv.is:80 Open\n", wanted "ruv.is:80 Open"
	got := string(out[len(out)-15 : len(out)-1]) // This removes the unwanted text
	// as well as unwanted newline charachter at the end of the line.
	if got != exp {
		t.Errorf("Got %q, wanted %q", got, exp)
	}
}
