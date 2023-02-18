# goSniff
### A Concurrent port scanner.
--------

## About the project
<p>
	The goal of this project is te create a simple port sniff, we require it to be fast and therefore we use concurrency to get results as fast as possible.
</p>
## Future goals

    - Make input parsing better
    - Gather more information about ports and their Usage

## Components
<p>
The program is broken into 4 parts: 

	- Argument Parsing
	- Gathering ports
	- Gathering Hosts
	- Checking the connection

Each of these parts run various go routines under different circumstances. 
</p>
## Installation

```bash
git clone git@github.com:creep1g/goSniff.git
cd goSniff
make build    // builds in current directory
make install // adds goSniff to path

```
## Usage 
```bash
go run sniff.go -p <PORTS> <HOST/NETWORK> 
``` 
For example : 
```bash
go run sniff.go -p 22,80,8070-8080 website.com 89.11.12.21 127.0.0.0/24
```
Normally it only outputs successful connections but if we wan't to see unsuccesful connections we can add a verbose flag 
```bash
go run sniff.go -p 22,80,8070-8080 website.com 89.11.12.21 127.0.0.0/24 -v
```

## Documentations
### Note: Some of the documentation is in the form of comments in the file sniff.go. The rest will be here.
<p>

### Packages used : 
- fmt
  - Used to relate messages to STDOUT see line 178 in sendTcp function of sniff.go
- net
  - The net package is used to parse IP addresses and CIDR notation strings see line 133 of parseHosts and line 109 of cidrParse in sniff.go
- os
  - The os package is used for argument parsing see line 200 in moin() of sniff.go
- strconv
  - strconv is used to convert strings to integers and vice versa see lines 64 - 69 of parsePorts in sniff.go.
- strings
	- Used mostly in the argument parsing to split input strings. see line 63 of parsePorts in sniff.go
- sync
  - Synt provides us with the waitgroup type, which is in essence just a counter, before every goroutine we add to it and after it returns we remove from it. Examples on lines 216 and 221 in the main function of sniff.go
- time
  - Used to create a timelimit for our tcp request. See line 170 sendTcp in sniff.go

## Functions
---
#### Main()
##### lines 193 - 245
 The main function is the one that is called as soon as we call the program. from lines 195 to 206 are only used to check if the user input is valid or not. 
 <br/>
 There after some variables are initialized most notably our portList(line 208) and sem(line 209). More information on variables can be seen in the[variables](#Variables) part of this section.
In main there is a single anonymous function which can be found in lines 220 - 229. this anonymous function is run as a goroutine which runs our consumer. 
```go
	hosts := make(chan string) // A channel of strings we fill with our hostnames 
	wg.Add(1)				   // Add to our sync.WaitGroup counter
	
	sem <- struct{}{}          // Add to our semaphore
	// Start the consumer.
	go func(hosts chan string, sem chan struct{}, portList []string) {
		defer wg.Done()		  // Decrease Waitgroup
		for host := range hosts {	// Iterate over our host and wait for a hostname to be sent to it
			wg.Add(1)				// Add to our waitgroup
			sem <- struct{}{}		// Add to our semaphore
			go sendTcp(host, portList, sem)	// When a host appears we send that hostname, 
											// with our list of ports to the sendTcp function and check if we get an open connection
		}
		<-sem // Release semaphore
	}(hosts, sem, portList)
```
This goroutine is run before we start gathering our hostnames with the [parseHosts](#parseHosts) function.

```go
	wg.Add(1)              // As before we add to our waitgroupt
	sem <- struct{}{}	   // And our semaphore
	var hostArgs []string  // a list of strings 

	// If our verbose flag is set we need to remove it from our list of hosts
	if verbose > 0 {
		hostArgs = os.Args[3 : len(os.Args)-1] // Add the hostnames gathered from user input
	} else {
		hostArgs = os.Args[3:len(os.Args)]
	}

	go parseHosts(hostArgs, hosts, sem) // Parse hosts

	wg.Wait()  // Wait for every go routine to finish running
	close(sem) // Close our semaphore channel.

```
In lines 253 - 244 we can see how the verbose flag is parsed, this could be done more simply with additional packages but i decided to run with this for personal simplicity.

### parsePorts
##### lines 47 - 85
Parses the comma seperated section of the user input, it takes in 3 parameters 
- portList *[]string
	- A reference to a list of strings we fill with port numbers 
- ports string
  - The comma seperated input from the command line
- sem chan struct{}
  - Our semaphore lock.

This function contains a single anonymous function which is ran whenever we get a "port" of the range format "123-125".
```go
func parsePorts(portList *[]string, ports string, sem chan struct{}) {
	defer wg.Done() // Decrease wait group
	splitPorts := strings.Split(os.Args[2], ",") // Split the input on commas 
	// Iterate over each index of the slice
	for _, port := range splitPorts {  
		if strings.Contains(port, "-") {
			sem <- struct{}{} // Add to semaphore
			wg.Add(1)         // Add to waitgroup
			// We do this with goroutines to concurrently process multiple inputs of port ranges.
			go func(port string, portList *[]string, sem <-chan struct{}) {  // Here we use a read-only semaphore. 
				defer wg.Done() // Decrease wait group
				portRange := strings.Split(port, "-")	// Plit range by "-"
				start, _ := strconv.Atoi(portRange[0])	// Convert strings to int
				end, _ := strconv.Atoi(portRange[1])
				if (start < end) && (start > 0) && (end <= 65535) {  // If our range is valid continue
					for i := start; i <= end; i++ {
						*portList = append(*portList, strconv.Itoa(i))  // Converts the integer value to a string and appends to our port list
					}
				} else {
					fmt.Println(port, "-> is not avalid range.")
				}
				<-sem // Remove from semaphore
			}(port, *&portList, sem)
			continue // go to next iteration of our for loop
		}
		if _, err := strconv.Atoi(port); err != nil {
			// Checks if there are alphabetic letters in our port.
			fmt.Println(port, "-> is not a valid port.")
		}
		*portList = append(*portList, port) // Append port to portList
	}
}
```
### parseHosts
##### lines 123 - 149
Parses the host section of the commandline arguments. 
Takes 3 parameters :
- hostArgs []string
  - A slice containing each host/network passed to our program from the command line
- hosts chan string
  - This channel is the main channel used in our program, parseHosts writes each hostname gathered to it and as seen in [main](#main)'s anonymous function  where we read from it.
- sem chan struct{}
  - Our semaphore lock

In developement there was a problem where our host channel would close too early to avoid that I create a single use waitgroup for when I parse CIDR
strings, with this we can wait for the cidr parsers to finish and then continue.
```go
func parseHosts(hostArgs []string, hosts chan string, sem chan struct{}) {
	defer wg.Done()           // Release waitgroup
	defer close(hosts)        // Close the host channel once we return our function
	var cidrWg sync.WaitGroup // A waitgroup used to avoid closing the host channel too early
	for _, host := range hostArgs {

		ip := net.ParseIP(host) // Check if our host is a valid ip address

		if ip != nil {
			// A valid IP written into the channel
			hosts <- ip.String() // Write to our channel
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
		hosts <- host // Write to our channel
	}
	<-sem // Release lock
}
```

### parseCidr
##### lines 102 - 117

parseCidr is a helper function of [parseHosts](#parseHosts). It's purpose is to generate each hostname under the subnetmask passed in. 
It takes 4 parameters:
- cidrString string
  - A string representation of our host and subnet for example "192.168.0.0/24"
- c chan<- string
  -  this is our host channel passed in as write only. 
- sem chan struct{}
  - Our semaphore lock
- cidrWg *sync.WaitGroup
  - This is a special waitgroup created just for this function to avoid closing our host channel too soon. 
```go
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
```

### sendTcp
##### lines 153 - 186
sendTcp is used to initiate a tcp connection to hostname, for each host we send a request on every port in the port list and check if we get a succesful connection.
sendTcp takes 3 parameters: 
- host string
  - This is our hostname we are about to initiate a connection with
- ports []string
  - Our list of ports we will be trying.
- sem chan struct{}
  - Our semaphore lock.
  
```go
func sendTcp(host string, ports []string, sem chan struct{}) {
	defer wg.Done() // Deduct WaitGrop

	// Here we iterate over each port in our port list
	for _, port := range ports { 
		wg.Add(1) // Add to waitgroup before anon function
		sem <- struct{}{}  // Add to semaphore

		// For every port a goroutine is created to send a tcp message to our host
		// With this we can start multiple go routines and send multiple messages at the same time
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

			<-sem // release semaphore
		}(port, host, sem)
	}
	<-sem
}
```
### addBit
##### lines 85 - 92
addBit is a helper function of [parseCidr](#parseCidr) which simply adds a single bit to our ip address when generating each hostname under the subnet mask. Takes a single parameter: the IP address in question
```go
func addBit(ip net.IP) {
	for i := len(ip) - 1; i >= 0; i-- {
		ip[i]++
		if ip[i] > 0 {
			break
		}
	}
}
```
### Variables
#### Globals
- var wg sync.WaitGroup (line 16)
  - This is our main WaitGroup which we use to make sure each goroutine finishes before our program stops
- const MAX = 512 (line 22)
  - This is the maximum amount of go routines allowed at a same time. This number was chosen after playing around with it a little bit and observing that if it was any lower than 128 my scanner would be a bit slower than we would like and anything above 512 seemed to not make any difference.

#### Notable Variables
- hosts chan string
  - Our main channel which we write our hosts into and
- portList []string
  - Where we write our ports into
- sem chan struct{}
  - The main semaphore lock in our program
</p>
