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
	Response_type string     `json:"response_type"`
	Text          string     `json:"text"`
	Attachments   []AContent `json:"attachments"`
}

type AContent struct {
	Color  string     `json:"color"`
	Title  string     `json:"title"`
	Text   string     `json:"text,omitempty"`
	Fields []FContent `json:"fields,omitempty"`
}

type FContent struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short string `json:"short,omitempty"`
}

func main() {
	addr := ":" + os.Getenv("PORT")
	http.HandleFunc("/", handle)
	log.Fatal(http.ListenAndServe(addr, nil))
}

func make_short_field(title string, value string) FContent {
	return FContent{
		Title: title,
		Value: value,
		Short: "true",
	}
}

func handle(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Error parsing form.", http.StatusBadRequest)
		return
	}

	return_style := "in_channel"

	command := r.Form.Get("command")
	// If command is blank or whatever just use default, which should be /quote, make it public
	// If command is /pquote then make it private
	if command == "/pquote" {
		return_style = "ephemeral"
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
		fmt.Fprintf(w, "Format options can be: fields\n")
		return
	}

	args := strings.Split(arg_text, " ")
	if len(args) < 1 {
		fmt.Fprintf(w, "No arguments given, try help\n")
		return
	}
	symbols := args[0]
	var formats string
	if len(args) > 1 {
		formats = args[1]
	}

	use_fields := false
	for _, option := range strings.Split(formats, ",") {
		if option == "fields" {
			use_fields = true
		}
	}

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

	message := Message{
		Response_type: return_style,
		Text:          "Yahoo Finance says:",
		Attachments:   []AContent{},
	}

	for _, data := range alldata {

		// sn
		symbol := data[0]
		name := data[1]

		// baopl1d1
		//		bid, _ := strconv.ParseFloat(data[2], 64)
		//		ask, _ := strconv.ParseFloat(data[3], 64)
		//		open, _ := strconv.ParseFloat(data[4], 64)
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

		var output string
		var field_array []FContent

		if use_fields {
			field_array = append(field_array, make_short_field("Day", fmt.Sprintf("%.2f - %.2f", day_low, day_high)))
			field_array = append(field_array, make_short_field("52-week", year_range))
			field_array = append(field_array, make_short_field("MktCap", mcap))
			field_array = append(field_array, make_short_field("Vol", vol))
			field_array = append(field_array, make_short_field("Date", date))
		} else {
			output = fmt.Sprintf("Day: %.2f - %.2f,\t52-week: %s\nMktCap: %s, Vol: %s, Date: %s\n",
				day_low, day_high, year_range, mcap, vol, date)
		}

		color := "good"
		if change < 0 {
			color = "danger"
		}

		my_attach_content := AContent{
			Title:  fmt.Sprintf("%s (%s) %.2f\t%+.2f (%+.2f%%)", name, symbol, last, change, change_pct),
			Color:  color,
			Text:   output,
			Fields: field_array,
		}

		message.Attachments = append(message.Attachments, my_attach_content)

	}

	js, err := json.Marshal(message)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)

}
