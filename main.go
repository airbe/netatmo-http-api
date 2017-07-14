package main

import (
	"encoding/json"
	"flag"
	"fmt"
	netatmo "github.com/romainbureau/netatmo-api-go"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"
)

type Config struct {
	ClientId     string `yaml:"client_id"`
	ClientSecret string `yaml:"client_secret"`
	Username     string `yaml:"username"`
	Password     string `yaml:"password"`
	HttpPort     int    `yaml:"http_port"`
	HttpAddr     string `yaml:"http_addr"`
}

var Configuration *Config

func check(e error) {
	if e != nil {
		log.Fatal(e)
	}
}

func init() {
	configFile := flag.String("config", "config.yml", "configuration file")
	flag.Parse()
	file, err := ioutil.ReadFile(*configFile)
	check(err)
	errYaml := yaml.Unmarshal(file, &Configuration)
	check(errYaml)
}

func InternalServerError(w http.ResponseWriter) {
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte("Internal Server Error"))
}

func netatmoHandler(w http.ResponseWriter, r *http.Request) {
	netatmoClient, err := netatmo.NewClient(netatmo.Config{
		ClientID:     Configuration.ClientId,
		ClientSecret: Configuration.ClientSecret,
		Username:     Configuration.Username,
		Password:     Configuration.Password,
	})

	if err != nil {
		log.Println(err.Error())
		InternalServerError(w)
	}

	values, err := netatmoClient.Read()
	if err != nil {
		log.Println(err.Error())
		InternalServerError(w)
	}

	var Values = make(map[string]float32)
	for _, station := range values.Stations() {
		for _, module := range station.Modules() {
			_, data := module.Data()
			moduleName := strings.ToLower(module.ModuleName)
			for key, value := range data {
				metricName := fmt.Sprintf("netatmo.%s.%s", moduleName, key)
				Values[metricName] = value.(float32)
			}
		}
	}
	j, err := json.Marshal(Values)
	if err != nil {
		log.Println(err.Error())
		InternalServerError(w)
	}
	w.Header().Set("Cache-Control", "no-cache, must-revalidate")
	w.Header().Set("Expires", "Mon, 24 Oct 1982 05:00:00 GMT")
	w.Header().Set("Content-type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(j))
}

func main() {
	log.Println(fmt.Sprintf("listening on %d", Configuration.HttpPort))
	http.HandleFunc("/", netatmoHandler)
	s := &http.Server{
		Addr:           fmt.Sprintf("%s:%d", Configuration.HttpAddr, Configuration.HttpPort),
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	log.Fatal(s.ListenAndServe())
}
