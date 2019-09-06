package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"golang.org/x/net/html"
)

type meta struct {
	Status  int
	Headers map[string][]string
}

type page struct {
	URL      string
	Meta     meta
	Elements map[string]int
}

type parseChan struct {
	url chan string
	res chan page
	err chan error
}

var (
	threads = flag.Int("tr", 4, "number of parsing threads")
	urlFile = flag.String("url", "input.txt", "the pass to the list of URLs")
	resFile = flag.String("res", "output.txt", "the pass to the output file")
	//Preparing channels
	parseURL = make(chan string)
	parseRes = make(chan page)
	parseErr = make(chan error)
	parseCh  = parseChan{
		url: parseURL,
		res: parseRes,
		err: parseErr,
	}
)

//Reads the list of URLs from urlFile
func readURLs() ([]string, error) {
	fmt.Println("Opening file with URLs...")
	urlFile, err := os.OpenFile(*urlFile, os.O_RDONLY, 0666)
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
	urlList := make([]string, 0, urlNumber)
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
		urlList = append(urlList, str)
	}
	fmt.Println("URLs have been successfully read")
	return urlList, nil
}

//Parse URLs from the list
func urlParser() {
	for url := range parseCh.url {
		resp, err := http.Get(url)
		if err != nil {
			parseCh.err <- err
			continue
		}
		defer resp.Body.Close()
		doc, err := html.Parse(resp.Body)
		if err != nil {
			parseCh.err <- err
			continue
		}

		//Creating a container for the data
		var pg page
		pg.Meta.Headers = make(map[string][]string)
		pg.Elements = make(map[string]int)

		pg.URL = url
		pg.Meta.Status = resp.StatusCode
		//Copying info about HTTP headers
		for k, val := range resp.Header {
			pg.Meta.Headers[k] = val
		}

		//Loocking for HTML tags and counting their amount
		var f func(*html.Node)
		f = func(n *html.Node) {
			if n.Type == html.ElementNode {
				pg.Elements[n.Data] = pg.Elements[n.Data] + 1
			}
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				f(c)
			}
		}
		f(doc)
		parseCh.res <- pg
	}
}

//Collects results from urlParser
func collector(urlNumber int) []page {
	resultList := make([]page, 0, urlNumber)
	for i := 0; i < urlNumber; i++ {
		select {
		case result := <-parseCh.res:
			resultList = append(resultList, result)

		case urlErr := <-parseCh.err:
			fmt.Println("ERROR:")
			fmt.Println(urlErr)
		}
	}
	return resultList
}

//Encode slice of result structures to JSON and writes it to the resFile
func output(result []page) {
	file, err := os.Create(*resFile)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	jsonRes, err := json.MarshalIndent(result, "", "\t")
	if err != nil {
		panic(err)
	}
	_, err = file.Write(jsonRes)
	if err != nil {
		panic(err)
	}
}

func main() {
	//Parsing comand line tags
	flag.Parse()
	fmt.Println("The number of threads = ", *threads)
	fmt.Println("Inpute file = ", *urlFile)
	fmt.Println("Outpute file = ", *resFile)
	urlList, err := readURLs()
	if err != nil {
		panic(err)
	}
	//Creating threads number of goroutines for parsing
	go func() {
		for i := 0; i < *threads; i++ {
			go urlParser()
		}
	}()
	//
	go func(urlList []string) {
		for _, str := range urlList {
			parseCh.url <- strings.TrimSuffix(str, "\n")
		}
		close(parseCh.url)
	}(urlList)
	result := collector(len(urlList))
	output(result)
}
