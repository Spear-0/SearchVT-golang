package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Api_KEY      string `yaml:"api_key"`
	Search_API   string `yaml:"search_api"`
	Download_API string `yaml:"download_api"`
	Limit        int    `yaml:"limit"`
	Proxy        string `yaml:"proxy"`
}
type Meta struct {
	Cursor         string   `json:"cursor"`
	Total_hits     int      `json:"total_hits"`
	Allowed_orders []string `json:"allowed_orders"`
	Days_back      int      `json:"days_back"`
}
type Data struct {
	Type string `json:"type"`
	Id   string `json:"id"`
}
type Links struct {
	Self string `json:"self"`
	Next string `json:"next"`
}

type ResultCollects struct {
	Meta  Meta
	Data  []Data
	Links Links
}

type Bar struct {
	Percent int
	Cur     int
	Total   int
	Rate    string
	Graph   string
}

func (bar *Bar) NewOption(start int, total int) {
	bar.Cur = start
	bar.Total = total
	if bar.Graph == "" {
		bar.Graph = "#"
	}
	bar.Percent = bar.getPercent()
	for i := 0; i < int(bar.Percent); i += 2 {
		bar.Rate += bar.Graph
	}
}
func (bar *Bar) getPercent() int {
	return int(float32(bar.Cur) / float32(bar.Total) * 100)
}

func (bar *Bar) NewOptionWithGraph(start, total int, graph string) {
	bar.Graph = graph
	bar.NewOption(start, total)
}

func (bar *Bar) Play(cur int, msg string) {
	bar.Cur = cur
	last := bar.Percent
	bar.Percent = bar.getPercent()
	if bar.Percent != last && bar.Percent%2 == 0 {
		bar.Rate += bar.Graph
	}
	fmt.Printf("\rDoanloading: [%-50s]%3d%%  %8d/%d - %s", bar.Rate, bar.Percent, bar.Cur, bar.Total, msg)
}
func (bar *Bar) Finish() {
	fmt.Println()
}

func main() {
	query := flag.String("q", "engines:acad and p:10+ and fs:2022-09-01T00:00:00+ not type:peexe", "Query stynax")
	isDown := flag.Bool("d", false, "Wether Download Query Result (default false)")
	flag.Parse()

	yamlContent, err := ioutil.ReadFile("config.yaml")
	Checkerr(err, "Can not find file: config.yaml")

	if !isDir("download") {
		log.Fatalln("download is not a directory")
	}
	var config Config
	_ = yaml.Unmarshal(yamlContent, &config)

	Search(fmt.Sprintf(config.Search_API, url.QueryEscape(*query), config.Limit), *isDown, config)

}

func Search(query string, isdown bool, config Config) {
	// http.Client
	uri, err := url.Parse(config.Proxy)
	Checkerr(err, "Can not Parse Proxy in Search")
	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(uri),
		},
	}
	req, _ := http.NewRequest("GET", query, nil)
	req.Header.Set("X-Apikey", config.Api_KEY)
	resp, _ := client.Do(req)
	content, err := ioutil.ReadAll(resp.Body)
	Checkerr(err, "Response Error")
	var res ResultCollects
	err = json.Unmarshal(content, &res)
	Checkerr(err, "Can not marshal json data")

	if isdown {
		Download(res, config)
	}

}
func Download(res ResultCollects, config Config) {
	uri, err := url.Parse(config.Proxy)
	Checkerr(err, "Can not Parse Proxy in Download")
	var bar Bar
	bar.NewOption(0, len(res.Data))
	for k, v := range res.Data {
		client := &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyURL(uri),
			},
		}
		req, err := http.NewRequest("GET", fmt.Sprintf(config.Download_API, v.Id), nil)
		Checkerr(err, "Can not Create HTTP Request")
		req.Header.Set("Accept", "application/json")
		req.Header.Set("X-Apikey", config.Api_KEY)
		resp, err := client.Do(req)
		Checkerr(err, "Get Response Fail")
		resCon, err := ioutil.ReadAll(resp.Body)
		Checkerr(err, "Can not Read Response Body")
		err = ioutil.WriteFile("download/"+v.Id, resCon, 0666)
		Checkerr(err, "Can not Save file: "+v.Id)
		bar.Play(k, v.Id)
	}
	bar.Finish()
}
func Checkerr(err error, errMsg string) {
	if err != nil {
		log.Fatalln(errMsg)
	}
}

func isDir(path string) bool {
	s, err := os.Stat(path)
	Checkerr(err, "Can not find download directory, Please create it!")
	return s.IsDir()
}
