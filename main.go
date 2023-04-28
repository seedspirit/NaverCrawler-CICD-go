package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/chromedp"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

type subwayData struct {
	lineNum    string `json:"lineNum"`
	weekTag    string `json:"weekTag"`
	stationNm  string `json:"stationNm"`
	inOutTag   string `json:"inOutTag"`
	arriveTime string `json:"arriveTime"`
}

var data []subwayData

// 최종적으로 정보를 저장할 파일 이름(년_월_일_timetable_호선이름.json)을 만드는 함수
func makingFinalFileName(lineNum string) string {
	loc, err := time.LoadLocation("Asia/Seoul")
	checkErr(err)
	now := time.Now()
	t := now.In(loc)
	fileTime := t.Format("2006_01_02")
	finalFileName := fileTime + "_timetable_" + lineNum + ".json"

	return finalFileName
}

// 파일 안 json 데이터를 파싱하여 golang 자료 구조에 맞게 변환
func readStationINFO() map[string][]map[string]interface{} {
	startFileName := "subway_information.json"
	byteValue, _ := os.ReadFile(startFileName)
	var INFO = map[string][]map[string]interface{}{}
	err := json.Unmarshal(byteValue, &INFO)
	if err != nil {
		panic(err)
	}
	return INFO
}

// target인 호선과 그 호선의 naverCode, stationNm으로 이루어진 slice 반환하는 함수
func extractTargetLine(INFO map[string][]map[string]interface{}) (string, []map[string]interface{}) {
	// INFO에서 key(호선 명) 뽑아내기
	keys := make([]string, len(INFO))
	i := 0
	for k := range INFO {
		keys[i] = k
		i++
	}

	lineNum := keys[0]
	info, _ := INFO[lineNum]

	return lineNum, info
}

// 크롤링을 위한 chromdp 인스턴스 생성하기 -> 타깃 페이지로 접근하기 (getPageHTML) -> 크롤링해서 정보 저장하기 (crawler)
func runCrawler(URL string, lineNum string, stationNm string) {
	// settings for crawling

	// create chrome instance
	ctx, cancel := chromedp.NewContext(
		context.Background(),
		chromedp.WithDebugf(log.Printf),
	)
	defer cancel()

	// 크롤링 대상 페이지 가져오기
	htmlContent, _ := getPageHTML(ctx, URL)

	// 페이지 소스 크롤링 & 필요한 정보 추출
	crawler(htmlContent, lineNum, stationNm)
}

// 네이버 검색창에 쿼리 날리기 & 클릭하여 시간표 진입한 후 그 페이지의 HTML 정보 가져오는 함수
func getPageHTML(ctx context.Context, URL string) (string, error) {
	var htmlContent string
	time.Sleep(time.Second * 4)
	err := chromedp.Run(ctx,
		chromedp.Navigate(URL),
		// 클릭해야 할 부분이 나올때까지 기다리기
		chromedp.WaitVisible(".app"),
		chromedp.Click("body > div.app > div > div > div > div.end_section.station_info_section > div.at_end.sofzqce > div.collapse_group_wrap.contents_collapse > div:nth-child(1)"),
		chromedp.OuterHTML("html", &htmlContent, chromedp.ByQuery))

	checkErr(err)
	return htmlContent, nil
}

// 타깃 페이지 HTML에서 필요한 정보만 추출 후 정리하기
func crawler(htmlContent string, lineNum string, stationNm string) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	checkErr(err)

	// 정보를 저장할 struct 인스턴스 생성 후 lineNum, stationNm 정보 기록
	d := subwayData{}
	d.lineNum = lineNum
	d.stationNm = stationNm

	// weektag 알아내기 & d에 기록
	weekTags := doc.Find(".item_day")
	tag, _ := weekTags.Attr("\"aria-selected\" : \"true\"")
	switch tag {
	case "평일":
		d.weekTag = "1"
	case "토요일":
		d.weekTag = "2"
	case "공휴일":
		d.weekTag = "3"
	}

	// inoutTag와 arriveTime 알아내서 d에 기록
	doc.Find(".table_schedule > tbody > tr").Each(func(i int, tr *goquery.Selection) {
		tr.Find("td").Each(func(j int, td *goquery.Selection) {
			// inOutTag 정보 기록
			switch j {
			case 0:
				d.inOutTag = "1" // 1: 상행
			case 1:
				d.inOutTag = "2" // 2: 하행
			}

			// arriveTime 정보 기록
			tmp := td.Find(".inner_timeline > .wrap_time > .time")
			if tmp != nil {
				d.arriveTime = tmp.Text() + ":00"
			}
		})
	})
	data = append(data, d)
	fmt.Println(lineNum, "호선 ", stationNm, " - 입력 완료")

}

// struct를 json 형태로 변환 후 makingFileName에서 나온 이름으로 파일 쓰기
func writeFile(fileName string, data []subwayData) {
	content, err := json.Marshal(data)
	if err != nil {
		log.Fatalln("JSON marshaling failed: %s", err)
	}
	_ = os.WriteFile(fileName, content, 0644)
}

// 에러 체킹용 함수
func checkErr(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}

func main() {
	// subway_information.json 파일 읽고, 안의 내용 파싱
	INFO := readStationINFO()

	// target인 호선과 그 호선의 naverCode, stationNm으로 이루어진 slice 반환
	lineNum, info := extractTargetLine(INFO)
	fmt.Println("타겟 라인 : ", lineNum)

	// 각 역의 정보를 바탕으로 크롤링 시작
	var baseURL string = "https://pts.map.naver.com/end-subway/ends/web/"
	for _, val := range info {
		// val안의 값들은 interface이기 떄문에 type assertion 필요
		naverCode := int(val["naverCode"].(float64)) // 왜인지 모르겠지만 처음 파일에서 interface로 값 가져올 때 float64로 가져와짐
		stationNm := val["stationNm"].(string)
		fmt.Println("네이버 코드 : ", naverCode, "역 이름 : ", stationNm, " 크롤링 시작")
		URL := baseURL + strconv.Itoa(naverCode) + "/home"
		runCrawler(URL, lineNum, stationNm)
	}

	// 정리한 정보를 json 파일 형식으로 저장 ("년_월_일_timetable_호선이름.json")
	finalFilename := makingFinalFileName(lineNum)
	writeFile(finalFilename, data)
}
