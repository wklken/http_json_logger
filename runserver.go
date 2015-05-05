package main

import (
	"encoding/json"
	"fmt"
	"github.com/astaxie/beego/config"
	"github.com/gorilla/mux"
	"http_json_logger/logs"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strings"
	"time"
)

var loggers = make(map[string]*logs.JsonLogger)

// 处理json上报
func JsonInputHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	platform := params["platform"]
	doc_type := params["doc_type"]

	// 1. is platform legal?
	is_valid_platform := VALID_PLATFORMS[platform]
	if !is_valid_platform {
		http.Error(w, "Invalid platform", http.StatusBadRequest)
		return
	}

	// 2. is doc_type legal?
	is_valid_doc_type := VALID_DOC_TYPES[platform][doc_type]
	if !is_valid_doc_type {
		http.Error(w, "Invalid doc type", http.StatusBadRequest)
		return
	}

	// 3. get json string from request
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		// panic("error")
		http.Error(w, "Read request Body Error", http.StatusBadRequest)
		return
	}

	// 4. judge if is json body
	d := json.NewDecoder(strings.NewReader(string(body)))

	// default num type in golang is float64, will trans to 1.2423434e+07, `UseNumber` make sure use Number
	d.UseNumber()
	var data interface{}
	if err := d.Decode(&data); err != nil {
		http.Error(w, "Wrong Json Body", http.StatusBadRequest)
		return
	}

	// var data interface{}
	// err = json.Unmarshal([]byte(body), &data)
	// if err != nil {
	// http.Error(w, "Wrong Json Body", http.StatusBadRequest)
	// return
	// }
	// log.WriteJson(string(body))

	// 5. write to log
	logger_key := fmt.Sprintf("%s.%s", platform, doc_type)
	log, ok := loggers[logger_key]
	if ok {
		// add timestamp to it
		now := time.Now()
		data.(map[string]interface{})["ts"] = now.Unix()

		// force to be the valid value in url
		data.(map[string]interface{})["doctype"] = doc_type
		data.(map[string]interface{})["platform"] = platform

		fmt.Println(data)

		json_str, _ := json.Marshal(data)
		log.WriteJson(json_str)
		w.WriteHeader(204)
	} else {
		http.Error(w, "Unknow doc_type, need to add to whitelist and restart collect", http.StatusNotImplemented)
		return
	}

}

// ================================  read config begin ===================================

func get_command_args() []string {
	args := os.Args
	if len(args) != 2 {
		fmt.Println("WRONG PARAMS")
		os.Exit(1)
	}
	return args
}

func read_ini() config.ConfigContainer {
	args := get_command_args()
	ini_config, err := config.NewConfig("ini", args[1])
	if err != nil {
		fmt.Println("ERROR: read config failed!")
		os.Exit(1)
	}
	return ini_config
}

func register_doc_type_log(log_data_path string, platform string, doc_type string) {

	uid := fmt.Sprintf("%s.%s", platform, doc_type)
	file_name := fmt.Sprintf("%s.log", doc_type)

	file_dir := path.Join(log_data_path, platform)
	if _, err := os.Stat(file_dir); os.IsNotExist(err) {
		fmt.Printf("no such file or directory: %s", file_dir)
		os.Mkdir(file_dir, 0755)
		return
	}

	file_path := path.Join(file_dir, file_name)

	log := logs.NewLogger(100, fmt.Sprintf(`{"filename":"%s"}`, file_path))
	loggers[uid] = log

}

func read_valid_configs(ini_config config.ConfigContainer) (map[string]bool, map[string]map[string]bool) {

	platforms := ini_config.Strings("platforms::list")

	fmt.Println("Register Platforms:", platforms)

	valid_platforms := make(map[string]bool)
	valid_doc_types := make(map[string]map[string]bool)

	log_data_path := ini_config.String("log_data_path")

	for _, platform := range platforms {
		// 1. read platform
		valid_platforms[platform] = true

		// 2. read doc type
		sections_attr_name := fmt.Sprintf("platform.%s::list", platform)
		doc_types := ini_config.Strings(sections_attr_name)
		fmt.Println("Register DocTypes:", platform, doc_types)

		valid_platform_doc_types := make(map[string]bool)
		for _, doc_type := range doc_types {

			if len(doc_type) < 1 {
				continue
			}

			valid_platform_doc_types[doc_type] = true
			register_doc_type_log(log_data_path, platform, doc_type)

		}
		valid_doc_types[platform] = valid_platform_doc_types
	}

	return valid_platforms, valid_doc_types
}

var VALID_PLATFORMS map[string]bool
var VALID_DOC_TYPES map[string]map[string]bool

// ================================  read config end ===================================

func main() {

	r := mux.NewRouter()
	r.HandleFunc("/collect/{platform:[a-z]+}/{doc_type:[a-zA-Z0-9]+}", JsonInputHandler).Methods("POST")
	http.Handle("/", r)

	var ini_config = read_ini()
	// init global vars
	VALID_PLATFORMS, VALID_DOC_TYPES = read_valid_configs(ini_config)

	// bind := "127.0.0.1:6400"
	bind := ini_config.String("bind")
	fmt.Println("Bind to host:", bind)

	err := http.ListenAndServe(bind, nil) //设置监听的端口
	if err != nil {
		fmt.Println("ListenAndServe: ", err)
		os.Exit(1)
	} else {
		fmt.Println("ListenAndServe: ", bind)
	}

}
