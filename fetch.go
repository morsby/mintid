package mintid

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"time"

	"github.com/gocolly/colly"
)

type Person struct {
	FirstName string
	LastName  string
	ansID     string
	inst      string
	tjnr      string
	personID  int
	collector *colly.Collector
}

// Shift contains information about a work day
type Shift struct {
	Start time.Time
	End   time.Time
	Label string
}

// tmp struct containing personal information
type tmpPersonalInfo struct {
	Response struct {
		FirstName  string `json:"fornavn"`
		LastName   string `json:"efternavn"`
		Employment []struct {
			AnsID string `json:"ansId"`
			Tjnr  string `json:"tjnr"`
			Inst  string `json:"instKode"`
		} `json:"ansaettelsesforholdList"`
	} `json:"response"`
}

// tmp struct containing other information ...
type tmpUserInfo struct {
	Username           string `json:"UserName"`
	Departmentrelation string `json:"DepartmentRelation"`
	Loggedon           bool   `json:"LoggedOn"`
	Personid           int    `json:"PersonId"`
}

func Login(username, pwd string) (Person, error) {
	var person Person

	// create a new collector
	person.collector = colly.NewCollector(
		colly.AllowURLRevisit(),
		colly.UserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:88.0) Gecko/20100101 Firefox/88.0"),
	)

	// authenticate
	err := person.collector.Post("https://medarbejdernet.dk/eai/SDLogin", map[string]string{
		"login-form-type": "pwd",
		"amtype":          "basic",
		"loginSelect":     "/medarbejdernet/redirectuser",
		"channel":         "medarbejdernet",
		"username":        username,
		"password":        pwd})
	if err != nil {
		return Person{}, err
	}

	// start scraping
	//person.Collector.Visit("https://medarbejdernet.dk/medarbejdernet/html/protected/portalHome.html?isRedirect=true")

	// Get personal info
	person.collector.OnResponse(func(r *colly.Response) {
		// Save personal info:
		if r.Request.URL.String() == "https://medarbejdernet.dk/medarbejdernet/ajax?sid=gel" {
			tmpInfo := tmpPersonalInfo{}
			json.Unmarshal(r.Body, &tmpInfo)

			person.ansID = tmpInfo.Response.Employment[0].AnsID
			person.tjnr = tmpInfo.Response.Employment[0].Tjnr
			person.inst = tmpInfo.Response.Employment[0].Inst
			person.FirstName = tmpInfo.Response.FirstName
			person.LastName = tmpInfo.Response.LastName
		}

	})
	person.collector.Post("https://medarbejdernet.dk/medarbejdernet/ajax?sid=gel", nil)

	person.collector.Visit("https://medarbejdernet.dk/mintid/Default.aspx?inst=" + person.inst + "&tjnr=" + person.inst + "&ansId=" + person.ansID)

	userInfoRegexp := regexp.MustCompile("var UserInfo = ({.*})")

	// To get proper sessions
	// and find own ID
	person.collector.OnResponse(func(r *colly.Response) {
		if r.Request.URL.String() == "https://medarbejdernet.dk/mintid/Default.aspx?menu=function&tab=groupEmployee" {
			var userinfo tmpUserInfo
			userinfoRaw := userInfoRegexp.FindSubmatch(r.Body)[1]
			json.Unmarshal(userinfoRaw, &userinfo)
			person.personID = userinfo.Personid
		}
	})

	person.collector.Visit("https://medarbejdernet.dk/mintid/Default.aspx?menu=function&tab=groupEmployee")

	/*person.Collector.Post("https://medarbejdernet.dk/mintid/Modules/RosterAjax.aspx?menu=function&tab=groupEmployee&task=config", map[string]string{
		"object": `{"Dimensions": [{"PlanId": 1, "Kind": 6, "Clicks": 0}, {"PlanId": 1, "Kind": 0, "Clicks": 0}], "PersonIds": ["115561"], "Start": {"DT":"202105100000"}, "End": {"DT":"202105170000"}, "Position": {"MenuId": "function", "TabId": "groupEmployee", "Number": 5}}`,
	}) */

	// To set current time
	person.collector.Post("https://medarbejdernet.dk/mintid/Modules/DateAjax.aspx?menu=function&tab=groupEmployee&task=setStart", map[string]string{
		"object": fmt.Sprintf(`{"Start": {"DT":"202105100000"}, "PersonId": "%d", "Position": {"MenuId": "function", "TabId": "groupEmployee", "Number": 3}}`, person.personID),
	})
	return person, nil
}

// tmp struct
type tmpResponse struct {
	Start         string        `json:"Start"`
	End           string        `json:"End"`
	Resultdetails []interface{} `json:"ResultDetails"`
	Holidays      interface{}   `json:"Holidays"`
	Roster        []struct {
		Hidedata interface{} `json:"HideData"`
		Row      []struct {
			Start         string `json:"Start"`
			End           string `json:"End"`
			ID            string `json:"Id"`
			Originalstart string `json:"OriginalStart"`
			Originalend   string `json:"OriginalEnd"`
			Label         string `json:"Label"`
			Code          string `json:"Code"`
			Status        int    `json:"Status"`
			Remark        string `json:"Remark"`
			Dimension     struct {
				Planid int `json:"PlanId"`
				Kind   int `json:"Kind"`
				Clicks int `json:"Clicks"`
			} `json:"Dimension"`
			Personid  int  `json:"PersonId"`
			Forcesave bool `json:"ForceSave"`
			Shiftinfo struct {
				Duties []struct {
					Sort     string `json:"Sort"`
					Sorttext string `json:"SortText"`
					Label    string `json:"Label"`
					Payerid  string `json:"PayerId"`
					Payer    string `json:"Payer"`
					Start    string `json:"Start"`
					End      string `json:"End"`
					Isempty  bool   `json:"IsEmpty"`
				} `json:"Duties"`
				Salarysort string `json:"SalarySort"`
				Sorttext   string `json:"SortText"`
				Category   string `json:"Category"`
				Payer      string `json:"Payer"`
			} `json:"ShiftInfo,omitempty"`
			Lasttrans int  `json:"Lasttrans"`
			Present   bool `json:"Present"`
			Color     struct {
				R int `json:"R"`
				G int `json:"G"`
				B int `json:"B"`
			} `json:"Color"`
			Isworkplan bool `json:"IsWorkPlan"`
		} `json:"Row"`
		Personid  int         `json:"PersonId"`
		Dimension interface{} `json:"Dimension"`
	} `json:"Roster"`
	Errorcode int    `json:"ErrorCode"`
	Text      string `json:"Text"`
	Success   bool   `json:"Success"`
	Elapsed   int    `json:"Elapsed"`
}

func (person Person) Fetch(starttime, endtime string) ([]Shift, error) {

	pdfDateRegexp := regexp.MustCompile(`new PdcDate\(([0-9]+),([0-9]+),([0-9]+),([0-9]+),([0-9]+)\)`)

	var shifts []Shift

	person.collector.OnResponse(func(r *colly.Response) {
		// read schedule response
		var tmp tmpResponse
		res := bytes.TrimPrefix(r.Body, []byte("\xef\xbb\xbf"))
		res = pdfDateRegexp.ReplaceAll(res, []byte("\"$1-$2-$3 $4:$5\""))
		err := json.Unmarshal([]byte(res), &tmp)
		if err != nil {
			panic(err)
		}

		loc, err := time.LoadLocation("Europe/Copenhagen")
		if err != nil {
			panic("Cannot parse timezone")
		}
		// iterate Roster -> Row
		for _, rost := range tmp.Roster {
			for _, row := range rost.Row {
				// not a real shift?
				if row.Shiftinfo.Salarysort == "NUL" {
					continue
				}

				start, err := time.ParseInLocation("2006-01-02 15:04", row.Start, loc)
				if err != nil {
					panic(err)
				}
				end, err := time.ParseInLocation("2006-01-02 15:04", row.End, loc)
				if err != nil {
					panic(err)
				}
				shifts = append(shifts, Shift{Start: start, End: end, Label: row.Label})
			}
		}
	})

	person.collector.Post("https://medarbejdernet.dk/mintid/Modules/RosterAjax.aspx?menu=function&tab=groupEmployee&task=get", map[string]string{
		"object": "{\"Start\": {\"DT\":\"" + starttime + "\"}, \"End\": {\"DT\":\"" + endtime + "\"}, \"Dimensions\": [{\"PlanId\": 1, \"Kind\": 6, \"Clicks\": 0}, {\"PlanId\": 1, \"Kind\": 0, \"Clicks\": 0}], \"PersonIds\": [115561], \"Position\": {\"MenuId\": \"function\", \"TabId\": \"groupEmployee\", \"Number\": 5}}",
	})

	return shifts, nil
}
