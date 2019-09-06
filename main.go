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
	//THREADS - the number of threads
	THREADS = flag.Int("tr", 2, "number of threads")
	//URLFILE - the pass to the list of URLs
	URLFILE = flag.String("url", "input.txt", "the pass to the list of URLs")
	//RESFILE - the pass to the output file
	RESFILE = flag.String("res", "output.txt", "the pass to the output file")
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

		var pg page
		pg.Meta.Headers = make(map[string][]string)
		pg.Elements = make(map[string]int)

		pg.URL = url
		pg.Meta.Status = resp.StatusCode
		for k, val := range resp.Header {
			pg.Meta.Headers[k] = val
		}

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

func collector(urlNumber int) []page {
	resultList := make([]page, 0, urlNumber)
	for i := 0; i < urlNumber; i++ {
		select {
		case result := <-parseCh.res:
			resultList = append(resultList, result)

		case urlErr := <-parseCh.err:
			fmt.Println(urlErr)
		}
	}
	return resultList
}

func output(result []page) {
	file, err := os.Create(*RESFILE)
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
	flag.Parse()
	urlList, err := readURLs()
	if err != nil {
		panic(err)
	}
	//Creating THREADS number of goroutines for parsing
	go func() {
		for i := 0; i < *THREADS; i++ {
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
	fmt.Println(result)
}
