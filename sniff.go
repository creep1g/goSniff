// Package main provides main  
package main

// scanner.go is a simple port scanner which is
import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// sync waitGroup 
var wg sync.WaitGroup

// MAX = 512  
// The max number of concurrent GoRoutines
// After some testing the upper limit of concurrent GoRoutines was set to 512
// Anything lower seems to be too time consuming and anything higher does not seems to be needed
const MAX = 512

// var verbose = 0  
// The verbose flag is initialized to 0
// If user inputs -v we print out unsuccessful connections
var verbose = 0

// printUsage function  
// A small manual for our little program.
func printUsage() {
	fmt.Println("Usage  : goSniff -p  <PORTS> <IP_ADDRESS/CIDR>")
	fmt.Println("Example: goSniff -p 22,88-108,8080 10.10.10.10 website.com 10.10.0.0/24")
	fmt.Println("Verbose: goSniff -p 22,88-108,8080 10.10.10.10 website.com 10.10.0.0/24 -v")
}

// parsePorts function  
// Parses the comma seperated section of the user input
// Parameters: portList *[]string, ports string, sem chan struct{}
// portList is a reference of a list holding each port gathered from the input
// ports is the actual commandline argument of comma seperated ports and ranges
// sem is channel representing a semaphore lock.
func parsePorts(portList *[]string, ports string, sem chan struct{}) {
	defer wg.Done() // Decrease wait group
	// Split the input string on comma
	splitPorts := strings.Split(os.Args[2], ",")
	// Iterate over each index of the slice
	for _, port := range splitPorts {

		if strings.Contains(port, "-") {
			// If the given index is a range of ports
			// fx 80-89, we split it into start end end variables
			// and iterate over that range.
			sem <- struct{}{} // Add to semaphore
			wg.Add(1)         // Add to waitgroup
			// We do this with goroutines to concurrently process multiple inputs of port ranges.
			go func(port string, portList *[]string, sem <-chan struct{}) {
				defer wg.Done() // Decrease wait group
				portRange := strings.Split(port, "-")
				start, _ := strconv.Atoi(portRange[0])
				end, _ := strconv.Atoi(portRange[1])
				if (start < end) && (start > 0) && (end <= 65535) {
					for i := start; i <= end; i++ {
						*portList = append(*portList, strconv.Itoa(i))
					}
				} else {
					fmt.Println(port, "-> is not avalid range.")
				}
				<-sem // Remove from semaphore
			}(port, *&portList, sem)
			continue
		}

		if _, err := strconv.Atoi(port); err != nil {
			// Checks if there are alphabetic letters in our port.
			fmt.Println(port, "-> is not a valid port.")
		}

		*portList = append(*portList, port)
	}
}

// addBit function  
// A cidrParse helper function takes parameters ip-address of type net.IP
func addBit(ip net.IP) {
	for i := len(ip) - 1; i >= 0; i-- {
		ip[i]++
		if ip[i] > 0 {
			break
		}
	}
}

// cidrParse function  
// Prases CIDR notations using net.ParseCIDR function and a host channel
// Takes input parameters of cidrString string, c chan<- string, sem chan struct{}, cidrWg *sync.WaitGroup
// cidrString is the string value of the network in the input fx: 192.168.0.0/16
// c is a write only channel that takes in strings
// sem is a representation of a semaphore lock
// cidrWg is a reference of a special waitgroup made just for this function to avoid
// closing the host channel too early
func cidrParse(cidrString string, c chan<- string, sem chan struct{}, cidrWg *sync.WaitGroup) {
	defer cidrWg.Done() // Decrease special waitgroup for this function

	ip, net, err := net.ParseCIDR(cidrString)
	// Parse cidr returns 3 values the IP address input with cidr notation
	// The network it belongs too and an error

	if err == nil {
		for ip := ip.Mask(net.Mask); net.Contains(ip); addBit(ip) {
			c <- ip.String() // Write each IP address to the host channel
		}
	} else {
		fmt.Println(err)
	}
	<-sem // release semaphore
}

// parseHosts function  
// Takes a list of hosts picked up from the user input
// a host channel which we write our hosts into
// And a semaphore
func parseHosts(hostArgs []string, hosts chan string, sem chan struct{}) {
	defer wg.Done()           // Release waitgroup
	defer close(hosts)        // Close the host channel once we return our function
	var cidrWg sync.WaitGroup // A waitgroup used to avoid closing the host channel too early
	for _, host := range hostArgs {

		ip := net.ParseIP(host) // Check if our host is a valid ip address

		if ip != nil {
			// A valid IP written into the channel
			hosts <- ip.String()
			continue
		}

		if strings.Contains(host, "/") {
			// For CIDR notation we have to concurrently generate each host in the network
			sem <- struct{}{} // Add to the semaphore
			cidrWg.Add(1)     // Add to cidr waitgroup
			go cidrParse(host, hosts, sem, &cidrWg)
			cidrWg.Wait() // Wait for CIDR functions to finish running
			continue
		}
		// If our host is a domain name "website.com"
		hosts <- host
	}
	<-sem // Release lock
}

// sendTcp function  
// Takes host as a parameter,
func sendTcp(host string, ports []string, sem chan struct{}) {
	defer wg.Done()

	for _, port := range ports {
		wg.Add(1)
		sem <- struct{}{}

		go func(port string, host string, sem <-chan struct{}) {
			defer wg.Done() // Decrease WaitGroup once function returns

			// Timeout ensures that we stop trying to connect if
			// the connection is not successfull within the given time
			// in this case it is 1 second.
			timeout := time.Second

			conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), timeout)
			// Here I make sure to close the connection once the function has returned.

			if conn != nil {
				// If we get a link we show it in the standard output.
				defer conn.Close()
				fmt.Println(net.JoinHostPort(host, port), "Open")
			}
			if err != nil {
				if verbose > 0 {
					fmt.Println(net.JoinHostPort(host, port), "Closed")
				}
			}

			<-sem
		}(port, host, sem)
	}
	<-sem
}

// main function  
func main() {
	// Make sure our input is valid
	if len(os.Args) < 4 {
		printUsage()
		return
	}

	if os.Args[1] != "-p" {
		printUsage()
		return
	}
	if os.Args[len(os.Args)-1] == "-v" {
		verbose = 1
	}

	var portList []string           // A slice that holds every port gathered from std-in
	sem := make(chan struct{}, MAX) // This channel will act as a semaphore wich will block if MAX is reached

	wg.Add(1) // Adds a worker to the wait group
	parsePorts(&portList, os.Args[2], sem)
	wg.Wait() // Wait for ports to finish

	hosts := make(chan string)
	wg.Add(1)
	//
	sem <- struct{}{}
	// Start the consumer.
	go func(hosts chan string, sem chan struct{}, portList []string) {
		defer wg.Done()
		for host := range hosts {
			wg.Add(1)
			sem <- struct{}{}
			go sendTcp(host, portList, sem)
			// fmt.Println(host)
		}
		<-sem
	}(hosts, sem, portList)

	wg.Add(1)
	sem <- struct{}{}
	var hostArgs []string

	if verbose > 0 {
		hostArgs = os.Args[3 : len(os.Args)-1]
	} else {
		hostArgs = os.Args[3:len(os.Args)]
	}

	go parseHosts(hostArgs, hosts, sem)

	wg.Wait()
	close(sem)
}
