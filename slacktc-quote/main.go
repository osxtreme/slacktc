package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
)

const debug = false

type Message struct {
	Response_type string `json:"response_type"`
	Text          string `json:"text"`
}

func main() {
	addr := ":" + os.Getenv("PORT")
	http.HandleFunc("/", handle)
	log.Fatal(http.ListenAndServe(addr, nil))
}

func handle(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Error parsing form.", http.StatusBadRequest)
		return
	}

	token := r.Form.Get("token")
	my_app_verify_token, valid := os.LookupEnv("SLACK_APP_VERIFY_TOKEN")
	if !valid {
		fmt.Fprintf(w, "No app token found from config\n")
		http.Error(w, "Error: No app token found from config", http.StatusBadRequest)
		return
	}

	if token != my_app_verify_token {
		fmt.Fprintf(w, "App token invalid.\n")
		http.Error(w, "Error: App token invalid.", http.StatusBadRequest)
		return
	}

	if debug {
		fmt.Fprintf(w, "INC: %v %v %v %v\n", r.Method, r.URL, r.Proto, r.Form.Encode())

		//		for name, headers := range r.Header {
		//			fmt.Fprintf(w, "Name: %s Headers: %s\n", name, headers)
		//		}
	}

	// We are trimming angle brackets because slack's new url/user/channel escaping adds them if you set your
	// command to enable "Escape channels, users, and links sent to your app", so if it's a url, trim it
	arg_text := strings.Trim(r.Form.Get("text"), "<>")
	if debug {
		fmt.Fprintf(w, "args: %v\n", arg_text)
	}

	if arg_text == "help" {
		fmt.Fprintf(w, "Usage: /quote symbol1<,symbol2,...> <field1,field2,...>\n")
		fmt.Fprintf(w, "Fields can be: name,cap\n")
		return
	}

	args := strings.Split(arg_text, " ")
	symbols := args[0]
	//format := args[1]

	// Ideas and examples from https://github.com/doneland/yquotes

	base_url := "http://download.finance.yahoo.com/d/quotes.csv"
	default_format := "snbaopl1d1ghwj1v"

	my_url := base_url + "?s=" + symbols + "&f=" + default_format
	resp, err := http.Get(my_url)
	if err != nil {
		fmt.Fprintf(w, err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	reader := csv.NewReader(resp.Body)
	alldata, err := reader.ReadAll()
	if err != nil {
		fmt.Fprintf(w, err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// TODO: clean up date

	output := ""
	for _, data := range alldata {

		// sn
		symbol := data[0]
		name := data[1]

		// baopl1d1
		bid, _ := strconv.ParseFloat(data[2], 64)
		ask, _ := strconv.ParseFloat(data[3], 64)
		open, _ := strconv.ParseFloat(data[4], 64)
		prevClose, _ := strconv.ParseFloat(data[5], 64)
		last, _ := strconv.ParseFloat(data[6], 64)
		date := data[7]

		// ghwj1v
		day_low, _ := strconv.ParseFloat(data[8], 64)
		day_high, _ := strconv.ParseFloat(data[9], 64)
		year_range := data[10]
		mcap := data[11]
		vol := data[12]

		change := last - prevClose
		change_pct := change / prevClose * 100

		output += fmt.Sprintf("Quote: %s (%s) - Last: %.2f %+.2f (%+.2f%%), Day range: %.2f-%.2f 52-week range: %s. MktCap: %s. Vol: %s, Bid/Ask: %.2f/%.2f, Open: %.2f, Date: %s\n",
			name, symbol, last, change, change_pct, day_low, day_high, year_range, mcap, vol, bid, ask, open, date)
	}

	message := Message{"in_channel", output}
	js, err := json.Marshal(message)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)

}
