package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/opesun/goquery"
)

type element struct {
	TagName string
	Count   int32
}

type meta struct {
	Status  uint16
	Headers map[string]string
}

type page struct {
	URL      string
	Meta     meta
	Elements []element
}

type parseChan struct {
	url chan string
	res chan page
	err chan error
}

var (
	//THREADS - the number of threads
	THREADS = flag.Int("tr", 4, "number of threads")
	//URLFILE - the pass to the list of URLs
	URLFILE = flag.String("URL", "input.txt", "the pass to the list of URLs")
	//Preparing channel
	parseURL = make(chan string)
	parseRes = make(chan page)
	parseErr = make(chan error)
	parseCh  = parseChan{
		url: parseURL,
		res: parseRes,
		err: parseErr,
	}
)

func readURLs() ([]string, error) {
	fmt.Println("Opening file with URLs...")
	urlFile, err := os.OpenFile(*URLFILE, os.O_RDONLY, 0666)
	if err != nil {
		return nil, err
	}
	defer urlFile.Close()
	fmt.Println("The file has been successfully opened")

	//counting the number of URLs in the file
	reader := bufio.NewReader(urlFile)
	urlNumber := 0
	for {
		str, err := reader.ReadString('\n')
		if (err == io.EOF && str == "") || str == "\n" {
			break
		} else {
			if err != nil && err != io.EOF {
				return nil, err
			}
		}
		urlNumber++
	}
	//setting the offset for the next read to the begining of the file
	if _, err := urlFile.Seek(0, 0); err != nil {
		return nil, err
	}
	listOfURLs := make([]string, 0, urlNumber)
	fmt.Println("Reading URLs...")
	for {
		str, err := reader.ReadString('\n')
		if (err == io.EOF && str == "") || str == "\n" {
			break
		} else {
			if err != nil && err != io.EOF {
				return nil, err
			}
		}
		listOfURLs = append(listOfURLs, str)
	}
	fmt.Println("URLs have been successfully read")
	return listOfURLs, nil
}

func parse() {
	for url := range parseCh.url {
		x, err := goquery.ParseUrl(url)
	}
}

func main() {
	flag.Parse()
	listOfURLs, err := readURLs()
	if err != nil {
		panic(err)
	}
	//Creating THREADS number of goroutines for parsing
	go func() {
		for i := 0; i < *THREADS; i++ {
			go parse()
		}
	}()
	//
	go func(listOfURLs []string) {
		for _, str := range listOfURLs {
			parseCh.url <- str
		}
	}(listOfURLs)

}
