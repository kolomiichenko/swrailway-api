package swrailway

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type (
	obj map[string]interface{}

	stationStruct struct {
		ID    string `json:"id"`
		Info  string `json:"info"`
		Label string `json:"label"`
	}
	sheduleStruct struct {
		ID     string // td[0]
		Period string // td[1]
		Route  string // td[2]

		ArrivalFrom        string // td[3]
		ArrivalStationFrom string // td[4]
		DepartureFrom      string // td[5]

		ArrivalTo        string // td[6]
		ArrivalStationTo string // td[7]
		DepartureTo      string // td[8]

		TimeInTrip string // td[9]
		Distance   string // td[10]
		ActiveFrom string // td[11]
		ActiveTo   string // td[12]
	}
)

func GetStation(id, lang string) stationStruct {
	return apiGetStation(obj{
		"JSON": "station",
		"lng":  lang,
		"id":   id,
	})
}

func GetStations(name, lang string) []stationStruct {
	return apiGetStations(obj{
		"JSON": "station",
		"lng":  lang,
		"term": name,
	})
}

func GetShedule(date, lang, from, to string, onlyRemaining bool) []sheduleStruct {

	dateR := "0"
	if onlyRemaining {
		dateR = "1"
	}

	ret := parseShedule(apiGetShedule(obj{
		"sid1":         from,
		"sid2":         to,
		"startPicker2": date,
		"dateR":        dateR, // 0 - all days, 1 - today
		"lng":          lang,  // _ru, _ua, _en
	}))

	if onlyRemaining {
		ret = removeMissed(ret)
	}

	return ret
}

func removeMissed(shedule []sheduleStruct) []sheduleStruct {
	var ret []sheduleStruct

	loc, err := time.LoadLocation("Europe/Kiev")
	if err != nil {
		panic(err)
	}

	hoursNow := time.Now().In(loc).Format("15")
	minutesNow := time.Now().In(loc).Format("04")

	for _, s := range shedule {

		splitedDepartureFrom := strings.Split(s.DepartureFrom, ":")

		if hoursNow < splitedDepartureFrom[0] ||
			(hoursNow == splitedDepartureFrom[0] && minutesNow < splitedDepartureFrom[1]) {

			ret = append(ret, s)
		}
	}

	return ret
}

func parseShedule(body []byte) []sheduleStruct {

	var ret []sheduleStruct

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		log.Fatal(err)
	}
	selector := "#geo2g > tbody"
	doc.Find(selector).Each(func(i int, s *goquery.Selection) {
		s.Find("tr:not(.pix)").Each(func(j int, row *goquery.Selection) {

			if j > 2 {
				var tmp sheduleStruct

				row.Find("td").Each(func(k int, col *goquery.Selection) {

					switch k {
					case 0:
						tmp.ID = strings.TrimSpace(col.Text())
					case 1:
						tmp.Period = strings.TrimSpace(col.Text())
					case 2:
						tmp.Route = strings.TrimSpace(col.Text())
					case 3:
						tmp.ArrivalFrom = strings.TrimSpace(col.Text())
					case 4:
						tmp.ArrivalStationFrom = strings.TrimSpace(col.Text())
					case 5:
						tmp.DepartureFrom = strings.TrimSpace(col.Text())
					case 6:
						tmp.ArrivalTo = strings.TrimSpace(col.Text())
					case 7:
						tmp.ArrivalStationTo = strings.TrimSpace(col.Text())
					case 8:
						tmp.DepartureTo = strings.TrimSpace(col.Text())
					case 9:
						tmp.TimeInTrip = strings.TrimSpace(col.Text())
					case 10:
						tmp.Distance = strings.TrimSpace(col.Text())
					case 11:
						tmp.ActiveFrom = strings.TrimSpace(col.Text())
					case 12:
						tmp.ActiveTo = strings.TrimSpace(col.Text())
					}

				})

				if tmp.ID != "" {
					ret = append(ret, tmp)
				}
			}
		})
	})

	return ret
}

// http://swrailway.gov.ua/timetable/eltrain/?JSON=station&lng=_ua&id=88
func apiGetStation(request obj) stationStruct {
	if request["lng"] == "_ua" {
		request["lng"] = ""
	}
	var result stationStruct
	ret := apiRequest("GET", request)
	json.Unmarshal(ret, &result)
	return result
}

// http://swrailway.gov.ua/timetable/eltrain/?JSON=station&lng=_ua&term=%D0%A1%D0%B2%D1%8F
func apiGetStations(request obj) []stationStruct {
	if request["lng"] == "_ua" {
		request["lng"] = ""
	}
	var result []stationStruct
	ret := apiRequest("GET", request)
	json.Unmarshal(ret, &result)
	return result
}

// http://swrailway.gov.ua/timetable/eltrain/?sid1=85&sid2=88&startPicker2=2018-07-18&dateR=0&lng=_ua
func apiGetShedule(request obj) []byte {
	return apiRequest("GET", request)
}

func apiRequest(httpMethod string, request obj) []byte {

	var requestStr string
	for k, v := range request {
		requestStr = requestStr + fmt.Sprintf("&%v=%v", k, v)
	}
	if len(requestStr) > 0 {
		requestStr = "?" + requestStr[1:len(requestStr)]
	}

	reqURL := "https://swrailway.gov.ua/timetable/eltrain/" + requestStr

	req, err := http.NewRequest(httpMethod, reqURL, bytes.NewBuffer([]byte{}))

	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/106.0.0.0 Safari/537.36")

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	client := &http.Client{
		Transport: tr,
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Println(err.Error())
		return []byte{}
	}
	defer resp.Body.Close()
	result, _ := ioutil.ReadAll(resp.Body)

	if resp.StatusCode >= 300 {
		log.Println(string(result))
	}

	return result
}
